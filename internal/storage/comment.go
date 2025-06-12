package storage

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	ErrCommentNotFound = errors.New("comment not found")
)

type Comment struct {
	ID        primitive.ObjectID  `json:"id,omitempty" bson:"_id,omitempty"`
	UserID    primitive.ObjectID  `json:"user_id" bson:"user_id"`
	PostID    primitive.ObjectID  `json:"post_id" bson:"post_id"`
	ParentID  *primitive.ObjectID `json:"parent_id,omitempty" bson:"parent_id,omitempty"` // make it pointer to allow nil
	Content   string              `json:"content" bson:"content"`
	CreatedAt time.Time           `json:"created_at" bson:"created_at"`
}

// ParentComment - need to add 'bson' in the struct to be able to decode from mongoDB
type ParentComment struct {
	ID        primitive.ObjectID `json:"id,omitempty" bson:"_id,omitempty"`
	UserID    primitive.ObjectID `json:"user_id" bson:"user_id"`
	Content   string             `json:"content" bson:"content"`
	CreatedAt time.Time          `json:"created_at" bson:"created_at"`
}

type CommentWithParentAndUser struct {
	ID            primitive.ObjectID `json:"id"`
	UserID        primitive.ObjectID `json:"user_id"`
	Username      string             `json:"username"`
	PostID        primitive.ObjectID `json:"post_id"`
	Content       string             `json:"content"`
	CreatedAt     time.Time          `json:"created_at"`
	ParentComment *ParentComment     `json:"parent_comment,omitempty"`
}

type CommentStorage struct {
	collection  *mongo.Collection
	userStorage *UserStorage
	postStorage *PostStorage
}

func (c *CommentStorage) Create(ctx context.Context, comment *Comment) (CommentWithParentAndUser, error) {
	client := c.collection.Database().Client()

	var result CommentWithParentAndUser

	txnFunc := func(sessCtx mongo.SessionContext) (interface{}, error) {
		comment.ID = primitive.NewObjectID()
		comment.CreatedAt = time.Now()

		ctxTimeout, cancel := context.WithTimeout(ctx, QueryTimeout)
		defer cancel()

		_, err := c.collection.InsertOne(ctxTimeout, comment)

		if err != nil {
			return nil, fmt.Errorf("failed to create comment: %w", err)
		}

		// can not directly use Collection.Post.IncrementCommentCount, because it's not initialized on that path
		// it's only initialized through app := &application in main, but not worthy to import from there
		if err := c.postStorage.IncrementCommentCount(ctxTimeout, comment.PostID); err != nil {
			return nil, fmt.Errorf("failed to increment comment count: %w", err)
		}

		var user User
		if err := c.userStorage.collection.FindOne(ctxTimeout, bson.M{"_id": comment.UserID}).Decode(&user); err != nil {
			return nil, fmt.Errorf("failed to get user: %w", err)
		}

		result = CommentWithParentAndUser{
			ID:        comment.ID,
			UserID:    comment.UserID,
			Username:  user.Username,
			PostID:    comment.PostID,
			Content:   comment.Content,
			CreatedAt: comment.CreatedAt,
		}

		var parent ParentComment
		if comment.ParentID != nil {
			if err := c.collection.FindOne(ctxTimeout, bson.M{"_id": comment.ParentID}).Decode(&parent); err != nil {
				return nil, fmt.Errorf("failed to get parent comment: %w", err)
			}
			result.ParentComment = &parent
		}

		return nil, nil
	}

	err := withTransaction(ctx, client, txnFunc)

	return result, err
}

func (c *CommentStorage) Exists(ctx context.Context, commentID primitive.ObjectID) (bool, error) {
	count, err := c.collection.CountDocuments(ctx, bson.M{"_id": commentID})
	if err != nil {
		return false, fmt.Errorf("failed to check if the comment exists: %w", err)
	}

	if count == 0 {
		return false, ErrCommentNotFound
	}
	return true, nil
}

func (c *CommentStorage) GetByPostID(ctx context.Context, postID primitive.ObjectID) ([]CommentWithParentAndUser, error) {
	ctxTimeout, cancel := context.WithTimeout(ctx, QueryTimeout)
	defer cancel()

	// fetch comment
	cursor, err := c.collection.Find(ctxTimeout, bson.M{"post_id": postID},
		options.Find().SetSort(bson.M{"created_at": -1}))
	if err != nil {
		return nil, fmt.Errorf("failed to get comment by post id: %w", err)
	}
	defer cursor.Close(ctxTimeout)

	var comments []Comment

	if err = cursor.All(ctxTimeout, &comments); err != nil {
		return nil, fmt.Errorf("failed to decode comments: %w", err)
	}

	// get all parent comments and username in a single batch query
	// collect parentIDs and userIDs
	parentIDs := make([]primitive.ObjectID, 0)
	userIDs := make([]primitive.ObjectID, 0, len(comments))
	for _, c := range comments {
		// check if ParentID is a non-zero value (not omitted.empty)
		// can not use ParentID != nil, because it's not a pointer
		if c.ParentID != nil {
			parentIDs = append(parentIDs, *c.ParentID)
		}
		userIDs = append(userIDs, c.UserID)
	}

	// fetch all parent comments
	var parentComments []ParentComment
	if len(parentIDs) > 0 {
		pCursor, err := c.collection.Find(ctxTimeout, bson.M{"_id": bson.M{"$in": parentIDs}})
		if err != nil {
			return nil, fmt.Errorf("failed to get comment by parent id: %w", err)
		}
		defer pCursor.Close(ctxTimeout)

		if err = pCursor.All(ctxTimeout, &parentComments); err != nil {
			return nil, fmt.Errorf("failed to decode parent comments: %w", err)
		}
	}

	// fetch all usernames
	var users []User
	if len(userIDs) > 0 {
		uCursor, err := c.userStorage.collection.Find(ctxTimeout, bson.M{"_id": bson.M{"$in": userIDs}})
		if err != nil {
			return nil, fmt.Errorf("failed to get users: %w", err)
		}
		defer uCursor.Close(ctxTimeout)

		if err = uCursor.All(ctxTimeout, &users); err != nil {
			return nil, fmt.Errorf("failed to decode users: %w", err)
		}
	}

	// index for quick lookup
	parentMap := make(map[primitive.ObjectID]*ParentComment)
	for _, p := range parentComments {
		parentMap[p.ID] = &p
	}

	userMap := make(map[primitive.ObjectID]string)
	for _, u := range users {
		userMap[u.ID] = u.Username
	}

	// final response
	result := make([]CommentWithParentAndUser, 0, len(comments))
	for _, c := range comments {
		comment := CommentWithParentAndUser{
			ID:        c.ID,
			UserID:    c.UserID,
			Username:  userMap[c.UserID],
			PostID:    c.PostID,
			Content:   c.Content,
			CreatedAt: c.CreatedAt,
		}
		if c.ParentID != nil {
			if parent, ok := parentMap[*c.ParentID]; ok {
				comment.ParentComment = parent
			}
		}
		result = append(result, comment)
	}

	return result, nil
}
