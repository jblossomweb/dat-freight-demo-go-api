package main

import (
	"dat-freight-demo-go-api/internal/env"
)

type config struct {
	port     string
	mongoURI string
	dbName   string
}

func loadConfig() (config, error) {
	mongoURI, err := env.Required("MONGO_URI")
	if err != nil {
		return config{}, err
	}

	return config{
		port:     env.Get("PORT", "8080"),
		mongoURI: mongoURI,
		dbName:   env.Get("MONGO_DB_NAME", "freight"),
	}, nil
}
