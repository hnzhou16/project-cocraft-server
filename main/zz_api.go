package main

import (
	"errors"
	"github.com/hnzhou16/project-social/internal/ai"
	"github.com/hnzhou16/project-social/internal/aws"
	"github.com/hnzhou16/project-social/internal/security"
	"go.uber.org/zap"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/hnzhou16/project-social/internal/auth"
	"github.com/hnzhou16/project-social/internal/mailer"
	"github.com/hnzhou16/project-social/internal/storage"
)

// zz_api.go is named to ensure it's compiled last.
// This allows all handler functions (defined in other files) to be available.

type application struct {
	config        config
	storage       storage.Collection
	logger        *zap.SugaredLogger
	mailer        mailer.Client
	authenticator auth.UserAuthenticator
	awsPresigner  *aws.Presigner
	aiImage       ai.Client
}

type config struct {
	addr       string
	env        string
	version    string
	dbConfig   dbConfig
	mailConfig mailConfig
	authConfig authConfig
	awsConfig  awsConfig
	aiConfig   aiConfig
}

type dbConfig struct {
	uri             string
	dbName          string
	maxPoolSize     uint64
	minPoolSize     uint64
	maxConnIdleTime time.Duration
	maxConnTimeOut  time.Duration
}

type mailConfig struct {
	apiKey        string
	fromEmail     string
	activationURL string
	exp           time.Duration
}

type authConfig struct {
	secret string
	exp    time.Duration
	iss    string
}

type awsConfig struct {
	accessKey       string
	secretAccessKey string
	region          string
	s3Bucket        string
	exp             time.Duration
}

type aiConfig struct {
	apiKey      string
	apiUrl      string
	imageNumber int
	imageSize   string
}

func (app *application) mount() *chi.Mux {
	// mux is returned in chi
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// timeout request context
	r.Use(middleware.Timeout(60 * time.Second))

	r.Get("/health", app.healthCheckHandler)

	// authentication
	r.Route("/authentication", func(r chi.Router) {
		r.Post("/user", app.registerUserHandler)
		r.Post("/token", app.createTokenHandler)
	})

	// user
	r.Route("/user", func(r chi.Router) {
		r.Put("/activate/{token}", app.activateUserHandler)

		r.Route("/admin", func(r chi.Router) {
			r.Use(app.authCtxMiddleware)
			r.Use(app.RequirePermission(security.PermAdmin))

			r.Get("/", app.getAllUsersHandler)
		})

		r.Route("/{userID}", func(r chi.Router) {
			r.Use(app.authCtxMiddleware)

			r.Get("/", app.getUserHandler)

			// follow/unfollow
			r.Group(func(r chi.Router) {
				r.Use(app.RequirePermission(security.PermUser))

				r.Get("/following", app.getFollowingUserHandler)
				r.Get("/follow-status", app.followStatusHandler)
				r.Post("/follow", app.followUserHandler)
				r.Delete("/follow", app.unfollowUserHandler)
			})

			// generate images
			r.Post("/generate-image", app.generateImageHandler)

			// upload images
			r.With(app.RequirePermission(security.PermUser)).
				Post("/upload-image", app.generateUploadURLHandler)
		})
	})

	// feed
	r.Route("/feed", func(r chi.Router) {
		r.Get("/public", app.getPublicFeedHandler)

		r.Group(func(r chi.Router) {
			r.Use(app.authCtxMiddleware)
			r.Use(app.RequirePermission(security.PermUser))

			r.Get("/", app.getFeedHandler)
		})
	})

	// post
	r.Route("/post", func(r chi.Router) {
		r.Use(app.authCtxMiddleware)
		r.Use(app.RequirePermission(security.PermUser))
		r.Post("/", app.createPostHandler)
		// add '/user', otherwise chi will continue to confuse {userID} with {postID} in the following routes
		r.Get("/user/{userID}", app.getAllUserPostsHandler)

		r.Route("/{postID}", func(r chi.Router) {
			r.Use(app.postCtxtMiddleware)

			r.Get("/", app.getPostHandler)

			r.Group(func(r chi.Router) {
				r.Use(app.RequirePostOwnership)

				r.Patch("/", app.updatePostHandler)
				r.Delete("/", app.deletePostHandler)
			})

			// like
			r.Patch("/like", app.toggleLikePostHandler)

			// comment
			r.Route("/comment", func(r chi.Router) {
				r.Get("/", app.getCommentHandler)

				r.With(app.RequirePermission(security.PermUser)).
					Post("/", app.createCommentHandler)
			})
		})
	})

	// review
	r.Route("/review", func(r chi.Router) {
		r.Use(app.authCtxMiddleware)
		r.Use(app.RequirePermission(security.PermUser))

		r.Post("/create-review", app.createReviewHandler)
		r.Route("/{reviewID}", func(r chi.Router) {
			r.Delete("/delete-review", app.deleteReviewHandler)
		})

	})

	return r
}

func (app *application) run(mux *chi.Mux) *http.Server {

	srv := &http.Server{
		Addr:         app.config.addr,
		Handler:      mux,
		WriteTimeout: 30 * time.Second,
		ReadTimeout:  10 * time.Second,
		IdleTimeout:  time.Minute,
	}

	app.logger.Infow("server started", "addr", app.config.addr, "env", app.config.env)

	// Start server in a goroutine to allow graceful shutdown
	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			app.logger.Fatal("❌ Server failed: %v", err)
		}
	}()

	return srv
}
