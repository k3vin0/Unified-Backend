package model

import "go.mongodb.org/mongo-driver/bson/primitive"

type IngredientsResult = []Ingredient
type RecipeResult = []RecipeReturnType

// Ingredient represents the data structure for an ingredient in the database.
type Ingredient struct {
	ObjectID primitive.ObjectID `bson:"_id,omitempty"` // Use `omitempty` to ignore empty values during marshalling and to allow MongoDB to auto-generate the ID.
	Name     string             `bson:"name"`
	Calories int                `bson:"calories_per_gram"`
}

// IngredientIDType to match the incoming JSON structure for ingredients.
type IngredientIDType struct {
	ObjectID string `json:"ObjectID"`
}

type RecipeReturnType struct {
	Name string             `bson:"name"`
	ID   []IngredientIDType `bson:"ingredients"`
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
