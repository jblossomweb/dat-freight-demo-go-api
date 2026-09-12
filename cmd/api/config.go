package main

import "os"

type config struct {
	port     string
	mongoURI string
	dbName   string
}

func loadConfig() config {
	return config{
		port:     getEnv("PORT", "8080"),
		mongoURI: getEnv("MONGO_URI", "mongodb://localhost:27017"),
		dbName:   getEnv("MONGO_DB_NAME", "freight"),
	}
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok && value != "" {
		return value
	}
	return fallback
}
