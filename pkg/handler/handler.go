package handler

import (
	"context"
	"dynamicrecipes/pkg/cache"
	"dynamicrecipes/pkg/model"
	"dynamicrecipes/pkg/repository"
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
	"net/url"
	"os"
	"strings"
	"sync"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

func getAllRecipes(client *mongo.Client) (*[]model.Recipe, error) {
	if cachedRecipes, ok := cache.LoadRecipesCache("allRecipes"); ok {
		return cachedRecipes, nil
	}
	collection := client.Database("Recipe_Service").Collection("recipes")
	filter := bson.D{{}}

	cur, err := collection.Find(context.TODO(), filter)
	if err != nil {
		return nil, err
	}
	defer cur.Close(context.TODO())

	var results model.RecipeResult
	if err = cur.All(context.TODO(), &results); err != nil {
		return nil, err
	}

	// Prepare a channel to collect errors that might occur in goroutines.
	errChan := make(chan error, 1)
	// Prepare a wait group to synchronize all goroutines.
	var wg sync.WaitGroup

	finalReturnValue := make([]model.Recipe, len(results))
	ingredientRepo := repository.NewIngredientRepository(client)

	for i, recipeItem := range results {
		wg.Add(1) // Increment the WaitGroup counter.
		go func(i int, recipeItem model.RecipeReturnType) {
			defer wg.Done() // Decrement the counter when the goroutine completes.

			var ingredientsResponse []model.Ingredient
			for _, id := range recipeItem.ID {
				ingredient, err := ingredientRepo.FindByID(context.TODO(), id.ObjectID)
				if err != nil {
					errChan <- err // Send any error that occurs to the error channel.
					return
				}
				ingredientsResponse = append(ingredientsResponse, *ingredient)
			}

			finalReturnValue[i] = model.Recipe{Name: recipeItem.Name, Ingredients: ingredientsResponse}
		}(i, recipeItem)
	}

	// Wait for all goroutines to finish.
	wg.Wait()
	close(errChan) // Close the error channel.

	// Check if any errors were reported by the goroutines.
	for err := range errChan {
		if err != nil {
			return nil, err
		}
	}
	cache.StoreRecipesInCache("allRecipes", &finalReturnValue)

	return &finalReturnValue, nil
}

// TODO: Implement HTTP handlers
func InitRoutes(e *echo.Echo, client *mongo.Client) {
	corsUrls := os.Getenv("LOCAL_CORS_URLS")
	allowOrigins := strings.Split(corsUrls, ",")
	e.Use(middleware.Logger())
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: allowOrigins, // Be cautious with *, specify origins if possible
		AllowMethods: []string{http.MethodGet, http.MethodPut, http.MethodPost, http.MethodDelete},
	}))

	e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Skipper: func(c echo.Context) bool {
			// Don't log if the request is for the WebSocket endpoint
			return c.Path() == "/ws"
		},
	}))

	e.GET("/ingredients", func(c echo.Context) error {
		if cachedIngredients, ok := cache.LoadIngredientsCache("allIngredients"); ok {
			return c.JSON(http.StatusOK, cachedIngredients)
		}
		collection := client.Database("Recipe_Service").Collection("Ingredients")
		filter := bson.D{{}}

		cur, err := collection.Find(context.TODO(), filter)
		if err != nil {
			log.Fatal(err) // Consider how to handle errors more gracefully
		}
		defer cur.Close(context.TODO())

		var results model.IngredientsResult
		if err = cur.All(context.TODO(), &results); err != nil {
			log.Fatal(err)
		}
		cache.StoreIngredientsInCache("allIngredients", &results)

		// for _, ingredient := range results {
		// 	fmt.Printf("Name: %s, Calories: %d\n", ingredient.Name, ingredient.Calories)
		// }
		return c.JSON(http.StatusOK, results)
	})

	e.GET("/ingredient", func(c echo.Context) error {
		idStr := c.QueryParam("id") // example ObjectId as a string

		if idStr == "" {
			return echo.NewHTTPError(http.StatusBadRequest, "No params provided")
		}

		ingredientsRepo := repository.NewIngredientRepository(client)

		ingredient, err := ingredientsRepo.FindByID(context.TODO(), idStr)

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
		type Ingredient struct {
			Name     string `bson:"name"`
			Calories int    `bson:"calories_per_gram"`
		}
		var newIngredients []Ingredient // Assuming Ingredient is your struct type for the collection
		// fmt.Print(c)
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
			fmt.Print((err))
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to insert ingredients")
		}

		cache.InvalidateIngredientsCache("allIngredients")
		// Respond with the result of the insert operation
		return c.JSON(http.StatusCreated, result.InsertedIDs)
	})

	e.POST("/recipes", func(c echo.Context) error {
		var newRecipes []model.RecipePostType // Assuming Recipes is your struct type for the collection

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
					recipe.Ingredients[i] = model.IngredientIDType{ObjectID: oid.Hex()}
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
		cache.InvalidateRecipesCache("allRecipes")
		// Respond with the result of the insert operation
		return c.JSON(http.StatusCreated, result.InsertedIDs)
	})
	e.DELETE("/ingredients/:name", func(c echo.Context) error {
		name := c.Param("name")
		decodedParam, err := url.QueryUnescape(name)
		if err != nil {
			// handle the error
			return echo.NewHTTPError(http.StatusBadRequest, "Invalid search parameter")
		}

		ingredientsRepository := repository.NewIngredientRepository(client)

		result, err := ingredientsRepository.DeleteByName(context.TODO(), decodedParam)

		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Could not delete ingredient")
		}

		if result.DeletedCount == 0 {
			// No document was found with the provided name
			return echo.NewHTTPError(http.StatusNotFound, "No ingredient found with the given name")
		}

		cache.InvalidateIngredientsCache("allIngredients")
		return c.JSON(http.StatusOK, map[string]interface{}{
			"message": "Ingredient successfully deleted",
			"name":    decodedParam,
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
		cache.InvalidateRecipesCache("allRecipes")
		return c.JSON(http.StatusOK, map[string]interface{}{
			"message": "Ingredient successfully deleted",
			"id":      id,
		})
	})

	e.PUT("/ingredients/:id", func(c echo.Context) error {
		// Extract the ingredient ID from the URL parameter.
		id := c.Param("id")

		// Define a struct for the request body. Here, we allow either field to be updated.
		type updateRequest struct {
			Name     *string `json:"name,omitempty"`
			Calories *int    `json:"calories,omitempty"`
		}
		var updateData updateRequest

		// Bind the request body to the struct.
		if err := c.Bind(&updateData); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Invalid input")
		}

		// Create an update document based on the provided data.
		update := bson.M{}
		if updateData.Name != nil {
			update["name"] = *updateData.Name
		}
		if updateData.Calories != nil {
			update["calories_per_gram"] = *updateData.Calories
		}

		// Get the repository and perform the update.
		ingredientsRepository := repository.NewIngredientRepository(client)
		updatedIngredient, err := ingredientsRepository.UpdateByID(context.TODO(), id, update)

		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Could not update ingredient")
		}

		if updatedIngredient == nil {
			// No document was found with the provided ID.
			return echo.NewHTTPError(http.StatusNotFound, "No ingredient found with the given ID")
		}

		cache.InvalidateIngredientsCache("allIngredients")

		// Return the updated ingredient and a success message.
		return c.JSON(http.StatusOK, map[string]interface{}{
			"message":    "Ingredient successfully updated",
			"ingredient": updatedIngredient,
		})
	})

	e.GET("/ws", HandleWebSocketConnection)

	// e.GET("/debug/pprof/*", echo.WrapHandler(http.DefaultServeMux))

}
