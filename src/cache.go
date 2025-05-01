package main

import (
	"encoding/gob"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
)

const (
	cacheDirName  = "gostawin"
	cacheFileName = "stats.gob"
)

// GetCachePath returns the full path to the cache file.
func GetCachePath() (string, error) {
	usr, err := user.Current()
	if err != nil {
		return "", fmt.Errorf("failed to get current user: %w", err)
	}
	cacheDir := filepath.Join(usr.HomeDir, ".cache", cacheDirName)
	return filepath.Join(cacheDir, cacheFileName), nil
}

// EnsureCacheDir creates the cache directory if it doesn't exist.
func EnsureCacheDir() error {
	cachePath, err := GetCachePath()
	if err != nil {
		return err // Error already formatted by GetCachePath
	}
	cacheDir := filepath.Dir(cachePath)
	// os.MkdirAll is idempotent, no error if directory already exists
	if err := os.MkdirAll(cacheDir, 0o750); err != nil { // Use 0750 for slightly better permissions
		return fmt.Errorf("failed to create cache directory '%s': %w", cacheDir, err)
	}
	return nil
}

// SavePrograms encodes and saves the program map to the cache file.
func SavePrograms(programs map[string]Program) error {
	if err := EnsureCacheDir(); err != nil {
		// Error already formatted by EnsureCacheDir
		return fmt.Errorf("failed prerequisite for saving programs: %w", err)
	}

	cachePath, err := GetCachePath()
	if err != nil {
		return err // Error already formatted by GetCachePath
	}

	// Create or truncate the file
	file, err := os.Create(cachePath)
	if err != nil {
		return fmt.Errorf("failed to create/open cache file '%s': %w", cachePath, err)
	}
	defer file.Close() // Ensure file is closed

	// Encode the data using gob
	encoder := gob.NewEncoder(file)
	if err := encoder.Encode(programs); err != nil {
		// Consider deleting the possibly corrupted file here?
		// os.Remove(cachePath)
		return fmt.Errorf("failed to encode programs to cache file '%s': %w", cachePath, err)
	}

	return nil // Success
}

// LoadPrograms decodes and loads the program map from the cache file.
// Returns an empty map if the file doesn't exist.
func LoadPrograms() (map[string]Program, error) {
	cachePath, err := GetCachePath()
	if err != nil {
		return nil, err // Error already formatted by GetCachePath
	}

	file, err := os.Open(cachePath)
	if err != nil {
		if os.IsNotExist(err) {
			// File not found is not an error, just return an empty map
			return make(map[string]Program), nil
		}
		// Other errors opening the file (e.g., permissions)
		return nil, fmt.Errorf("failed to open cache file '%s': %w", cachePath, err)
	}
	defer file.Close() // Ensure file is closed

	var programs map[string]Program
	decoder := gob.NewDecoder(file)
	if err := decoder.Decode(&programs); err != nil {
		// Handle potential decoding errors (e.g., corrupted file, format change)
		// Consider returning an empty map and logging a warning instead of a hard error
		// log.Printf("Warning: Failed to decode cache file '%s', starting fresh: %v", cachePath, err)
		// return make(map[string]Program), nil
		return nil, fmt.Errorf("failed to decode programs from cache file '%s': %w", cachePath, err)
	}

	if programs == nil { // Ensure we don't return a nil map if decoding results in nil
		programs = make(map[string]Program)
	}

	return programs, nil // Success
}
