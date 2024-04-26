package main

import (
	"context"
	"dynamicrecipes/pkg/config"
	"dynamicrecipes/pkg/handler"
	"log"
	"os/signal"
	"syscall"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// CreateMongoClient initializes a new MongoDB client and returns it.
func CreateMongoClient(ctx context.Context, opts *options.ClientOptions) (*mongo.Client, error) {
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
func main() {
	// TODO: Implement
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	mongoClient, err := config.Config(ctx)
	if err != nil {
		log.Fatalf("Error: %s", err)
	}

	handler.Handler(mongoClient)
	// Now you can use the client for database operations

	// Don't forget to disconnect the client when you're done
	<-ctx.Done()
	// Gracefully shutdown other services if needed
	stop()
	_ = mongoClient.Disconnect(context.Background())
}
