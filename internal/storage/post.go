package storage

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/hnzhou16/project-cocraft-server/internal/security"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	ErrPostNotFound = errors.New("post not found")
)

type Post struct {
	ID           primitive.ObjectID   `json:"id,omitempty" bson:"_id,omitempty"`
	UserID       primitive.ObjectID   `json:"user_id" bson:"user_id"`
	UserRole     security.Role        `json:"user_role" bson:"user_role"`
	Title        string               `json:"title" bson:"title"`
	Content      string               `json:"content" bson:"content"`
	Tags         []string             `json:"tags,omitempty" bson:"tags,omitempty"`
	Mentions     []string             `json:"mentions,omitempty" bson:"mentions,omitempty"`
	Images       []string             `json:"images,omitempty" bson:"images,omitempty"`
	LikeBy       []primitive.ObjectID `json:"-" bson:"like_by"`
	LikeCount    int64                `json:"like_count" bson:"like_count"`
	CommentCount int64                `json:"comment_count" bson:"comment_count"`
	Version      int64                `json:"version" bson:"version"`
	CreatedAt    time.Time            `json:"created_at,omitempty" bson:"created_at,omitempty"`
	UpdatedAt    time.Time            `json:"updated_at,omitempty" bson:"updated_at,omitempty"`
}

type PostWithLikeStatus struct {
	Post        Post   `json:"post"`
	Username    string `json:"username"`
	LikedByUser bool   `json:"liked_by_user"`
}

type PostStorage struct {
	collection *mongo.Collection
}

func (p *PostStorage) Create(ctx context.Context, post *Post) error {
	now := time.Now()
	post.ID = primitive.NewObjectID()
	post.LikeBy = []primitive.ObjectID{}
	post.CreatedAt = now
	post.UpdatedAt = now

	ctxTimeout, cancel := context.WithTimeout(ctx, QueryTimeout)
	defer cancel()

	result, err := p.collection.InsertOne(ctxTimeout, post)

	if err != nil {
		return fmt.Errorf("failed to create post: %w", err)
	}

	post.ID = result.InsertedID.(primitive.ObjectID)
	return nil
}

func (p *PostStorage) GetFeed(ctx context.Context, user *User, cq CursorQuery) ([]PostWithLikeStatus, error) {
	filter := bson.M{}

	sort := -1
	if cq.Sort == "asc" {
		sort = 1
	}

	if user != nil {
		if cq.ShowFollowing && len(cq.FolloweeIDs) > 0 {
			filter["user_id"] = bson.M{"$in": cq.FolloweeIDs}
		}

		if cq.ShowMentioned {
			// Check if the array contains the username
			filter["mentions"] = user.Username // !!! Direct equality check
		}
	}

	if len(cq.Roles) > 0 {
		filter["user_role"] = bson.M{"$in": cq.Roles}
	}

	// cursor query based on post id
	if cq.Cursor != "" && cq.Cursor != "undefined" {
		cursorID, err := primitive.ObjectIDFromHex(cq.Cursor)
		if err != nil {
			return nil, fmt.Errorf("invalid cursor ID: %w", err)
		}
		if sort == -1 {
			filter["_id"] = bson.M{"$lt": cursorID}
		} else {
			filter["_id"] = bson.M{"$gt": cursorID}
		}
	}

	opts := options.Find().
		SetSort(bson.M{"created_at": sort}).
		SetLimit(int64(cq.Limit))

	cursor, err := p.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to find posts: %w", err)
	}
	defer cursor.Close(ctx)

	var posts []Post

	if err := cursor.All(ctx, &posts); err != nil {
		return nil, fmt.Errorf("failed to decode posts: %w", err)
	}

	var result []PostWithLikeStatus
	if user == nil {
		for _, post := range posts {
			result = append(result, PostWithLikeStatus{Post: post, LikedByUser: false})
		}
		return result, nil
	}

	for _, post := range posts {
		liked := false
		for _, id := range post.LikeBy {
			if user.ID == id {
				liked = true
				break
			}
		}
		result = append(result, PostWithLikeStatus{Post: post, LikedByUser: liked})
	}

	return result, nil
}

