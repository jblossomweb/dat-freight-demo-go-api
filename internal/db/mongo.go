// Package db provides MongoDB client construction and connectivity checks.
package db

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// Connect creates a MongoDB client and verifies connectivity with a ping.
func Connect(ctx context.Context, uri string) (*mongo.Client, error) {
	connectCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		return nil, fmt.Errorf("connect to mongo: %w", err)
	}

	if err := client.Ping(connectCtx, nil); err != nil {
		return nil, fmt.Errorf("ping mongo: %w", err)
	}

	return client, nil
}
