package storage

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/hnzhou16/project-cocraft-server/internal/security"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

var (
	ErrDupUsername  = errors.New("a user with this username already exists")
	ErrDupEmail     = errors.New("a user with this email already exists")
	ErrUserNotFound = errors.New("user not found")
)

type User struct {
	ID        primitive.ObjectID `json:"id,omitempty" bson:"_id,omitempty"` // omit the 0 generated during instantiating
	Username  string             `json:"username" bson:"username"`
	Email     string             `json:"email" bson:"email"`
	Password  string             `json:"-" bson:"password"`
	Role      security.Role      `json:"role" bson:"role"`
	Profile   Profile            `json:"profile,omitempty" bson:"profile,omitempty"`
	Rating    Rating             `json:"rating,omitempty" bson:"rating,omitempty"`
	IsActive  bool               `json:"is_active" bson:"is_active"`
	CreatedAt time.Time          `json:"created_at,omitempty" bson:"created_at,omitempty"`
	UpdatedAt time.Time          `json:"updated_at,omitempty" bson:"updated_at,omitempty"`
}

type Profile struct {
	Bio      string  `json:"bio,omitempty" bson:"bio,omitempty"`
	Location string  `json:"location,omitempty" bson:"location,omitempty"`
	Contact  Contact `json:"contact,omitempty" bson:"contact,omitempty"`
}

type Contact struct {
	Email string `json:"email,omitempty" bson:"email,omitempty"`
	Phone string `json:"phone,omitempty" bson:"phone,omitempty"`
}

type Rating struct {
	TotalRating float32 `json:"total_rating,omitempty" bson:"total_rating,omitempty"`
	RatingCount int     `json:"rating_count,omitempty" bson:"rating_count,omitempty"`
}

type UserStorage struct {
	collection    *mongo.Collection
	inviteStorage *InviteStorage
}

func (u *UserStorage) Create(ctx context.Context, user *User) error {
	user.ID = primitive.NewObjectID()
	user.IsActive = false
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()

	ctxTimeout, cancel := context.WithTimeout(ctx, QueryTimeout)
	defer cancel()

	var existingUser User
	err := u.collection.FindOne(
		ctxTimeout,
		bson.M{"$or": []bson.M{
			{"username": user.Username},
			{"email": user.Email},
		}},
	).Decode(&existingUser)

	if err == nil {
		if existingUser.Email == user.Email {
			return ErrDupEmail
		} else if existingUser.Username == user.Username {
			return ErrDupUsername
		}
	} else if !errors.Is(err, mongo.ErrNoDocuments) {
		return fmt.Errorf("error checking existence of user: %w", err)
	}

	result, err := u.collection.InsertOne(ctxTimeout, bson.M{
		"_id":        user.ID,
		"username":   user.Username,
		"email":      user.Email,
		"password":   user.Password,
		"role":       user.Role,
		"profile":    user.Profile,
		"rating":     user.Rating,
		"is_active":  user.IsActive,
		"created_at": user.CreatedAt,
		"updated_at": user.UpdatedAt,
	})

	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	user.ID = result.InsertedID.(primitive.ObjectID)
	return nil
}

func (u *UserStorage) CreateAndInvite(ctx context.Context, user *User, token string, inviteExp time.Duration) error {
	client := u.collection.Database().Client()
	txnFunc := func(sessCtx mongo.SessionContext) (interface{}, error) {
		if err := u.Create(sessCtx, user); err != nil {
			return nil, fmt.Errorf("failed to create user: %w", err)
		}

		if err := u.inviteStorage.Create(sessCtx, user.ID, token, inviteExp); err != nil {
			return nil, fmt.Errorf("failed to create invite: %w", err)
		}

		return nil, nil
	}
	return withTransaction(ctx, client, txnFunc)
}

func (u *UserStorage) getUserFromInvite(ctx context.Context, token string) (*User, error) {
	hash := sha256.Sum256([]byte(token))
	hashToken := hex.EncodeToString(hash[:])

	ctxTimeout, cancel := context.WithTimeout(ctx, QueryTimeout)
	defer cancel()

	var invite struct {
		UserID string `bson:"user_id"`
	}

	err := u.inviteStorage.collection.FindOne(ctxTimeout, bson.M{"token": hashToken}).Decode(&invite)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrInviteNotFound
		}
		return nil, fmt.Errorf("failed to find invite: %w", err)
	}

	return u.GetByID(ctxTimeout, invite.UserID)
}