func (p *PostStorage) GetTrending(ctx context.Context, user *User, cq CursorQuery) ([]PostWithLikeStatus, error) {
	pipeline := mongo.Pipeline{
		// Optional: only show past 48hr posts
		//{{Key: "$match", Value: bson.M{
		//	"created_at": bson.M{"$gte": time.Now().Add(-48 * time.Hour)},
		//}}},
		{{Key: "$addFields", Value: bson.M{
			"engagement_score": bson.M{"$add": bson.A{"$like_count", "$comment_count"}},
		}}},
		{{Key: "$sort", Value: bson.D{
			{Key: "engagement_score", Value: -1},
			{Key: "created_at", Value: -1}, // tie-breaker
		}}},
		{{Key: "$limit", Value: cq.Limit}},
	}

	cursor, err := p.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to aggregate trending posts: %w", err)
	}
	defer cursor.Close(ctx)

	var posts []Post

	if err := cursor.All(ctx, &posts); err != nil {
		return nil, fmt.Errorf("failed to decode posts: %w", err)
	}

	var result []PostWithLikeStatus
	if user == nil {
		for _, post := range posts {
			result = append(result, PostWithLikeStatus{Post: post, LikedByUser: false})
		}
		return result, nil
	}

	for _, post := range posts {
		liked := false
		for _, id := range post.LikeBy {
			if user.ID == id {
				liked = true
				break
			}
		}
		result = append(result, PostWithLikeStatus{Post: post, LikedByUser: liked})
	}

	return result, nil
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
			return nil, ErrPostNotFound
		}
		return nil, fmt.Errorf("post query failed: %w", err)
	}

	return &post, nil
}

func (p *PostStorage) GetByUserID(ctx context.Context, userID primitive.ObjectID, cq CursorQuery) ([]PostWithLikeStatus, error) {
	ctxTimeout, cancel := context.WithTimeout(ctx, QueryTimeout)
	defer cancel()

	sort := -1
	if cq.Sort == "asc" {
		sort = 1
	}

	filter := bson.M{"user_id": userID}

	// cursor query based on post id
	if cq.Cursor != "" && cq.Cursor != "undefined" {
		cursorID, err := primitive.ObjectIDFromHex(cq.Cursor)
		if err != nil {
			return nil, fmt.Errorf("invalid cursor ID: %w", err)
		}
		if sort == -1 {
			filter["_id"] = bson.M{"$lt": cursorID}
		} else {
			filter["_id"] = bson.M{"$gt": cursorID}
		}
	}

	opts := options.Find().
		SetSort(bson.M{"created_at": sort}).
		SetLimit(int64(cq.Limit))

	cursor, err := p.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to find posts: %w", err)
	}
	defer cursor.Close(ctx)

	var posts []Post

	if err := cursor.All(ctxTimeout, &posts); err != nil {
		return nil, fmt.Errorf("failed to decode posts: %w", err)
	}

	var result []PostWithLikeStatus
	for _, post := range posts {
		liked := false
		for _, id := range post.LikeBy {
			if userID == id {
				liked = true
				break
			}
		}
		result = append(result, PostWithLikeStatus{Post: post, LikedByUser: liked})
	}

	return result, nil
}

func (p *PostStorage) GetCountByUserID(ctx context.Context, userID primitive.ObjectID) (int, error) {
	ctxTimeout, cancel := context.WithTimeout(ctx, QueryTimeout)
	defer cancel()

	count, err := p.collection.CountDocuments(ctxTimeout, bson.M{"user_id": userID})
	if err != nil {
		return 0, fmt.Errorf("failed to count posts: %w", err)
	}

	return int(count), nil
}

