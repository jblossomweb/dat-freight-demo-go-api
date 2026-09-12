// Package health implements the GET /health endpoint.
package health

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
)

// pingTimeout bounds how long the Mongo ping may take per request.
const pingTimeout = 3 * time.Second

// Handler returns the GET /health handler backed by the given Mongo client.
// It performs a live ping on every request rather than relying on cached
// startup state.
func Handler(client *mongo.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		pingCtx, cancel := context.WithTimeout(r.Context(), pingTimeout)
		defer cancel()

		status := "ok"
		code := http.StatusOK
		mongoStatus := "ok"

		if err := client.Ping(pingCtx, nil); err != nil {
			status = "degraded"
			code = http.StatusServiceUnavailable
			mongoStatus = "unreachable"
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(code)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"status": status,
			"mongo":  mongoStatus,
		})
	}
}
