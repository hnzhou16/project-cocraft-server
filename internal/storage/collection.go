package storage

import (
	"context"
	"fmt"
	"github.com/hnzhou16/project-cocraft-server/internal/db"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
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
		GetFeed(ctx context.Context, user *User, cq CursorQuery) ([]PostWithLikeStatus, error)
		GetTrending(ctx context.Context, user *User, cq CursorQuery) ([]PostWithLikeStatus, error)
		GetByID(ctx context.Context, postID string) (*Post, error)
		GetByUserID(ctx context.Context, userID primitive.ObjectID, cq CursorQuery) ([]PostWithLikeStatus, error)
		GetCountByUserID(ctx context.Context, userID primitive.ObjectID) (int, error)
		Search(ctx context.Context, user *User, query string, cq CursorQuery) ([]PostWithLikeStatus, error)
		Update(ctx context.Context, post *Post) error
		ToggleLike(ctx context.Context, userID primitive.ObjectID, post *Post) (bool, error)
		IncrementCommentCount(ctx context.Context, postID primitive.ObjectID) error
		Delete(ctx context.Context, postID string) error
	}

	Comment interface {
		Create(ctx context.Context, c *Comment) (CommentWithParentAndUser, error)
		Exists(context.Context, primitive.ObjectID) (bool, error)
		GetByPostID(ctx context.Context, postID primitive.ObjectID) ([]CommentWithParentAndUser, error)
	}

	Review interface {
		Create(ctx context.Context, review *Review, ratedUser *User) error
		GetByRatedUserID(ctx context.Context, userID primitive.ObjectID) ([]Review, error)
		Delete(ctx context.Context, reviewID string, ratedUser *User) error
	}

	Follow interface {
		GetFollowing(ctx context.Context, followerID primitive.ObjectID) ([]primitive.ObjectID, error)
		GetFollowerCount(ctx context.Context, followeeID primitive.ObjectID) (int, error)
		GetFollowingCount(ctx context.Context, followerID primitive.ObjectID) (int, error)
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
	// initialize TTL index only on invite collection to clean up expired invite
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

func EnsureIndexes(ctx context.Context, c Collection) error {
	//User collection
	_, err := c.User.(*UserStorage).collection.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "email", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys:    bson.D{{Key: "username", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{{Key: "role", Value: 1}}, // for filtering by role
		},
		{
			Keys: bson.D{{Key: "created_at", Value: 1}}, // sorting/filtering by creation date
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create user indexes: %w", err)
	}

	//Post collection
	_, err = c.Post.(*PostStorage).collection.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "user_id", Value: 1}}},     // find posts by user
		{Keys: bson.D{{Key: "user_role", Value: 1}}},   // filter by user role
		{Keys: bson.D{{Key: "created_at", Value: -1}}}, // sorting feed
	})
	if err != nil {
		return fmt.Errorf("failed to create post indexes: %w", err)
	}

	//Comment collection
	_, err = c.Comment.(*CommentStorage).collection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "post_id", Value: 1}}, // get comments by post
	})
	if err != nil {
		return fmt.Errorf("failed to create comment indexes: %w", err)
	}

	//Review collection
	_, err = c.Review.(*ReviewStorage).collection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "rated_user_id", Value: 1}}, // get reviews for a user
	})
	if err != nil {
		return fmt.Errorf("failed to create review indexes: %w", err)
	}

	//Follow collection
	_, err = c.Follow.(*FollowStorage).collection.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "follower_id", Value: 1}}},  // get who a user follows
		{Keys: bson.D{{Key: "following_id", Value: 1}}}, // get followers of a user
	})
	if err != nil {
		return fmt.Errorf("failed to create follow indexes: %w", err)
	}

	return nil
}
