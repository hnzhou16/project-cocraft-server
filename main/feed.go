package main

import (
	"fmt"
	"github.com/hnzhou16/project-cocraft-server/internal/storage"
	"net/http"
)

type feedResponse struct {
	Posts      []storage.PostWithLikeStatus `json:"posts"`
	NextCursor *string                      `json:"next_cursor"` // !!! pointer allows nil
}

func (app *application) getPublicFeedHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	cq := storage.CursorQuery{
		Limit: 10,
		Sort:  "desc",
	}

	if err := cq.Parse(r); err != nil {
		app.badRequestError(w, r, err)
		return
	}

	if cq.ShowFollowing || cq.ShowMentioned {
		app.OutputJSON(w, http.StatusOK, nil)
	}

	posts, err := app.storage.Post.GetFeed(ctx, nil, cq)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	for i, post := range posts {
		user, err := app.storage.User.GetByID(ctx, post.Post.UserID.Hex())
		if err != nil {
			app.internalServerError(w, r, fmt.Errorf("failed to fetch username for userID %s: %w", post.Post.UserID.Hex(), err))
			return
		}
		posts[i].Username = user.Username
	}

	// do not allow infinite scroll for public feed
	response := feedResponse{
		Posts:      posts,
		NextCursor: nil,
	}

	app.OutputJSON(w, http.StatusOK, response)
}

func (app *application) getFeedHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := getUserFromCtx(r)

	cq := storage.CursorQuery{
		Limit: 10,
		Sort:  "desc",
	}

	if err := cq.Parse(r); err != nil {
		app.badRequestError(w, r, err)
		return
	}

	if cq.ShowFollowing == true {
		followeeIDs, err := app.storage.Follow.GetFollowing(ctx, user.ID)
		if err != nil {
			app.internalServerError(w, r, fmt.Errorf("failed to get following user: %w", err))
			return
		}

		// ! if user not following anyone, should avoid passing into the query
		if len(followeeIDs) == 0 {
			app.OutputJSON(w, http.StatusOK, nil)
		}
		cq.FolloweeIDs = followeeIDs
	}

	if err := Validate.Struct(cq); err != nil {
		app.badRequestError(w, r, err)
		return
	}

	posts, err := app.storage.Post.GetFeed(ctx, user, cq)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	// fetch username dynamically to the posts
	for i, post := range posts {
		user, err := app.storage.User.GetByID(ctx, post.Post.UserID.Hex())
		if err != nil {
			app.internalServerError(w, r, fmt.Errorf("failed to fetch username for userID %s: %w", post.Post.UserID.Hex(), err))
			return
		}
		posts[i].Username = user.Username
	}

	// !!! return cursor only if len(posts) >= limit, avoiding infinite loop of infinite scrolling if here is just 1 post
	var nextCursor *string
	if len(posts) >= cq.Limit {
		cursor := posts[len(posts)-1].Post.ID.Hex()
		nextCursor = &cursor
	}

	response := feedResponse{
		Posts:      posts,
		NextCursor: nextCursor,
	}

	app.OutputJSON(w, http.StatusOK, response)
}

func (app *application) getTrendingHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := getUserFromCtx(r)

	cq := storage.CursorQuery{
		Limit: 10,
		Sort:  "desc",
	}

	if err := cq.Parse(r); err != nil {
		app.badRequestError(w, r, err)
		return
	}

	if err := Validate.Struct(cq); err != nil {
		app.badRequestError(w, r, err)
		return
	}

	posts, err := app.storage.Post.GetTrending(ctx, user, cq)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	// fetch username dynamically to the posts
	for i, post := range posts {
		user, err := app.storage.User.GetByID(ctx, post.Post.UserID.Hex())
		if err != nil {
			app.internalServerError(w, r, fmt.Errorf("failed to fetch username for userID %s: %w", post.Post.UserID.Hex(), err))
			return
		}
		posts[i].Username = user.Username
	}

	var nextCursor *string
	if len(posts) >= cq.Limit {
		cursor := posts[len(posts)-1].Post.ID.Hex()
		nextCursor = &cursor
	}

	response := feedResponse{
		Posts:      posts,
		NextCursor: nextCursor,
	}

	app.OutputJSON(w, http.StatusOK, response)
}

func (app *application) getSearchHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := getUserFromCtx(r)

	cq := storage.CursorQuery{
		Limit: 10,
		Sort:  "desc",
	}
	if err := cq.Parse(r); err != nil {
		app.badRequestError(w, r, err)
		return
	}

	if cq.Search == "" {
		app.badRequestError(w, r, fmt.Errorf("search query is required"))
		return
	}

	if err := Validate.Struct(cq); err != nil {
		app.badRequestError(w, r, err)
		return
	}

	posts, err := app.storage.Post.Search(ctx, user, cq.Search, cq)
	if err != nil {
		app.internalServerError(w, r, fmt.Errorf("search error: %w", err))
		return
	}

	for i, post := range posts {
		u, err := app.storage.User.GetByID(ctx, post.Post.UserID.Hex())
		if err != nil {
			app.internalServerError(w, r, fmt.Errorf("failed to fetch username for userID %s: %w", post.Post.UserID.Hex(), err))
			return
		}
		posts[i].Username = u.Username
	}

	var nextCursor *string
	if len(posts) >= cq.Limit {
		cursor := posts[len(posts)-1].Post.ID.Hex()
		nextCursor = &cursor
	}

	response := feedResponse{
		Posts:      posts,
		NextCursor: nextCursor,
	}

	app.OutputJSON(w, http.StatusOK, response)
}
