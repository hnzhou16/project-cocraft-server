package storage

import (
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type Comment struct {
	ID        primitive.ObjectID `json:"id,omitempty" bson:"_id,omitempty"`
	UserID    primitive.ObjectID `json:"user_id" bson:"user_id"`
	PostID    primitive.ObjectID `json:"post_id" bson:"post_id"`
	Content   string             `json:"content" bson:"content"`
	CreatedAt time.Time          `json:"created_at" bson:"created_at"`
}

type CommentStorage struct {
	collection  *mongo.Collection
	postStorage *PostStorage
}

func (c *CommentStorage) Create(ctx context.Context, comment *Comment) error {
	client := c.collection.Database().Client()

	txnFunc := func(sessCtx mongo.SessionContext) (interface{}, error) {
		comment.ID = primitive.NewObjectID()
		comment.CreatedAt = time.Now()

		ctxTimeout, cancel := context.WithTimeout(ctx, QueryTimeout)
		defer cancel()

		_, err := c.collection.InsertOne(ctxTimeout, bson.M{
			"_id":        comment.ID,
			"user_id":    comment.UserID,
			"post_id":    comment.PostID,
			"content":    comment.Content,
			"created_at": comment.CreatedAt,
		})

		if err != nil {
			return nil, fmt.Errorf("failed to create comment: %w", err)
		}

		// can not directly use Collection.Post.IncrementCommentCount, because it's not initialized on that path
		// it's only initialized through app := &application in main, but not worthy to import from there
		if err := c.postStorage.IncrementCommentCount(ctxTimeout, comment.PostID); err != nil {
			return nil, fmt.Errorf("failed to increment comment count: %w", err)
		}

		return nil, nil
	}

	return withTransaction(ctx, client, txnFunc)
}

func (c *CommentStorage) GetByPostID(ctx context.Context, postID primitive.ObjectID) (*[]Comment, error) {
	ctxTimeout, cancel := context.WithTimeout(ctx, QueryTimeout)
	defer cancel()

	cursor, err := c.collection.Find(ctxTimeout, bson.M{"post_id": postID},
		options.Find().SetSort(bson.M{"created_at": -1}))
	if err != nil {
		return nil, fmt.Errorf("failed to get comment by post id: %w", err)
	}
	defer func() {
		if err := cursor.Close(ctxTimeout); err != nil {
			log.Printf("failed to close cursor: %v", err)
		}
	}()

	var comments []Comment

	if err = cursor.All(ctxTimeout, &comments); err != nil {
		return nil, fmt.Errorf("failed to get comment by post id: %w", err)
	}

	return &comments, nil
}
