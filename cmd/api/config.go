package main

import (
	"fmt"
	"os"
)

type config struct {
	port     string
	mongoURI string
	dbName   string
}

func loadConfig() (config, error) {
	mongoURI, err := requiredEnv("MONGO_URI")
	if err != nil {
		return config{}, err
	}

	return config{
		port:     getEnv("PORT", "8080"),
		mongoURI: mongoURI,
		dbName:   getEnv("MONGO_DB_NAME", "freight"),
	}, nil
}

func requiredEnv(key string) (string, error) {
	value, ok := os.LookupEnv(key)
	if !ok || value == "" {
		return "", fmt.Errorf("required environment variable %s is not set", key)
	}
	return value, nil
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok && value != "" {
		return value
	}
	return fallback
}
