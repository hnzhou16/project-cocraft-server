package storage

import (
	"context"
	"errors"
	"fmt"
	"github.com/hnzhou16/project-social/internal/security"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type Post struct {
	ID           primitive.ObjectID `json:"id,omitempty" bson:"_id,omitempty"`
	UserID       primitive.ObjectID `json:"user_id" bson:"user_id"`
	UserRole     security.Role      `json:"user_role" bson:"user_role"`
	Title        string             `json:"title" bson:"title"`
	Content      string             `json:"content" bson:"content"`
	Tags         []string           `json:"tags,omitempty" bson:"tags,omitempty"`
	Mentions     []string           `json:"mentions,omitempty" bson:"mentions,omitempty"`
	ImagesPath   []string           `json:"images,omitempty" bson:"images,omitempty"`
	CommentCount int64              `json:"comment_count" bson:"comment_count"`
	Version      int64              `json:"version" bson:"version"`
	CreatedAt    time.Time          `json:"created_at,omitempty" bson:"created_at,omitempty"`
	UpdatedAt    time.Time          `json:"updated_at,omitempty" bson:"updated_at,omitempty"`
}

type PostStorage struct {
	collection *mongo.Collection
}

func (p *PostStorage) Create(ctx context.Context, post *Post) error {
	now := time.Now()
	post.ID = primitive.NewObjectID()
	post.CreatedAt = now
	post.UpdatedAt = now

	ctxTimeout, cancel := context.WithTimeout(ctx, QueryTimeout)
	defer cancel()

	result, err := p.collection.InsertOne(ctxTimeout, bson.M{
		"_id":           post.ID,
		"user_id":       post.UserID,
		"user_role":     post.UserRole,
		"title":         post.Title,
		"content":       post.Content,
		"tags":          post.Tags,
		"mentions":      post.Mentions,
		"images_path":   post.ImagesPath,
		"comment_count": post.CommentCount,
		"version":       post.Version,
		"created_at":    post.CreatedAt,
		"updated_at":    post.UpdatedAt,
	})

	if err != nil {
		return fmt.Errorf("failed to create post: %w", err)
	}

	post.ID = result.InsertedID.(primitive.ObjectID)
	return nil
}

func (p *PostStorage) Exists(ctx context.Context, postID primitive.ObjectID) (bool, error) {
	count, err := p.collection.CountDocuments(ctx, bson.M{"_id": postID})
	if err != nil {
		return false, fmt.Errorf("failed to check if the post exists: %w", err)
	}
	return count > 0, nil
}

func (p *PostStorage) GetFeed(ctx context.Context, user *User, pq PaginationQuery) ([]Post, error) {
	filter := bson.M{}

	sort := -1
	if pq.Sort == "asc" {
		sort = 1
	}

	if user != nil {
		if len(pq.FolloweeIDs) > 0 {
			filter["user_id"] = bson.M{"$in": pq.FolloweeIDs}
		}

		if pq.ShowMentioned {
			// Check if the array contains the username
			filter["mentions"] = user.Username // Direct equality check
		}

		if len(pq.Roles) > 0 {
			filter["user_role"] = bson.M{"$in": pq.Roles}
		}
	}

	opts := options.Find().
		SetSort(bson.M{"created_at": sort}).
		SetSkip(int64(pq.Offset)).
		SetLimit(int64(pq.Limit))

	cursor, err := p.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to find posts: %w", err)
	}
	defer cursor.Close(ctx)

	var posts []Post
	if err := cursor.All(ctx, &posts); err != nil {
		return nil, fmt.Errorf("failed to decode posts: %w", err)
	}

	return posts, nil
}

func (p *PostStorage) GetByID(ctx context.Context, postID string) (*Post, error) {
	// ObjectID in MongoDB is a 12-byte binary value represented as a 24-character hexadecimal string
	objID, err := primitive.ObjectIDFromHex(postID)
	if err != nil {
		return nil, fmt.Errorf("failed to convert postID to ObjectID: %w", err)
	}

	ctxTimeout, cancel := context.WithTimeout(ctx, QueryTimeout)
	defer cancel()

	var post Post
	// Decode is MongoDB method to deserialize result in BSON of a query into Go struct
	err = p.collection.FindOne(ctxTimeout, bson.M{"_id": objID}).Decode(&post)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, fmt.Errorf("post not found")
		}
		return nil, fmt.Errorf("post query failed: %w", err)
	}

	return &post, nil
}

func (p *PostStorage) GetByUserID(ctx context.Context, userID primitive.ObjectID, pq PaginationQuery) ([]Post, error) {
	ctxTimeout, cancel := context.WithTimeout(ctx, QueryTimeout)
	defer cancel()

	sort := -1
	if pq.Sort == "asc" {
		sort = 1
	}

	log.Println(userID)

	filter := bson.M{"user_id": userID}

	opts := options.Find().
		SetSort(bson.M{"created_at": sort}).
		SetSkip(int64(pq.Offset)).
		SetLimit(int64(pq.Limit))

	cursor, err := p.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to find posts: %w", err)
	}
	defer cursor.Close(ctx)

	var posts []Post

	if err := cursor.All(ctxTimeout, &posts); err != nil {
		return nil, fmt.Errorf("failed to decode posts: %w", err)
	}

	return posts, nil
}

func (p *PostStorage) Update(ctx context.Context, post *Post) error {
	var objID = post.ID
	now := time.Now()

	update := bson.M{
		"$set": bson.M{
			"title":       post.Title,
			"content":     post.Content,
			"tags":        post.Tags,
			"mentions":    post.Mentions,
			"images_path": post.ImagesPath,
			"version":     post.Version,
			"updated_at":  now,
		},
	}

	ctx, cancel := context.WithTimeout(ctx, QueryTimeout)
	defer cancel()

	_, err := p.collection.UpdateOne(ctx, bson.M{"_id": objID}, update)
	if err != nil {
		return fmt.Errorf("failed to update post: %w", err)
	}

	return nil
}

func (p *PostStorage) IncrementCommentCount(ctx context.Context, postID primitive.ObjectID) error {
	_, err := p.collection.UpdateOne(ctx,
		bson.M{"_id": postID},
		bson.M{"$inc": bson.M{"comment_count": 1}},
	)
	return err
}

func (p *PostStorage) Delete(ctx context.Context, postID string) error {
	objID, err := primitive.ObjectIDFromHex(postID)
	if err != nil {
		return fmt.Errorf("failed to convert postID to ObjectID: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, QueryTimeout)
	defer cancel()

	_, err = p.collection.DeleteOne(ctx, bson.M{"_id": objID})
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return fmt.Errorf("post not found")
		}
		return fmt.Errorf("failed to delete post: %w", err)
	}

	return nil
}
