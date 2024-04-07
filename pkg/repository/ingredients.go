package repository

import (
	"context"
	"dynamicrecipes/pkg/model"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// IngredientRepository handles database operations related to ingredients.
type IngredientRepository struct {
	client *mongo.Client
}

// NewIngredientRepository creates a new IngredientRepository.
func NewIngredientRepository(client *mongo.Client) *IngredientRepository {
	return &IngredientRepository{client: client}
}

// FindByID finds an ingredient by its ID.
func (r *IngredientRepository) FindByID(ctx context.Context, ingredientID string) (*model.Ingredient, error) {
	collection := r.client.Database("Recipe_Service").Collection("Ingredients")
	objID, err := primitive.ObjectIDFromHex(ingredientID)
	if err != nil {
		return nil, fmt.Errorf("invalid ingredient ID: %w", err)
	}

	filter := bson.D{{Key: "_id", Value: objID}}
	var ingredient model.Ingredient
	if err := collection.FindOne(ctx, filter).Decode(&ingredient); err != nil {
		return nil, fmt.Errorf("failed to find ingredient: %w", err)
	}
	return &ingredient, nil
}

func (r *IngredientRepository) DeleteByName(ctx context.Context, ingredientName string) (*mongo.DeleteResult, error) {
	collection := r.client.Database("Recipe_Service").Collection("Ingredients")

	filter := bson.M{"name": ingredientName}

	result, err := collection.DeleteOne(ctx, filter)

	return result, err
}
