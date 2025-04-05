package db

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type DBConnection struct {
	Client *mongo.Client
	DB     *mongo.Database
}

// Connect setS MongoDB connection to DB
func Connect(uri, dbName string, maxPoolSize, minPoolSize uint64, maxConnIdleTime, maxConnTimeOut time.Duration) (*DBConnection, error) {
	if uri == "" || dbName == "" {
		return nil, fmt.Errorf("❌ Missing MongoDB URI or database name")
	}

	clientOptions := options.Client().
		ApplyURI(uri).
		SetMaxPoolSize(maxPoolSize).
		SetMinPoolSize(minPoolSize).
		SetMaxConnIdleTime(maxConnIdleTime).
		SetConnectTimeout(maxConnTimeOut)

	client, err := mongo.Connect(context.TODO(), clientOptions)
	if err != nil {
		return nil, fmt.Errorf("❌ Failed to connect to MongoDB: %w", err)
	}

	// verify connection with a Ping
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err = client.Ping(ctx, nil); err != nil {
		return nil, fmt.Errorf("❌ Failed to ping MongoDB: %w", err)
	}

	return &DBConnection{
		Client: client,
		DB:     client.Database(dbName),
	}, nil
}

// GetCollection returns a MongoDB collection
func (conn *DBConnection) GetCollection(collectionName string) *mongo.Collection {
	if conn.DB == nil {
		log.Fatalf("❌ Database is not initialized")
	}
	return conn.DB.Collection(collectionName)
}

func (conn *DBConnection) Disconnect() {
	if conn.Client != nil {
		err := conn.Client.Disconnect(context.TODO())
		if err != nil {
			log.Printf("⚠️ Failed to disconnect MongoDB: %v\n", err)
		} else {
			log.Println("✅ Disconnected from MongoDB!")
		}
	}
}
