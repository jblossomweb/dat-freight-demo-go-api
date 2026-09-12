package loads

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func TestRepositoryIntegration(t *testing.T) {
	if os.Getenv("MONGO_INTEGRATION") != "1" {
		t.Skip("set MONGO_INTEGRATION=1 to run MongoDB integration tests")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	uri := os.Getenv("MONGO_TEST_URI")
	if uri == "" {
		t.Fatal("MONGO_TEST_URI must be set when MONGO_INTEGRATION=1")
	}

	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		t.Fatalf("connect to MongoDB: %v", err)
	}
	defer client.Disconnect(context.Background())

	if err := client.Ping(ctx, nil); err != nil {
		t.Fatalf("ping MongoDB: %v", err)
	}

	databaseName := fmt.Sprintf("freight_integration_test_%d", time.Now().UnixNano())
	collection := client.Database(databaseName).Collection("loads")
	defer client.Database(databaseName).Drop(context.Background())

	fixtures := []any{
		bson.M{"id": "LD-000001", "companyName": "Alpha Freight", "price": 3000.0, "status": "Available"},
		bson.M{"id": "LD-000002", "companyName": "Bravo Freight", "price": 1000.0, "status": "Booked"},
		bson.M{"id": "LD-000003", "companyName": "Charlie Freight", "price": 2000.0, "status": "Available"},
	}
	if _, err := collection.InsertMany(ctx, fixtures); err != nil {
		t.Fatalf("insert fixtures: %v", err)
	}

	repository := NewRepository(collection)

	t.Run("CountAll", func(t *testing.T) {
		count, err := repository.CountAll(ctx)
		if err != nil {
			t.Fatalf("CountAll() error = %v", err)
		}
		if count != int64(len(fixtures)) {
			t.Fatalf("CountAll() = %d, want %d", count, len(fixtures))
		}
	})

	t.Run("CountMatching", func(t *testing.T) {
		count, err := repository.CountMatching(ctx, bson.M{"status": "Available"})
		if err != nil {
			t.Fatalf("CountMatching() error = %v", err)
		}
		if count != 2 {
			t.Fatalf("CountMatching() = %d, want 2", count)
		}
	})

	t.Run("Find applies filter sort and pagination", func(t *testing.T) {
		rows, err := repository.Find(ctx, bson.M{"status": "Available"}, &SortModelEntry{ColID: "price", Sort: "desc"}, 0, 1)
		if err != nil {
			t.Fatalf("Find() error = %v", err)
		}
		if len(rows) != 1 || rows[0].ID != "LD-000001" {
			t.Fatalf("Find() = %#v, want highest-priced available load", rows)
		}

		rows, err = repository.Find(ctx, bson.M{}, &SortModelEntry{ColID: "price", Sort: "asc"}, 1, 1)
		if err != nil {
			t.Fatalf("Find() with skip error = %v", err)
		}
		if len(rows) != 1 || rows[0].ID != "LD-000003" {
			t.Fatalf("Find() with skip = %#v, want LD-000003", rows)
		}
	})

	t.Run("FindByID success", func(t *testing.T) {
		load, err := repository.FindByID(ctx, "LD-000002")
		if err != nil {
			t.Fatalf("FindByID() error = %v", err)
		}
		if load.CompanyName != "Bravo Freight" || load.Price != 1000 {
			t.Fatalf("FindByID() = %#v, want Bravo Freight at 1000", load)
		}
	})

	t.Run("FindByID not found", func(t *testing.T) {
		_, err := repository.FindByID(ctx, "LD-999999")
		if !errors.Is(err, ErrNotFound) {
			t.Fatalf("FindByID() error = %v, want ErrNotFound", err)
		}
	})
}
