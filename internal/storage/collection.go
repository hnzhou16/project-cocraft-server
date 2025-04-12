package storage

import (
	"context"
	"fmt"
	"github.com/hnzhou16/project-social/internal/db"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"time"
)

var (
	QueryTimeout = 5 * time.Second
)

type Collection struct {
	// any struct satisfies the interface can be User
	User interface {
		Create(ctx context.Context, u *User) error
		CreateAndInvite(ctx context.Context, u *User, token string, inviteExp time.Duration) error
		Activate(ctx context.Context, token string) error
		GetAll(ctx context.Context) (*[]User, error)
		GetByID(ctx context.Context, userID string) (*User, error)
		GetByEmail(ctx context.Context, email string) (*User, error)
		ValidateUsername(ctx context.Context, mentions []string) ([]string, error)
		AddRating(ctx context.Context, user *User, score int) error
		ReduceRating(ctx context.Context, user *User, score int) error
		Delete(ctx context.Context, userID primitive.ObjectID) error
	}

	Post interface {
		Create(ctx context.Context, p *Post) error
		GetFeed(ctx context.Context, user *User, pq PaginationQuery) ([]PostWithLikeStatus, error)
		GetByID(ctx context.Context, postID string) (*Post, error)
		GetByUserID(ctx context.Context, userID primitive.ObjectID, pq PaginationQuery) ([]PostWithLikeStatus, error)
		Update(ctx context.Context, post *Post) error
		ToggleLike(ctx context.Context, userID primitive.ObjectID, post *Post) (bool, error)
		IncrementCommentCount(ctx context.Context, postID primitive.ObjectID) error
		Delete(ctx context.Context, postID string) error
	}

	Comment interface {
		Create(ctx context.Context, c *Comment) error
		Exists(context.Context, primitive.ObjectID) (bool, error)
		GetByPostID(ctx context.Context, postID primitive.ObjectID) ([]CommentWithParentAndUser, error)
	}

	Review interface {
		Create(ctx context.Context, review *Review, ratedUser *User) error
		Delete(ctx context.Context, reviewID string, ratedUser *User) error
	}

	Follow interface {
		GetFollowing(ctx context.Context, followerID primitive.ObjectID) ([]primitive.ObjectID, error)
		IsFollowing(ctx context.Context, followerID, followingID primitive.ObjectID) (bool, error)
		FollowUser(ctx context.Context, followerID, followingID primitive.ObjectID) error
		UnfollowUser(ctx context.Context, followerID, followingID primitive.ObjectID) error
	}

	Invite interface {
		CreateTTLIndex(ctx context.Context)
		Create(ctx context.Context, userID primitive.ObjectID, token string, inviteExp time.Duration) error
		Delete(ctx context.Context, userID primitive.ObjectID) error
	}
}

func NewMongoDBCollections(dbConn *db.DBConnection) Collection {
	userCollection := dbConn.GetCollection("user")
	postCollection := dbConn.GetCollection("post")
	commentCollection := dbConn.GetCollection("comment")
	inviteCollection := dbConn.GetCollection("invite")
	reviewCollection := dbConn.GetCollection("review")
	followCollection := dbConn.GetCollection("follow")

	userStorage := &UserStorage{
		collection:    userCollection,
		inviteStorage: &InviteStorage{collection: inviteCollection},
	}
	postStorage := &PostStorage{
		collection: postCollection,
	}
	commentStorage := &CommentStorage{
		collection:  commentCollection,
		userStorage: &UserStorage{collection: userCollection},
		postStorage: &PostStorage{collection: postCollection},
	}
	inviteStorage := &InviteStorage{
		collection: inviteCollection,
	}
	// initialize TTL index to clean up expired invite
	inviteStorage.CreateTTLIndex(context.Background())

	reviewStorage := &ReviewStorage{
		collection:  reviewCollection,
		userStorage: &UserStorage{collection: userCollection},
	}

	followStorage := &FollowStorage{
		collection: followCollection,
	}

	return Collection{
		User:    userStorage,
		Post:    postStorage,
		Comment: commentStorage,
		Invite:  inviteStorage,
		Review:  reviewStorage,
		Follow:  followStorage,
	}
}

func withTransaction(ctx context.Context, client *mongo.Client, txnFunc func(mongo.SessionContext) (interface{}, error)) error {
	session, err := client.StartSession()
	if err != nil {
		return fmt.Errorf("failed to start session: %w", err)
	}
	defer session.EndSession(ctx)

	_, err = session.WithTransaction(ctx, txnFunc)
	return err
}
