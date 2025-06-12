package main

import (
	"fmt"
	"github.com/hnzhou16/project-cocraft-server/internal/storage"
	"net/http"
)

func (app *application) getPublicFeedHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	pq := storage.PaginationQuery{
		Limit:  10,
		Offset: 0,
		Sort:   "desc",
	}

	feed, err := app.storage.Post.GetFeed(ctx, nil, pq)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	for i, post := range feed {
		user, err := app.storage.User.GetByID(ctx, post.Post.UserID.Hex())
		if err != nil {
			app.internalServerError(w, r, fmt.Errorf("failed to fetch username for userID %s: %w", post.Post.UserID.Hex(), err))
			return
		}
		feed[i].Username = user.Username
	}

	app.OutputJSON(w, http.StatusOK, feed)
}

func (app *application) getFeedHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := getUserFromCtx(r)

	pq := storage.PaginationQuery{
		Limit:  10,
		Offset: 0,
		Sort:   "desc",
	}

	if err := pq.Parse(r); err != nil {
		app.badRequestError(w, r, err)
		return
	}

	if pq.ShowFollowing == true {
		followeeIDs, err := app.storage.Follow.GetFollowing(ctx, user.ID)
		if err != nil {
			app.internalServerError(w, r, fmt.Errorf("failed to get following user: %w", err))
			return
		}

		// ! if user not following anyone, should avoid passing into the query
		if len(followeeIDs) == 0 {
			app.OutputJSON(w, http.StatusOK, nil)
		}
		pq.FolloweeIDs = followeeIDs
	}

	if err := Validate.Struct(pq); err != nil {
		app.badRequestError(w, r, err)
		return
	}

	feed, err := app.storage.Post.GetFeed(ctx, user, pq)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	// fetch username dynamically to the posts
	for i, post := range feed {
		user, err := app.storage.User.GetByID(ctx, post.Post.UserID.Hex())
		if err != nil {
			app.internalServerError(w, r, fmt.Errorf("failed to fetch username for userID %s: %w", post.Post.UserID.Hex(), err))
			return
		}
		feed[i].Username = user.Username
	}

	app.OutputJSON(w, http.StatusOK, feed)
}

func (app *application) getTrendingHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := getUserFromCtx(r)

	pq := storage.PaginationQuery{
		Limit:  10,
		Offset: 0,
		Sort:   "desc",
	}

	if err := pq.Parse(r); err != nil {
		app.badRequestError(w, r, err)
		return
	}

	if err := Validate.Struct(pq); err != nil {
		app.badRequestError(w, r, err)
		return
	}

	feed, err := app.storage.Post.GetTrending(ctx, user, pq)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	// fetch username dynamically to the posts
	for i, post := range feed {
		user, err := app.storage.User.GetByID(ctx, post.Post.UserID.Hex())
		if err != nil {
			app.internalServerError(w, r, fmt.Errorf("failed to fetch username for userID %s: %w", post.Post.UserID.Hex(), err))
			return
		}
		feed[i].Username = user.Username
	}

	app.OutputJSON(w, http.StatusOK, feed)
}
