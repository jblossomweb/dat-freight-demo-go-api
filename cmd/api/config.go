package main

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"dat-freight-demo-go-api/internal/env"
)

type config struct {
	port          string
	mongoURI      string
	dbName        string
	statsCacheTTL time.Duration
	allowedOrigins []string
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
	allowedOrigins, err := parseAllowedOrigins(env.Get("CORS_ALLOWED_ORIGINS", ""))
	if err != nil {
		return config{}, err
	}

	return config{
		port:          env.Get("PORT", "8080"),
		mongoURI:      mongoURI,
		dbName:        env.Get("MONGO_DB_NAME", "freight"),
		statsCacheTTL: statsCacheTTL,
		allowedOrigins: allowedOrigins,
	}, nil
}

func parseAllowedOrigins(value string) ([]string, error) {
	if strings.TrimSpace(value) == "" {
		return []string{}, nil
	}
	parts := strings.Split(value, ",")
	origins := make([]string, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))
	for _, part := range parts {
		origin := strings.TrimSpace(part)
		parsed, err := url.Parse(origin)
		if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" || (parsed.Path != "" && parsed.Path != "/") || parsed.RawQuery != "" || parsed.Fragment != "" || parsed.User != nil {
			return nil, fmt.Errorf("CORS_ALLOWED_ORIGINS contains invalid origin %q", origin)
		}
		origin = parsed.Scheme + "://" + parsed.Host
		if _, exists := seen[origin]; exists {
			continue
		}
		seen[origin] = struct{}{}
		origins = append(origins, origin)
	}
	return origins, nil
}
