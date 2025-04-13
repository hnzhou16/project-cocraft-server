package main

import (
	"errors"
	"github.com/go-chi/chi/v5"
	"github.com/hnzhou16/project-cocraft-server/internal/storage"
	"net/http"
)

type CreateReview struct {
	RatedUserID string `json:"rated_user_id" validate:"required"`
	Score       int    `json:"score" validate:"required"`
	Comment     string `json:"comment" validate:"omitempty,required,max=1500"`
}

type DeleteReview struct {
	RatedUserID string `json:"rated_user_id" validate:"required"`
}

func (app *application) createReviewHandler(w http.ResponseWriter, r *http.Request) {
	var payload CreateReview
	ctx := r.Context()

	if err := ReadJSON(w, r, &payload); err != nil {
		app.badRequestError(w, r, err)
		return
	}

	if err := Validate.Struct(payload); err != nil {
		app.badRequestError(w, r, err)
		return
	}

	ratedUser, err := app.storage.User.GetByID(ctx, payload.RatedUserID)
	if err != nil {
		switch {
		case errors.Is(err, storage.ErrUserNotFound):
			app.notFoundError(w, r, err)
		default:
			app.internalServerError(w, r, err)
		}
		return
	}

	rater := getUserFromCtx(r)

	review := &storage.Review{
		RatedUserID:   ratedUser.ID,
		RaterID:       rater.ID,
		RaterUsername: rater.Username,
		Score:         payload.Score,
		Comment:       payload.Comment,
	}

	if err := app.storage.Review.Create(ctx, review, ratedUser); err != nil {
		app.internalServerError(w, r, err)
		return
	}

	app.OutputJSON(w, http.StatusCreated, review)
}

func (app *application) deleteReviewHandler(w http.ResponseWriter, r *http.Request) {
	reviewID := chi.URLParam(r, "reviewID")

	var payload DeleteReview
	ctx := r.Context()

	if err := ReadJSON(w, r, &payload); err != nil {
		app.badRequestError(w, r, err)
		return
	}

	if err := Validate.Struct(payload); err != nil {
		app.badRequestError(w, r, err)
		return
	}

	ratedUser, err := app.storage.User.GetByID(ctx, payload.RatedUserID)
	if err != nil {
		switch {
		case errors.Is(err, storage.ErrUserNotFound):
			app.notFoundError(w, r, err)
		default:
			app.internalServerError(w, r, err)
		}
		return
	}

	err = app.storage.Review.Delete(ctx, reviewID, ratedUser)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
