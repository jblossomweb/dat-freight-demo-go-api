// Package env provides shared environment variable helpers.
package env

import (
	"fmt"
	"os"
)

// Get returns the environment value when it is set and non-empty;
// otherwise it returns fallback.
func Get(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok && value != "" {
		return value
	}
	return fallback
}

// Required returns a non-empty environment value or an error.
func Required(key string) (string, error) {
	value, ok := os.LookupEnv(key)
	if !ok || value == "" {
		return "", fmt.Errorf("required environment variable %s is not set", key)
	}
	return value, nil
}
