package main

import (
	"testing"
	"time"
)

func TestLoadConfig(t *testing.T) {
	t.Run("defaults", func(t *testing.T) {
		t.Setenv("MONGO_URI", "mongodb://example:27017")
		t.Setenv("PORT", "")
		t.Setenv("MONGO_DB_NAME", "")
		t.Setenv("LOAD_STATS_CACHE_TTL", "")

		got, err := loadConfig()
		if err != nil {
			t.Fatalf("loadConfig() error = %v", err)
		}
		if got.port != "8080" || got.mongoURI != "mongodb://example:27017" || got.dbName != "freight" || got.statsCacheTTL != 5*time.Minute {
			t.Fatalf("loadConfig() = %#v, want default port/db and configured URI", got)
		}
	})

	t.Run("environment overrides", func(t *testing.T) {
		t.Setenv("MONGO_URI", "mongodb://example:27017")
		t.Setenv("PORT", "9090")
		t.Setenv("MONGO_DB_NAME", "staging")
		t.Setenv("LOAD_STATS_CACHE_TTL", "30s")

		got, err := loadConfig()
		if err != nil {
			t.Fatalf("loadConfig() error = %v", err)
		}
		if got.port != "9090" || got.mongoURI != "mongodb://example:27017" || got.dbName != "staging" || got.statsCacheTTL != 30*time.Second {
			t.Fatalf("loadConfig() = %#v, want configured values", got)
		}
	})

	t.Run("zero disables stats cache", func(t *testing.T) {
		t.Setenv("MONGO_URI", "mongodb://example:27017")
		t.Setenv("LOAD_STATS_CACHE_TTL", "0")

		got, err := loadConfig()
		if err != nil {
			t.Fatalf("loadConfig() error = %v", err)
		}
		if got.statsCacheTTL != 0 {
			t.Fatalf("statsCacheTTL = %v, want 0", got.statsCacheTTL)
		}
	})

	for _, value := range []string{"invalid", "-1s"} {
		t.Run("invalid stats cache TTL "+value, func(t *testing.T) {
			t.Setenv("MONGO_URI", "mongodb://example:27017")
			t.Setenv("LOAD_STATS_CACHE_TTL", value)
			if _, err := loadConfig(); err == nil {
				t.Fatalf("loadConfig() error = nil, want error for %q", value)
			}
		})
	}
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
