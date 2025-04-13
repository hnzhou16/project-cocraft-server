package main

import (
	"context"
	"github.com/hnzhou16/project-cocraft-server/internal/ai"
	"go.uber.org/zap"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hnzhou16/project-cocraft-server/internal/auth"
	"github.com/hnzhou16/project-cocraft-server/internal/aws"
	"github.com/hnzhou16/project-cocraft-server/internal/db"
	"github.com/hnzhou16/project-cocraft-server/internal/env"
	"github.com/hnzhou16/project-cocraft-server/internal/mailer"
	"github.com/hnzhou16/project-cocraft-server/internal/storage"
	"github.com/lpernett/godotenv"
)

func main() {
	// load .env file
	if err := godotenv.Load(); err != nil {
		log.Println("⚠️ Error loading .env file")
	}

	// initialize app config
	cfg := config{
		addr:    env.GetString("ADDR", ":8080"),
		env:     env.GetString("ENV", "development"),
		version: env.GetString("VERSION", "1.0.0"),
		dbConfig: dbConfig{
			uri:             env.GetString("MONGODB_URI", ""),
			dbName:          env.GetString("MONGODB_NAME", ""),
			maxPoolSize:     uint64(env.GetInt("MONGODB_MAX_POOL_SIZE", 30)),
			minPoolSize:     uint64(env.GetInt("DB_MIN_POOL_SIZE", 10)),
			maxConnIdleTime: time.Duration(env.GetInt("DB_MAX_CONN_IDLE_TIME", 10)) * time.Second,
			maxConnTimeOut:  time.Duration(env.GetInt("DB_CONN_TIME_OUT", 10)) * time.Second,
		},
		mailConfig: mailConfig{
			apiKey:        env.GetString("SENDGRID_API_KEY", ""),
			fromEmail:     env.GetString("FROM_EMAIL", ""),
			activationURL: env.GetString("ACTIVATION_URL", "http://localhost:3000/activate"),
			exp:           time.Hour * 24 * 3,
		},
		authConfig: authConfig{
			secret: env.GetString("AUTH_TOKEN_SECRET", ""),
			exp:    time.Hour * 24 * 3,
			iss:    env.GetString("AUTH_TOKEN_ISS", ""),
		},
		awsConfig: awsConfig{
			accessKey:       env.GetString("AWS_ACCESS_KEY", ""),
			secretAccessKey: env.GetString("AWS_SECRET_ACCESS_KEY", ""),
			region:          env.GetString("AWS_REGION", ""),
			s3Bucket:        env.GetString("S3_BUCKET", ""),
			exp:             time.Minute * 5,
		},
		aiConfig: aiConfig{
			apiKey:      env.GetString("OPENAI_KEY", ""),
			apiUrl:      env.GetString("API_URL", ""),
			imageNumber: 1,
			imageSize:   "1024x1024",
		},
	}

	// initialize logger
	logger := zap.Must(zap.NewProduction()).Sugar()
	defer logger.Sync()

	// connect to db
	dbConn, err := db.Connect(
		cfg.dbConfig.uri,
		cfg.dbConfig.dbName,
		cfg.dbConfig.maxPoolSize,
		cfg.dbConfig.minPoolSize,
		cfg.dbConfig.maxConnIdleTime,
		cfg.dbConfig.maxConnTimeOut,
	)
	if err != nil {
		logger.Fatal("❌ Error connecting to database: %v", err)
	}
	defer dbConn.Disconnect()
	logger.Info("✅ Connected to MongoDB!")

	// Initialize MongoDB collections
	s := storage.NewMongoDBCollections(dbConn)

	// Initialize Mailer
	mailerSendgrid := mailer.NewSendgrid(cfg.mailConfig.apiKey, cfg.mailConfig.fromEmail)

	// Initialize Authenticator
	jwtAuthenticator := auth.NewJWTAuthenticator(cfg.authConfig.secret, cfg.authConfig.iss)

	// Initialize AWS
	awsPresigner, err := aws.NewPresigner(cfg.awsConfig.accessKey, cfg.awsConfig.secretAccessKey, cfg.awsConfig.region, cfg.awsConfig.s3Bucket)
	if err != nil {
		logger.Fatal("❌ failed to initialize AWS config: %v", err)
	}

	// Initialize OpenAI
	openAIImage := ai.NewImageGenerator(cfg.aiConfig.apiKey, cfg.aiConfig.apiUrl, cfg.aiConfig.imageSize, cfg.aiConfig.imageNumber)

	// Initialize app
	app := &application{
		config:        cfg,
		storage:       s,
		logger:        logger,
		mailer:        mailerSendgrid,
		authenticator: jwtAuthenticator,
		awsPresigner:  awsPresigner,
		aiImage:       openAIImage,
	}

	// Create the server
	mux := app.mount()

	// Start the server in a goroutine
	server := app.run(mux)

	// Gracefully shutdown
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	<-shutdown
	logger.Info("🛑 Server shutting down...")

	// Gracefully shut down the server with a timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Info("⚠️ Error during server shutdown: %v", err)
	}

	logger.Info("✅ Server gracefully stopped.")
}
