package config

// TODO: Implement configuration logic

import (
	"context"
	"log"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func Config(ctx context.Context) (*mongo.Client, error) {
	serverAPI := options.ServerAPI(options.ServerAPIVersion1)

	opts := options.Client().ApplyURI("mongodb+srv://kevinoagyemang:rFVaaH33lUS7YzWW@test.iuemrsv.mongodb.net/?retryWrites=true&w=majority&appName=test").SetServerAPIOptions(serverAPI)

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
