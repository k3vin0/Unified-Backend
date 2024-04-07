package cache

import (
	"dynamicrecipes/pkg/model"
	"sync"
)

var ingredientsCache sync.Map

func InvalidateIngredientsCache(cacheKey string) {
	// Deletes the entry for a key.
	ingredientsCache.Delete(cacheKey)
}

func LoadIngredientsCache(cacheKey string) (*[]model.Ingredient, bool) {
	// Try to load the cache value using the provided key.
	cached, ok := ingredientsCache.Load(cacheKey)
	if !ok {
		// The key was not found in the cache, return nil and false.
		return nil, false
	}
	// The key was found, assert the type to *[]model.Recipe and return it with true.
	return cached.(*[]model.Ingredient), true
}

func StoreIngredientsInCache(cacheKey string, value any) {
	recipeCache.Store(cacheKey, value)
}
