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
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type IngredientsResult = []Ingredient
type RecipeResult = []RecipeReturnType

type Ingredient struct {
	ObjectID primitive.ObjectID `bson:"_id"`
	Name     string             `bson:"name"`
	Calories int                `bson:"calories_per_gram"`
}

type RecipeReturnType struct {
	Name string             `bson:"name"`
	ID   []IngredientIDType `bson:"ingredients"`
}

// IngredientIDType to match the incoming JSON structure for ingredients.
type IngredientIDType struct {
	ObjectID string `json:"ObjectID"`
}

// RecipePostType adjusted to include a slice of IngredientIDType.
type RecipePostType struct {
	Name        string             `json:"Name"`
	Ingredients []IngredientIDType `json:"Ingredients"`
}

type Recipe struct {
	Name        string
	Ingredients []Ingredient
}

func getIngredientByID(client *mongo.Client, ingredientID string) (*Ingredient, error) {
	collection := client.Database("Recipe_Service").Collection("Ingredients")
	objID, err := primitive.ObjectIDFromHex(ingredientID)

	if err != nil {
		return nil, err
	}

	filter := bson.D{{Key: "_id", Value: objID}}

	var ingredient Ingredient
	err = collection.FindOne(context.TODO(), filter).Decode(&ingredient)
	if err != nil {
		return nil, err
	}

	return &ingredient, nil
}

func getAllRecipes(client *mongo.Client) (*[]Recipe, error) {
	collection := client.Database("Recipe_Service").Collection("recipes")
	filter := bson.D{{}}

	cur, err := collection.Find(context.TODO(), filter)
	if err != nil {
		return nil, err // Consider how to handle errors more gracefully
	}
	defer cur.Close(context.TODO())

	var results RecipeResult
	if err = cur.All(context.TODO(), &results); err != nil {
		return nil, err
	}
	var ingredientsResponse []Ingredient
	var finalReturnValue []Recipe = make([]Recipe, 0)
	for _, ingredient := range results {
		for _, id := range ingredient.ID {

			ingredient, err := getIngredientByID(client, id.ObjectID)
			if err != nil {
				return nil, err
			}
			ingredientsResponse = append(ingredientsResponse, *ingredient)
		}
		finalReturnValue = append(finalReturnValue, Recipe{Name: ingredient.Name, Ingredients: ingredientsResponse})

		fmt.Printf("Name: %s, Calories: %v\n", ingredient.Name, ingredient.ID)
	}

	return &finalReturnValue, err
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

		var results IngredientsResult
		if err = cur.All(context.TODO(), &results); err != nil {
			log.Fatal(err)
		}

		fmt.Print(results)

		for _, ingredient := range results {
			fmt.Printf("Name: %s, Ingredient: %d\n", ingredient.Name, ingredient.Calories)
		}
		return c.JSON(200, results)
	})

	e.GET("/ingredient", func(c echo.Context) error {
		idStr := c.QueryParam("id") // example ObjectId as a string

		if idStr == "" {
			return echo.NewHTTPError(http.StatusBadRequest, "No params provided")
		}

		ingredient, err := getIngredientByID(client, idStr)

		if err != nil {
			echo.NewHTTPError(http.StatusInternalServerError, "No Ingredient with specified id ")
		}

		return c.JSON(http.StatusOK, ingredient)

	})

	e.GET("/recipes", func(c echo.Context) error {

		result, err := getAllRecipes(client)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "unable to fetch recipes")
		}

		return c.JSON(200, result)
	})

	e.POST("/ingredients", func(c echo.Context) error {
		var newIngredients []Ingredient // Assuming Ingredient is your struct type for the collection

		// Bind the request body to newIngredients slice
		if err := c.Bind(&newIngredients); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Invalid input")
		}

		// Prepare a slice of interface{} to hold the documents for insertion
		var docs []interface{}
		for _, ingredient := range newIngredients {
			// fmt.Print(ingredient.ObjectID)
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

	e.POST("/recipes", func(c echo.Context) error {
		var newRecipes []RecipePostType // Assuming Recipes is your struct type for the collection

		// Bind the request body to newRecipes slice
		if err := c.Bind(&newRecipes); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Invalid input")
		}

		// Prepare a slice of interface{} to hold the documents for insertion
		var docs []interface{}
		for _, recipe := range newRecipes {
			// Convert IngredientIDType to primitive.ObjectID
			for i, ingredient := range recipe.Ingredients {
				if oid, err := primitive.ObjectIDFromHex(ingredient.ObjectID); err == nil {
					recipe.Ingredients[i] = IngredientIDType{ObjectID: oid.Hex()}
				} else {
					// Handle error if conversion fails
					return echo.NewHTTPError(http.StatusBadRequest, "Invalid ObjectID in Ingredients")
				}
			}
			docs = append(docs, recipe)
		}

		// Inserting the documents into the collection
		collection := client.Database("Recipe_Service").Collection("recipes")
		result, err := collection.InsertMany(context.TODO(), docs)
		if err != nil {
			// Handle error appropriately
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to insert recipes")
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
	e.DELETE("/recipes/:id", func(c echo.Context) error {
		id := c.Param("id")
		objID, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Could not convert hex to object ID")
		}

		filter := bson.D{{Key: "_id", Value: objID}}

		collection := client.Database("Recipe_Service").Collection("recipes")
		result, err := collection.DeleteOne(context.TODO(), filter)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Could not delete ingredient")
		}
		if result.DeletedCount == 0 {
			// No document was found with the provided name
			return echo.NewHTTPError(http.StatusNotFound, "No ingredient found with the given Object Id")
		}

		return c.JSON(http.StatusOK, map[string]interface{}{
			"message": "Ingredient successfully deleted",
			"id":      id,
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
