// Command seed imports loads.json into the loads collection.
package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"time"

	"dat-freight-demo-api/internal/db"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const batchSize = 1000

type seedFile struct {
	Loads []bson.M `json:"loads"`
}

func main() {
	mongoURI := getEnv("MONGO_URI", "mongodb://localhost:27017")
	dbName := getEnv("MONGO_DB_NAME", "freight")
	seedPath := getEnv("SEED_FILE", "internal/db/seeds/loads.json")

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	client, err := db.Connect(ctx, mongoURI)
	if err != nil {
		log.Fatalf("failed to connect to mongo: %v", err)
	}
	defer client.Disconnect(context.Background())

	loads, err := readLoads(seedPath)
	if err != nil {
		log.Fatalf("failed to read seed file: %v", err)
	}
	log.Printf("read %d loads from %s", len(loads), seedPath)

	collection := client.Database(dbName).Collection("loads")

	if err := collection.Drop(ctx); err != nil {
		log.Fatalf("failed to clear existing loads collection: %v", err)
	}

	inserted := 0
	for start := 0; start < len(loads); start += batchSize {
		end := min(start+batchSize, len(loads))
		batch := make([]any, end-start)
		for i, l := range loads[start:end] {
			batch[i] = l
		}

		if _, err := collection.InsertMany(ctx, batch); err != nil {
			log.Fatalf("failed to insert batch starting at %d: %v", start, err)
		}
		inserted += len(batch)
		log.Printf("inserted %d/%d", inserted, len(loads))
	}

	if err := createIDIndex(ctx, collection); err != nil {
		log.Fatalf("failed to create index on id: %v", err)
	}

	log.Printf("done: %d loads seeded into %s.loads", inserted, dbName)
}

// createIDIndex ensures a unique index on the "id" field, since that's the
// primary lookup key for GET /load/{id} and the id alias/filter on GET /loads.
func createIDIndex(ctx context.Context, collection *mongo.Collection) error {
	_, err := collection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "id", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	return err
}

func readLoads(path string) ([]bson.M, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var sf seedFile
	if err := json.NewDecoder(f).Decode(&sf); err != nil {
		return nil, err
	}
	return sf.Loads, nil
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}
