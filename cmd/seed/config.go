package main

import (
	"dat-freight-demo-go-api/internal/env"
)

type config struct {
	mongoURI string
	dbName   string
	seedPath string
}

func loadConfig() (config, error) {
	mongoURI, err := env.Required("MONGO_URI")
	if err != nil {
		return config{}, err
	}

	return config{
		mongoURI: mongoURI,
		dbName:   env.Get("MONGO_DB_NAME", "freight"),
		seedPath: env.Get("SEED_FILE", "internal/db/seeds/loads.json"),
	}, nil
}
