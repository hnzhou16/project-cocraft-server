package storage

import (
	"context"
	"errors"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type Follow struct {
	ID         primitive.ObjectID `json:"id,omitempty" bson:"_id,omitempty"`
	FollowerID primitive.ObjectID `json:"follower_id" bson:"follower_id"`
	FolloweeID primitive.ObjectID `json:"followee_id" bson:"followee_id"`
	CreatedAt  time.Time          `json:"created_at" bson:"created_at"`
}

type FollowStorage struct {
	collection *mongo.Collection
}

func (f *FollowStorage) GetFollowing(ctx context.Context, followerID primitive.ObjectID) ([]primitive.ObjectID, error) {
	cursor, err := f.collection.Find(ctx, bson.M{"follower_id": followerID})
	if err != nil {
		// only returns error for actual db failures
		return nil, err
	}
	defer cursor.Close(ctx)

	var followeeIDs []primitive.ObjectID

	for cursor.Next(ctx) {
		var follow Follow
		if err := cursor.Decode(&follow); err != nil {
			return nil, err
		}
		followeeIDs = append(followeeIDs, follow.FolloweeID)
	}

	return followeeIDs, cursor.Err()
}

func (f *FollowStorage) GetFollowerCount(ctx context.Context, followeeID primitive.ObjectID) (int, error) {
	ctxTimeout, cancel := context.WithTimeout(ctx, QueryTimeout)
	defer cancel()

	followerCount, err := f.collection.CountDocuments(ctxTimeout, bson.M{"followee_id": followeeID})
	if err != nil {
		return 0, fmt.Errorf("failed to get following user count: %w", err)
	}

	return int(followerCount), nil
}

func (f *FollowStorage) GetFollowingCount(ctx context.Context, followerID primitive.ObjectID) (int, error) {
	ctxTimeout, cancel := context.WithTimeout(ctx, QueryTimeout)
	defer cancel()

	followingCount, err := f.collection.CountDocuments(ctxTimeout, bson.M{"follower_id": followerID})
	if err != nil {
		return 0, fmt.Errorf("failed to get following user count: %w", err)
	}

	return int(followingCount), nil
}

func (f *FollowStorage) IsFollowing(ctx context.Context, followerID, followingID primitive.ObjectID) (bool, error) {
	err := f.collection.FindOne(ctx, bson.M{
		"follower_id": followerID,
		"followee_id": followingID,
	}).Err()

	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

func (f *FollowStorage) FollowUser(ctx context.Context, followerID, followingID primitive.ObjectID) error {
	_, err := f.collection.InsertOne(ctx, bson.M{
		"_id":         primitive.NewObjectID(),
		"follower_id": followerID,
		"followee_id": followingID,
		"created_at":  time.Now(),
	})
	return err
}

func (f *FollowStorage) UnfollowUser(ctx context.Context, followerID, followingID primitive.ObjectID) error {
	_, err := f.collection.DeleteOne(ctx, bson.M{
		"follower_id": followerID,
		"followee_id": followingID,
	})
	return err
}
