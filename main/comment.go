package main

import (
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/hnzhou16/project-social/internal/storage"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"net/http"
)

type CreateCommentPayload struct {
	UserID  string `json:"user_id" validate:"required"`
	Content string `json:"content" validate:"required,max=500"`
}

func (app *application) createCommentHandler(w http.ResponseWriter, r *http.Request) {
	var payload CreateCommentPayload
	ctx := r.Context()

	if err := ReadJSON(w, r, &payload); err != nil {
		app.badRequestError(w, r, err)
		return
	}

	if err := Validate.Struct(payload); err != nil {
		app.badRequestError(w, r, err)
	}

	userID, err := primitive.ObjectIDFromHex(payload.UserID)
	if err != nil {
		app.badRequestError(w, r, fmt.Errorf("invalid user ID"))
		return
	}

	postID, err := primitive.ObjectIDFromHex(chi.URLParam(r, "postID"))
	if err != nil {
		app.badRequestError(w, r, fmt.Errorf("invalid post ID"))
		return
	}

	comment := &storage.Comment{
		UserID:  userID,
		PostID:  postID,
		Content: payload.Content,
	}

	if err := app.storage.Comment.Create(ctx, comment); err != nil {
		app.internalServerError(w, r, err)
		return
	}

	app.OutputJSON(w, http.StatusCreated, comment)
}

func (app *application) getCommentHandler(w http.ResponseWriter, r *http.Request) {
	postID, err := primitive.ObjectIDFromHex(chi.URLParam(r, "postID"))
	if err != nil {
		app.badRequestError(w, r, fmt.Errorf("invalid post ID"))
		return
	}

	exists, err := app.storage.Post.Exists(r.Context(), postID)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	if !exists {
		app.notFoundError(w, r, fmt.Errorf("post not found"))
		return
	}

	comments, err := app.storage.Comment.GetByPostID(r.Context(), postID)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	app.OutputJSON(w, http.StatusOK, comments)
}
