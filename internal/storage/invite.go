package storage

import (
	"context"
	"errors"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log"
	"time"
)

var (
	ErrInviteNotFound = errors.New("invite not found")
)

type InviteStorage struct {
	collection *mongo.Collection
}

func (i *InviteStorage) CreateTTLIndex(ctx context.Context) {
	ctxTimeout, cancel := context.WithTimeout(ctx, QueryTimeout)
	defer cancel()

	indexModel := mongo.IndexModel{
		Keys:    bson.M{"expires_at": 1},
		Options: options.Index().SetExpireAfterSeconds(0),
	}

	_, err := i.collection.Indexes().CreateOne(ctxTimeout, indexModel)
	if err != nil {
		log.Fatalf("Failed to create TTL index on invite collection: %v", err)
	}
}

func (i *InviteStorage) Create(ctx context.Context, userID primitive.ObjectID, token string, inviteExp time.Duration) error {
	invite := bson.M{
		"user_id":    userID,
		"token":      token,
		"created_at": time.Now(),
		"expires_at": time.Now().Add(inviteExp),
	}

	ctxTimeout, cancel := context.WithTimeout(ctx, QueryTimeout)
	defer cancel()

	_, err := i.collection.InsertOne(ctxTimeout, invite)
	if err != nil {
		return fmt.Errorf("failed to create invite: %w", err)
	}

	return nil
}

func (i *InviteStorage) Delete(ctx context.Context, userID primitive.ObjectID) error {
	filter := bson.M{"user_id": userID}

	result, err := i.collection.DeleteOne(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to delete user invite with id %v: %w", userID, err)
	}

	if result.DeletedCount == 0 {
		return fmt.Errorf("user id %v: %w", userID, ErrInviteNotFound)
	}

	return nil
}
