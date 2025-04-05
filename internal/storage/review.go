package storage

import (
	"context"
	"errors"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Review struct {
	ID            primitive.ObjectID `json:"id,omitempty" bson:"_id,omitempty"`
	RatedUserID   primitive.ObjectID `json:"rated_user_id" bson:"rated_user_id"`
	RaterID       primitive.ObjectID `json:"rater_id" bson:"rater_id"`
	RaterUsername string             `json:"rater_username" bson:"rater_username"`
	Score         int                `json:"score" bson:"score"`
	Comment       string             `json:"comment" bson:"comment"`
	CreatedAt     time.Time          `json:"created_at" bson:"created_at"`
}

type ReviewStorage struct {
	collection  *mongo.Collection
	userStorage *UserStorage
}

func (r *ReviewStorage) Create(ctx context.Context, review *Review, ratedUser *User) error {
	client := r.collection.Database().Client()

	txnFunc := func(sessCtx mongo.SessionContext) (interface{}, error) {
		review.ID = primitive.NewObjectID()
		review.CreatedAt = time.Now()

		ctxTimeout, cancel := context.WithTimeout(ctx, QueryTimeout)
		defer cancel()

		_, err := r.collection.InsertOne(ctxTimeout, bson.M{
			"_id":            review.ID,
			"rated_user_id":  review.RatedUserID,
			"rater_id":       review.RaterID,
			"rater_username": review.RaterUsername,
			"score":          review.Score,
			"comment":        review.Comment,
			"created_at":     review.CreatedAt,
		})

		if err != nil {
			return nil, fmt.Errorf("failed to create review: %w", err)
		}

		if err := r.userStorage.AddRating(ctx, ratedUser, review.Score); err != nil {
			return nil, fmt.Errorf("failed to calculate rating: %w", err)
		}

		return review, nil
	}

	return withTransaction(ctx, client, txnFunc)
}

func (r *ReviewStorage) Delete(ctx context.Context, reviewID string, ratedUser *User) error {
	client := r.collection.Database().Client()

	txnFunc := func(sessCtx mongo.SessionContext) (interface{}, error) {
		objID, err := primitive.ObjectIDFromHex(reviewID)
		if err != nil {
			return nil, fmt.Errorf("failed to convert reviewID to ObjectID: %w", err)
		}

		ctx, cancel := context.WithTimeout(ctx, QueryTimeout)
		defer cancel()

		var review Review

		err = r.collection.FindOneAndDelete(ctx, bson.M{"_id": objID}).Decode(&review)
		if err != nil {
			if errors.Is(err, mongo.ErrNoDocuments) {
				return nil, fmt.Errorf("review not found")
			}
			return nil, fmt.Errorf("failed to delete review: %w", err)
		}

		if err := r.userStorage.ReduceRating(ctx, ratedUser, review.Score); err != nil {
			return nil, fmt.Errorf("failed to calculate rating: %w", err)
		}

		return nil, nil
	}

	return withTransaction(ctx, client, txnFunc)
}