func (u *UserStorage) Activate(ctx context.Context, token string) error {
	client := u.collection.Database().Client()
	txnFunc := func(sessCtx mongo.SessionContext) (interface{}, error) {
		// 1. find user with token
		user, err := u.getUserFromInvite(sessCtx, token)
		if err != nil {
			return nil, err
		}

		filter := bson.M{"_id": user.ID}
		update := bson.M{"$set": bson.M{
			"is_active":  true,
			"updated_at": time.Now(),
		}}

		// 2. update 'is_active' property
		_, err = u.collection.UpdateOne(sessCtx, filter, update)
		if err != nil {
			return nil, fmt.Errorf("failed to activate user with id %v: %w", user.ID, err)
		}

		// 3. delete invite
		if err := u.inviteStorage.Delete(sessCtx, user.ID); err != nil {
			return nil, fmt.Errorf("failed to delete user invite: %w", err)
		}

		return nil, nil
	}
	return withTransaction(ctx, client, txnFunc)
}

func (u *UserStorage) GetAll(ctx context.Context) (*[]User, error) {
	ctxTimeout, cancel := context.WithTimeout(ctx, QueryTimeout)
	defer cancel()

	var users []User

	cursor, err := u.collection.Find(ctxTimeout, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("failed to get users: %w", err)
	}
	defer cursor.Close(ctxTimeout)

	if err = cursor.All(ctxTimeout, &users); err != nil {
		return nil, fmt.Errorf("failed to get users: %w", err)
	}

	return &users, nil
}

func (u *UserStorage) GetByID(ctx context.Context, userID string) (*User, error) {
	objID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to convert user id to object id: %w", err)
	}

	ctxTimeout, cancel := context.WithTimeout(ctx, QueryTimeout)
	defer cancel()

	var user User

	err = u.collection.FindOne(ctxTimeout, bson.M{"_id": objID}).Decode(&user)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to find user with ID %v: %w", userID, err)
	}

	return &user, nil
}

func (u *UserStorage) GetByEmail(ctx context.Context, email string) (*User, error) {
	ctxTimeout, cancel := context.WithTimeout(ctx, QueryTimeout)
	defer cancel()

	var user User

	err := u.collection.FindOne(ctxTimeout, bson.M{"email": email}).Decode(&user)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to find user with email %v: %w", email, err)
	}

	return &user, nil
}

func (u *UserStorage) ValidateUsername(ctx context.Context, usernames []string) ([]string, error) {
	var validUsernames []string

	cursor, err := u.collection.Find(ctx, bson.M{
		"username": bson.M{"$in": usernames},
	})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var user struct {
			Username string `json:"username" bson:"username"`
		}
		if err := cursor.Decode(&user); err != nil {
			return nil, err
		}
		validUsernames = append(validUsernames, user.Username)
	}

	return validUsernames, nil
}

func (u *UserStorage) AddRating(ctx context.Context, user *User, score int) error {
	update := bson.M{
		"$inc": bson.M{
			"rating.total_rating": score,
			"rating.rating_count": 1,
		},
	}
	_, err := u.collection.UpdateOne(ctx,
		bson.M{"_id": user.ID},
		update,
	)

	return err
}

func (u *UserStorage) ReduceRating(ctx context.Context, user *User, score int) error {
	update := bson.M{
		"$inc": bson.M{
			"rating.total_rating": -score,
			"rating.rating_count": -1,
		},
	}
	result, err := u.collection.UpdateOne(
		ctx,
		bson.M{"_id": user.ID},
		update,
	)

	if err != nil {
		return fmt.Errorf("failed to update rating: %w", err)
	}

	if result.ModifiedCount == 0 {
		return fmt.Errorf("user rating not updated (user not found or no changes)")
	}

	return nil
}

func (u *UserStorage) Delete(ctx context.Context, userID primitive.ObjectID) error {
	client := u.collection.Database().Client()
	txnFunc := func(sessCtx mongo.SessionContext) (interface{}, error) {
		_, err := u.collection.DeleteOne(sessCtx, bson.M{"_id": userID})
		if err != nil {
			return nil, fmt.Errorf("failed to delete user: %w", err)
		}

		err = u.inviteStorage.Delete(sessCtx, userID)
		if err != nil {
			return nil, fmt.Errorf("failed to delete user invite: %w", err)
		}

		return nil, nil
	}

	return withTransaction(ctx, client, txnFunc)
}
