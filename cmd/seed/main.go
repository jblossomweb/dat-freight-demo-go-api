// Command seed imports loads.json into the loads collection.
package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"time"

	"dat-freight-demo-go-api/internal/db"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const batchSize = 1000

type seedFile struct {
	Loads []bson.M `json:"loads"`
}

func main() {
	cfg, err := loadConfig()
	if err != nil {
		log.Fatalf("invalid configuration: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	client, err := db.Connect(ctx, cfg.mongoURI)
	if err != nil {
		log.Fatalf("failed to connect to mongo: %v", err)
	}
	defer client.Disconnect(context.Background())

	loads, err := readLoads(cfg.seedPath)
	if err != nil {
		log.Fatalf("failed to read seed file: %v", err)
	}
	log.Printf("read %d loads from %s", len(loads), cfg.seedPath)

	collection := client.Database(cfg.dbName).Collection("loads")

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

	log.Printf("done: %d loads seeded into %s.loads", inserted, cfg.dbName)
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
