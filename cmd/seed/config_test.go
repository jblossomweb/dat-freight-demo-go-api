package main

import "testing"

func TestLoadConfig(t *testing.T) {
	t.Run("configured values", func(t *testing.T) {
		t.Setenv("MONGO_URI", "mongodb://example:27017")
		t.Setenv("MONGO_DB_NAME", "staging")
		t.Setenv("SEED_FILE", "fixtures/loads.json")

		got, err := loadConfig()
		if err != nil {
			t.Fatalf("loadConfig() error = %v", err)
		}
		want := config{
			mongoURI: "mongodb://example:27017",
			dbName:   "staging",
			seedPath: "fixtures/loads.json",
		}
		if got != want {
			t.Fatalf("loadConfig() = %#v, want %#v", got, want)
		}
	})

	t.Run("defaults", func(t *testing.T) {
		t.Setenv("MONGO_URI", "mongodb://example:27017")
		t.Setenv("MONGO_DB_NAME", "")
		t.Setenv("SEED_FILE", "")

		got, err := loadConfig()
		if err != nil {
			t.Fatalf("loadConfig() error = %v", err)
		}
		if got.mongoURI != "mongodb://example:27017" || got.dbName != "freight" || got.seedPath != "internal/db/seeds/loads.json" {
			t.Fatalf("loadConfig() = %#v, want configured URI and default db/path", got)
		}
	})
}

func TestLoadConfigRequiresMongoURI(t *testing.T) {
	for _, name := range []string{"unset", "empty"} {
		t.Run(name, func(t *testing.T) {
			t.Setenv("MONGO_URI", "")
			if _, err := loadConfig(); err == nil {
				t.Fatal("loadConfig() error = nil, want missing MONGO_URI error")
			}
		})
	}
}
