package main

import "testing"

func TestLoadConfig(t *testing.T) {
	t.Run("defaults", func(t *testing.T) {
		t.Setenv("MONGO_URI", "mongodb://example:27017")
		t.Setenv("PORT", "")
		t.Setenv("MONGO_DB_NAME", "")

		got, err := loadConfig()
		if err != nil {
			t.Fatalf("loadConfig() error = %v", err)
		}
		if got.port != "8080" || got.mongoURI != "mongodb://example:27017" || got.dbName != "freight" {
			t.Fatalf("loadConfig() = %#v, want default port/db and configured URI", got)
		}
	})

	t.Run("environment overrides", func(t *testing.T) {
		t.Setenv("MONGO_URI", "mongodb://example:27017")
		t.Setenv("PORT", "9090")
		t.Setenv("MONGO_DB_NAME", "staging")

		got, err := loadConfig()
		if err != nil {
			t.Fatalf("loadConfig() error = %v", err)
		}
		if got.port != "9090" || got.mongoURI != "mongodb://example:27017" || got.dbName != "staging" {
			t.Fatalf("loadConfig() = %#v, want configured values", got)
		}
	})
}

func TestLoadConfigRequiresMongoURI(t *testing.T) {
	t.Run("unset", func(t *testing.T) {
		t.Setenv("MONGO_URI", "")
		if _, err := loadConfig(); err == nil {
			t.Fatal("loadConfig() error = nil, want missing MONGO_URI error")
		}
	})

	t.Run("empty", func(t *testing.T) {
		t.Setenv("MONGO_URI", "")
		if _, err := loadConfig(); err == nil {
			t.Fatal("loadConfig() error = nil, want empty MONGO_URI error")
		}
	})
}