func (p *PostStorage) Search(ctx context.Context, user *User, query string, cq CursorQuery) ([]PostWithLikeStatus, error) {
	log.Println(cq)

	ctxTimeout, cancel := context.WithTimeout(ctx, QueryTimeout)
	defer cancel()

	regex := bson.M{"$regex": primitive.Regex{Pattern: query, Options: "i"}} // case-insensitive

	filter := bson.M{}

	orConditions := []bson.M{
		{"title": regex},
		{"content": regex},
		{"tags": regex},
	}

	andConditions := []bson.M{
		{"$or": orConditions},
	}

	sort := -1
	if cq.Sort == "asc" {
		sort = 1
	}

	if user != nil {
		if cq.ShowFollowing && len(cq.FolloweeIDs) > 0 {
			andConditions = append(andConditions, bson.M{"user_id": bson.M{"$in": cq.FolloweeIDs}})
		}

		if cq.ShowMentioned {
			// Check if the array contains the username
			andConditions = append(andConditions, bson.M{"mentions": user.Username}) // !!! Direct equality check
		}

		if len(cq.Roles) > 0 {
			andConditions = append(andConditions, bson.M{"user_role": bson.M{"$in": cq.Roles}})
		}
	}

	// cursor query based on post id
	if cq.Cursor != "" && cq.Cursor != "undefined" {
		cursorID, err := primitive.ObjectIDFromHex(cq.Cursor)
		if err != nil {
			return nil, fmt.Errorf("invalid cursor ID: %w", err)
		}
		if sort == -1 {
			andConditions = append(andConditions, bson.M{"_id": bson.M{"$lt": cursorID}})
		} else {
			andConditions = append(andConditions, bson.M{"_id": bson.M{"$gt": cursorID}})
		}
	}

	filter["$and"] = andConditions

	opts := options.Find().
		SetSort(bson.M{"_id": sort}).
		SetLimit(int64(cq.Limit))

	cursor, err := p.collection.Find(ctxTimeout, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to find posts: %w", err)
	}
	defer cursor.Close(ctx)

	var posts []Post
	if err := cursor.All(ctxTimeout, &posts); err != nil {
		return nil, fmt.Errorf("failed to decode posts: %w", err)
	}

	var result []PostWithLikeStatus
	for _, post := range posts {
		liked := false
		if user != nil {
			for _, id := range post.LikeBy {
				if user.ID == id {
					liked = true
					break
				}
			}
		}
		result = append(result, PostWithLikeStatus{
			Post:        post,
			LikedByUser: liked,
		})
	}

	return result, nil
}

func (p *PostStorage) Update(ctx context.Context, post *Post) error {
	var objID = post.ID
	now := time.Now()

	update := bson.M{
		"$set": bson.M{
			"title":      post.Title,
			"content":    post.Content,
			"tags":       post.Tags,
			"mentions":   post.Mentions,
			"images":     post.Images,
			"version":    post.Version,
			"updated_at": now,
		},
	}

	ctxTimeout, cancel := context.WithTimeout(ctx, QueryTimeout)
	defer cancel()

	_, err := p.collection.UpdateOne(ctxTimeout, bson.M{"_id": objID}, update)
	if err != nil {
		return fmt.Errorf("failed to update post: %w", err)
	}

	return nil
}

func (p *PostStorage) ToggleLike(ctx context.Context, userID primitive.ObjectID, post *Post) (bool, error) {
	ctxTimeout, cancel := context.WithTimeout(ctx, QueryTimeout)
	defer cancel()

	liked := false
	for _, id := range post.LikeBy {
		if id == userID {
			liked = true
			break
		}
	}

	// "$xx" make sure updating 'like_by' and 'like_count' is atomic
	var update = bson.M{}
	if liked {
		update = bson.M{
			"$pull": bson.M{"like_by": userID},
			"$inc":  bson.M{"like_count": -1},
		}
	} else {
		update = bson.M{
			"$addToSet": bson.M{"like_by": userID},
			"$inc":      bson.M{"like_count": 1},
		}
	}

	_, err := p.collection.UpdateByID(ctxTimeout, post.ID, update)
	if err != nil {
		return liked, fmt.Errorf("failed to update post: %w", err)
	}

	return !liked, nil
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
