package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/go-chi/chi/v5"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/hnzhou16/project-social/internal/security"
	"github.com/hnzhou16/project-social/internal/storage"
)

type ctxKey string

const (
	userCtx ctxKey = "user"
	postCtx ctxKey = "post"
)

// Middleware wraps an HTTP handler, modifying the request(r) or response(w) before passing control to next handler

// authCtxMiddleware - validate token and add user to ctx
func (app *application) authCtxMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// validate header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			app.unauthorizedError(w, r, fmt.Errorf("missing Authorization header"))
			return
		}

		// extract token
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			app.unauthorizedError(w, r, fmt.Errorf("invalid Authorization header"))
			return
		}

		tokenString := parts[1]
		jwtToken, err := app.authenticator.VerifyToken(tokenString)
		if err != nil {
			app.unauthorizedError(w, r, err)
			return
		}

		// fetch user
		claims, _ := jwtToken.Claims.(jwt.MapClaims)

		userID, ok := claims["sub"].(string)
		if !ok || userID == "" {
			app.unauthorizedError(w, r, err)
			return
		}

		ctx := r.Context()
		user, err := app.storage.User.GetByID(ctx, userID)
		if err != nil {
			app.unauthorizedError(w, r, err)
			return
		}

		// put user in ctx
		ctx = context.WithValue(ctx, userCtx, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// postCtxtMiddleware - add post to ctx
func (app *application) postCtxtMiddleware(next http.Handler) http.Handler {
	// return a new handler that wraps the original next handler
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		postID := chi.URLParam(r, "postID")
		ctx := r.Context()

		post, err := app.storage.Post.GetByID(ctx, postID)
		if err != nil {
			switch {
			case errors.Is(err, storage.ErrPostNotFound):
				app.notFoundError(w, r, err)
				return
			default:
				app.internalServerError(w, r, err)
				return
			}
		}

		ctx = context.WithValue(ctx, postCtx, post)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequirePermission return type is a middleware function -> func(http.Handler) http.Handler
func (app *application) RequirePermission(required security.Permission) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user := getUserFromCtx(r)

			if !security.HasPermission(user.Role, required) {
				app.forbiddenError(w, r, fmt.Errorf("role %s does not have required permission", user.Role))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func (app *application) RequirePostOwnership(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := getUserFromCtx(r)
		post := getPostFromCtx(r)

		if user.ID != post.UserID {
			app.forbiddenError(w, r, fmt.Errorf("user does not belong to post"))
			return
		}

		next.ServeHTTP(w, r)
	})
}

func getUserFromCtx(r *http.Request) *storage.User {
	user, _ := r.Context().Value(userCtx).(*storage.User)
	return user
}

func getPostFromCtx(r *http.Request) *storage.Post {
	post, _ := r.Context().Value(postCtx).(*storage.Post)
	return post
}
