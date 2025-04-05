package main

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/hnzhou16/project-social/internal/storage"
)

func (app *application) activateUserHandler(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")

	err := app.storage.User.Activate(r.Context(), token)

	if err != nil {
		switch {
		case errors.Is(err, storage.ErrUserNotFound):
			app.notFoundError(w, r, err)
		case errors.Is(err, storage.ErrInviteNotFound):
			app.notFoundError(w, r, err)
		default:
			app.internalServerError(w, r, err)
		}
		return
	}

	app.OutputJSON(w, http.StatusNoContent, nil)
}

func (app *application) getUserHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromCtx(r)
	app.OutputJSON(w, http.StatusOK, user)
}

func (app *application) getAllUsersHandler(w http.ResponseWriter, r *http.Request) {
	users, err := app.storage.User.GetAll(r.Context())
	if err != nil {
		app.internalServerError(w, r, err)
	}
	app.OutputJSON(w, http.StatusOK, users)
}

func (app *application) getFollowingUserHandler(w http.ResponseWriter, r *http.Request) {
	followerUserID := getUserFromCtx(r).ID
	ctx := r.Context()

	followeeIDs, err := app.storage.Follow.GetFollowing(ctx, followerUserID)
	if err != nil {
		app.internalServerError(w, r, fmt.Errorf("failed to get following user: %w", err))
		return
	}

	responseIDs := make([]string, 0, len(followeeIDs))
	for _, id := range followeeIDs {
		responseIDs = append(responseIDs, id.Hex())
	}

	app.OutputJSON(w, http.StatusOK, map[string]any{"following_ids": responseIDs})
}

func (app *application) followStatusHandler(w http.ResponseWriter, r *http.Request) {
	followerUserID := getUserFromCtx(r).ID
	followeeUserID := chi.URLParam(r, "userID")
	ctx := r.Context()

	followee, err := app.storage.User.GetByID(ctx, followeeUserID)
	if err != nil {
		switch {
		case errors.Is(err, storage.ErrUserNotFound):
			app.notFoundError(w, r, err)
		default:
			app.internalServerError(w, r, err)
		}
		return
	}

	isFollowing, err := app.storage.Follow.IsFollowing(ctx, followerUserID, followee.ID)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	app.OutputJSON(w, http.StatusOK, map[string]bool{"isFollowing": isFollowing})
}

// followUserHandler userID from the url is the followee ID
func (app *application) followUserHandler(w http.ResponseWriter, r *http.Request) {
	followerUserID := getUserFromCtx(r).ID
	followeeUserID := chi.URLParam(r, "userID")
	ctx := r.Context()

	followee, err := app.storage.User.GetByID(ctx, followeeUserID)
	if err != nil {
		switch {
		case errors.Is(err, storage.ErrUserNotFound):
			app.notFoundError(w, r, err)
		default:
			app.internalServerError(w, r, err)
		}
		return
	}

	if followerUserID == followee.ID {
		app.badRequestError(w, r, fmt.Errorf("you cannot follow yourself"))
		return
	}

	if err := app.storage.Follow.FollowUser(ctx, followerUserID, followee.ID); err != nil {
		app.internalServerError(w, r, err)
		return
	}

	app.OutputJSON(w, http.StatusOK, map[string]bool{"isFollowing": true})
}

func (app *application) unfollowUserHandler(w http.ResponseWriter, r *http.Request) {
	followerUserID := getUserFromCtx(r).ID
	followeeUserID := chi.URLParam(r, "userID")
	ctx := r.Context()

	followee, err := app.storage.User.GetByID(ctx, followeeUserID)
	if err != nil {
		switch {
		case errors.Is(err, storage.ErrUserNotFound):
			app.notFoundError(w, r, err)
		default:
			app.internalServerError(w, r, err)
		}
		return
	}

	if err := app.storage.Follow.UnfollowUser(ctx, followerUserID, followee.ID); err != nil {
		app.internalServerError(w, r, err)
		return
	}

	app.OutputJSON(w, http.StatusOK, map[string]bool{"isFollowing": false})
}
