package cache

import (
	"encoding/json"
	"gosynctasks/backend"
	"os"
	"path/filepath"
	"time"
)

// CachedData represents the structure of cached task lists
type CachedData struct {
	Lists     []backend.TaskList `json:"lists"`
	Timestamp int64              `json:"timestamp"`
}

// GetCacheDir returns the XDG-compliant cache directory path
func GetCacheDir() (string, error) {
	cacheDir := os.Getenv("XDG_CACHE_HOME")
	if cacheDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		cacheDir = filepath.Join(home, ".cache")
	}
	cacheDir = filepath.Join(cacheDir, "gosynctasks")
	return cacheDir, os.MkdirAll(cacheDir, 0755)
}

// GetCacheFile returns the full path to the task lists cache file
func GetCacheFile() (string, error) {
	cacheDir, err := GetCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(cacheDir, "lists.json"), nil
}

// LoadTaskListsFromCache loads task lists from the cache file
func LoadTaskListsFromCache() ([]backend.TaskList, error) {
	cacheFile, err := GetCacheFile()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(cacheFile)
	if err != nil {
		return nil, err
	}

	var cached CachedData
	if err := json.Unmarshal(data, &cached); err != nil {
		return nil, err
	}

	return cached.Lists, nil
}

// SaveTaskListsToCache saves task lists to the cache file with timestamp
func SaveTaskListsToCache(lists []backend.TaskList) error {
	cacheFile, err := GetCacheFile()
	if err != nil {
		return err
	}

	cached := CachedData{
		Lists:     lists,
		Timestamp: time.Now().Unix(),
	}

	data, err := json.MarshalIndent(cached, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(cacheFile, data, 0644)
}

// LoadTaskListsWithFallback attempts to load from cache, falls back to fetching from remote
func LoadTaskListsWithFallback(taskManager backend.TaskManager) ([]backend.TaskList, error) {
	// Try cache first
	lists, err := LoadTaskListsFromCache()
	if err == nil {
		return lists, nil
	}

	// Fetch from remote
	lists, err = taskManager.GetTaskLists()
	if err != nil {
		return nil, err
	}

	// Save to cache for next time
	_ = SaveTaskListsToCache(lists)
	return lists, nil
}

// RefreshAndCacheTaskLists force-fetches task lists from remote and updates cache
func RefreshAndCacheTaskLists(taskManager backend.TaskManager) ([]backend.TaskList, error) {
	lists, err := taskManager.GetTaskLists()
	if err != nil {
		return nil, err
	}
	_ = SaveTaskListsToCache(lists)
	return lists, nil
}
