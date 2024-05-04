package main

import (
	"context"
	"dynamicrecipes/pkg/config"
	"dynamicrecipes/pkg/handler"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/labstack/echo/v4"
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
	e := echo.New() // Initialize a new Echo instance

	// Context to handle signals for graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Configuration and MongoDB client setup
	mongoClient, err := config.Config(ctx) // Assuming this is a function that configures and connects a MongoDB client
	if err != nil {
		log.Fatalf("Error: %s", err)
	}

	handler.InitRoutes(e, mongoClient) // Setup your routes, assuming you have a function to do this

	go handler.StartBroadcasting()

	// Start server in a goroutine to allow it to run concurrently with the graceful shutdown logic
	go func() {
		if err := e.Start(":42069"); err != nil {
			e.Logger.Fatal("Error starting Echo server:", err)
		}
	}()

	// Set up channel on which to receive SIGINT signals for graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	// Wait for interrupt signal to gracefully shut down the server with a timeout of 10 seconds
	<-quit
	shutdownCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := e.Shutdown(shutdownCtx); err != nil {
		e.Logger.Fatal("Server shutdown failed:", err)
	}
	e.Logger.Info("Server gracefully stopped")

	// Don't forget to disconnect the MongoDB client
	if err := mongoClient.Disconnect(ctx); err != nil {
		log.Printf("Error disconnecting MongoDB: %v", err)
	}
}
