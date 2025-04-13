package main

import (
	"context"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/hnzhou16/project-cocraft-server/internal/storage"
	"net/http"
	"regexp"
)

type CreatePostPayload struct {
	Title      string   `json:"title" validate:"required,max=150"`
	Content    string   `json:"content" validate:"required,max=1500"`
	Tags       []string `json:"tags" validate:"omitempty,dive,required"`
	ImagesPath []string `json:"images_path" validate:"omitempty,dive,required"`
}

// UpdatePostPayload - all fields are pointers with nil as default, otherwise they'll be a 0("") as default
//
//	so fields can be nil if no input from the client vs "" if input is intentionally ""
type UpdatePostPayload struct {
	Title      *string   `json:"title" validate:"omitempty,max=150"`
	Content    *string   `json:"content" validate:"omitempty,max=1500"`
	Tags       *[]string `json:"tags" validate:"omitempty,dive,required"`
	ImagesPath *[]string `json:"images_path" validate:"omitempty,dive,required"`
	Version    int64     `json:"version" validate:"required"`
}

func extractMentions(text string) []string {
	re := regexp.MustCompile(`@([a-zA-Z0-9_]+)`)
	// '-1' means no limit, return all matches
	//[["@user1", "user1"], ["@user2", "user2"] ]
	matches := re.FindAllStringSubmatch(text, -1)

	var mentions []string
	for _, match := range matches {
		mentions = append(mentions, match[1])
	}

	return mentions
}

func (app *application) createPostHandler(w http.ResponseWriter, r *http.Request) {
	var payload CreatePostPayload
	ctx := r.Context()
	user := getUserFromCtx(r)

	if err := ReadJSON(w, r, &payload); err != nil {
		app.badRequestError(w, r, err)
		return
	}

	if err := Validate.Struct(payload); err != nil {
		app.badRequestError(w, r, err)
		return
	}

	mentions := extractMentions(payload.Content)

	if len(mentions) > 0 {
		validMentions, err := app.storage.User.ValidateUsername(ctx, mentions)
		if err != nil {
			app.notFoundError(w, r, err)
			return
		}
		mentions = validMentions
	}

	post := &storage.Post{
		UserID:       user.ID,
		UserRole:     user.Role,
		Title:        payload.Title,
		Content:      payload.Content,
		Tags:         payload.Tags,
		Mentions:     mentions,
		ImagesPath:   payload.ImagesPath,
		CommentCount: 0,
		Version:      1,
	}

	if err := app.storage.Post.Create(ctx, post); err != nil {
		app.internalServerError(w, r, err)
		return
	}

	app.OutputJSON(w, http.StatusCreated, post)
}

// s3KeysToUrl - turn images saved as s3 object keys into url
func (app *application) s3KeysToUrl(ctx context.Context, post *storage.Post) error {
	urls := make([]string, len(post.ImagesPath))
	for i, key := range post.ImagesPath {
		url, err := app.awsPresigner.GetImageURL(
			ctx,
			key,
			app.config.awsConfig.exp,
		)
		if err != nil {
			return fmt.Errorf("failed to get image url: %w", err)
		}
		urls[i] = url
	}

	post.ImagesPath = urls
	return nil
}

func (app *application) getAllUserPostsHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := chi.URLParam(r, "userID")

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

	user, err := app.storage.User.GetByID(ctx, userID)
	if err != nil {
		app.notFoundError(w, r, err)
		return
	}

	posts, err := app.storage.Post.GetByUserID(ctx, user.ID, pq)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	// !!! 'for _, post := range posts' creates a copy and won't modify the original slice
	// need to use index-based iteration
	if len(posts) > 0 {
		for i := range posts {
			if err := app.s3KeysToUrl(ctx, &posts[i].Post); err != nil {
				app.internalServerError(w, r, err)
				return
			}
		}
	}

	app.OutputJSON(w, http.StatusOK, posts)
}

func (app *application) getPostHandler(w http.ResponseWriter, r *http.Request) {
	post := getPostFromCtx(r)

	if err := app.s3KeysToUrl(r.Context(), post); err != nil {
		app.internalServerError(w, r, err)
		return
	}

	app.OutputJSON(w, http.StatusOK, post)
}

func (app *application) updatePostHandler(w http.ResponseWriter, r *http.Request) {
	post := getPostFromCtx(r)
	var payload UpdatePostPayload

	if err := ReadJSON(w, r, &payload); err != nil {
		app.badRequestError(w, r, err)
		return
	}

	if err := Validate.Struct(payload); err != nil {
		app.badRequestError(w, r, err)
		return
	}

	// Optimistic Concurrency Control - check version mismatch
	if payload.Version != 0 && payload.Version != post.Version {
		app.conflictError(w, r, "VERSION_MISMATCH", fmt.Errorf("version mismatch"))
		return
	}

	if payload.Title != nil {
		post.Title = *payload.Title
	}

	if payload.Content != nil {
		post.Content = *payload.Content
		mentions := extractMentions(*payload.Content)

		validMentions, err := app.storage.User.ValidateUsername(r.Context(), mentions)
		if err != nil {
			app.notFoundError(w, r, err)
			return
		}
		post.Mentions = validMentions
	}

	if payload.Tags != nil {
		post.Tags = *payload.Tags
	}

	if payload.ImagesPath != nil {
		post.ImagesPath = *payload.ImagesPath
	}

	post.Version++

	if err := app.storage.Post.Update(r.Context(), post); err != nil {
		app.internalServerError(w, r, err)
		return
	}

	app.OutputJSON(w, http.StatusCreated, post)
}

func (app *application) toggleLikePostHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromCtx(r)
	post := getPostFromCtx(r)

	liked, err := app.storage.Post.ToggleLike(r.Context(), user.ID, post)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	app.OutputJSON(w, http.StatusCreated, liked)
}

func (app *application) deletePostHandler(w http.ResponseWriter, r *http.Request) {
	postID := chi.URLParam(r, "postID")
	ctx := r.Context()

	err := app.storage.Post.Delete(ctx, postID)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
