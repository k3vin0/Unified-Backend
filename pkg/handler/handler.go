package handler

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type Result = []Ingredients

type Ingredients struct {
	Name     string `bson:"name"`
	Calories int    `bson:"calories_per_gram"`
}

// TODO: Implement HTTP handlers
func Handler(client *mongo.Client) {
	e := echo.New()
	e.Use(middleware.Logger())

	e.GET("/ingredients", func(c echo.Context) error {
		collection := client.Database("Recipe_Service").Collection("Ingredients")
		filter := bson.D{{}}

		cur, err := collection.Find(context.TODO(), filter)
		if err != nil {
			log.Fatal(err) // Consider how to handle errors more gracefully
		}
		defer cur.Close(context.TODO())

		var results Result
		if err = cur.All(context.TODO(), &results); err != nil {
			log.Fatal(err)
		}

		for _, ingredient := range results {

			fmt.Printf("Name: %s, Calories: %d\n", ingredient.Name, ingredient.Calories)
		}
		return c.JSON(200, results)
	})

	e.POST("/ingredients", func(c echo.Context) error {
		var newIngredients []Ingredients // Assuming Ingredient is your struct type for the collection

		// Bind the request body to newIngredients slice
		if err := c.Bind(&newIngredients); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Invalid input")
		}

		// Prepare a slice of interface{} to hold the documents for insertion
		var docs []interface{}
		for _, ingredient := range newIngredients {
			docs = append(docs, ingredient)
		}

		// Inserting the documents into the collection
		collection := client.Database("Recipe_Service").Collection("Ingredients")
		result, err := collection.InsertMany(context.TODO(), docs)
		if err != nil {
			// Handle error appropriately
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to insert ingredients")
		}

		// Respond with the result of the insert operation
		return c.JSON(http.StatusCreated, result.InsertedIDs)
	})

	e.DELETE("/ingredients/:name", func(c echo.Context) error {
		name := c.Param("name")

		filter := bson.M{"name": name}

		collection := client.Database("Recipe_Service").Collection("Ingredients")
		result, err := collection.DeleteOne(context.TODO(), filter)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Could not delete ingredient")
		}
		if result.DeletedCount == 0 {
			// No document was found with the provided name
			return echo.NewHTTPError(http.StatusNotFound, "No ingredient found with the given name")
		}

		fmt.Print(result)
		return c.JSON(http.StatusOK, map[string]interface{}{
			"message": "Ingredient successfully deleted",
			"name":    name,
		})
	})

	go func() {
		if err := e.Start(":42069"); err != nil {
			e.Logger.Fatal("Error starting Echo server:", err)
		}
	}()

	// Set up channel on which to receive SIGINT signals for graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	// Wait for interrupt signal to gracefully shut down the server with a timeout of 10 seconds.
	<-quit
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := e.Shutdown(ctx); err != nil {
		e.Logger.Fatal(err)
	}

	e.Logger.Info("Server gracefully stopped")
}
