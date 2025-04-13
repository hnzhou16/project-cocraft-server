package main

import (
	"errors"
	"fmt"
	"github.com/hnzhou16/project-cocraft-server/internal/storage"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"net/http"
)

type CreateCommentPayload struct {
	ParentID string `json:"parent_id,omitempty" validate:"omitempty,hexadecimal,len=24"`
	Content  string `json:"content" validate:"required,max=500"`
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
		return
	}

	user := getUserFromCtx(r)
	post := getPostFromCtx(r)

	comment := &storage.Comment{
		UserID:  user.ID,
		PostID:  post.ID,
		Content: payload.Content,
	}

	// use "" to check empty string rather than nil
	if payload.ParentID != "" {
		objID, err := primitive.ObjectIDFromHex(payload.ParentID)
		if err != nil {
			app.badRequestError(w, r, fmt.Errorf("invalid parent ID"))
			return
		}

		_, err = app.storage.Comment.Exists(ctx, objID)
		if err != nil {
			switch {
			case errors.Is(err, storage.ErrCommentNotFound):
				app.notFoundError(w, r, err)
				return
			default:
				app.internalServerError(w, r, err)
				return
			}
		}

		comment.ParentID = &objID
	}

	if err := app.storage.Comment.Create(ctx, comment); err != nil {
		app.internalServerError(w, r, err)
		return
	}

	app.OutputJSON(w, http.StatusCreated, comment)
}

func (app *application) getCommentHandler(w http.ResponseWriter, r *http.Request) {
	post := getPostFromCtx(r)

	comments, err := app.storage.Comment.GetByPostID(r.Context(), post.ID)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	app.OutputJSON(w, http.StatusOK, comments)
}
