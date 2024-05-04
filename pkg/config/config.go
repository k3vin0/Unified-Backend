package config

// TODO: Implement configuration logic

import (
	"context"
	"log"
	"os"

	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func Config(ctx context.Context) (*mongo.Client, error) {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
		return nil, err
	}

	// Get the value of the environment variable
	connectionString := os.Getenv("MONGODB_URI_STRING")

	serverAPI := options.ServerAPI(options.ServerAPIVersion1)

	opts := options.Client().ApplyURI(connectionString).SetServerAPIOptions(serverAPI)

	client, err := mongo.Connect(ctx, opts)
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
		return nil, err
	}

	// Optional: Confirm the connection is successful
	err = client.Ping(ctx, nil)
	if err != nil {
		_ = client.Disconnect(ctx)
		log.Fatalf("Failed to ping MongoDB: %v", err)
		return nil, err
	}

	return client, nil

}
