package cache

import (
	"dynamicrecipes/pkg/model"
	"sync"
)

var recipeCache sync.Map

func InvalidateRecipesCache(cacheKey string) {
	// Deletes the entry for a key.
	recipeCache.Delete(cacheKey)
}

func LoadRecipesCache(cacheKey string) (*[]model.Recipe, bool) {
	// Try to load the cache value using the provided key.
	cached, ok := recipeCache.Load(cacheKey)
	if !ok {
		// The key was not found in the cache, return nil and false.
		return nil, false
	}
	// The key was found, assert the type to *[]model.Recipe and return it with true.
	return cached.(*[]model.Recipe), true
}

func StoreRecipesInCache(cacheKey string, value any) {
	recipeCache.Store(cacheKey, value)
}
