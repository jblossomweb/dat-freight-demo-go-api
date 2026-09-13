package main

import (
	"fmt"
	"time"

	"dat-freight-demo-go-api/internal/env"
)

type config struct {
	port          string
	mongoURI      string
	dbName        string
	statsCacheTTL time.Duration
}

func loadConfig() (config, error) {
	mongoURI, err := env.Required("MONGO_URI")
	if err != nil {
		return config{}, err
	}
	statsCacheTTL, err := time.ParseDuration(env.Get("LOAD_STATS_CACHE_TTL", "5m"))
	if err != nil {
		return config{}, fmt.Errorf("parse LOAD_STATS_CACHE_TTL: %w", err)
	}
	if statsCacheTTL < 0 {
		return config{}, fmt.Errorf("LOAD_STATS_CACHE_TTL must not be negative")
	}

	return config{
		port:          env.Get("PORT", "8080"),
		mongoURI:      mongoURI,
		dbName:        env.Get("MONGO_DB_NAME", "freight"),
		statsCacheTTL: statsCacheTTL,
	}, nil
}
