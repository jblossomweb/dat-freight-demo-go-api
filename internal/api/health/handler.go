// Package health implements the GET /health endpoint.
package health

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

// pingTimeout bounds how long the Mongo ping may take per request.
const pingTimeout = 3 * time.Second

// MongoPinger is the small MongoDB boundary needed by the health handler.
type MongoPinger interface {
	Ping(context.Context) error
}

// Response is the GET /health payload.
type Response struct {
	Status string `json:"status" example:"ok"`
	Mongo  string `json:"mongo" example:"ok"`
}

// UnavailableResponse is the GET /health payload when MongoDB cannot be reached.
type UnavailableResponse struct {
	Status string `json:"status" example:"degraded"`
	Mongo  string `json:"mongo" example:"unreachable"`
}

// Handler returns the GET /health handler backed by the given Mongo client.
// It performs a live ping on every request rather than relying on cached
// startup state.
//
// @Summary      Liveness/readiness check
// @Description  Performs a live MongoDB ping on every request (not cached startup state).
// @Tags         health
// @Produce      json
// @Success      200 {object} health.Response
// @Failure      503 {object} health.UnavailableResponse
// @Router       /health [get]
func Handler(client MongoPinger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		pingCtx, cancel := context.WithTimeout(r.Context(), pingTimeout)
		defer cancel()

		resp := Response{Status: "ok", Mongo: "ok"}
		code := http.StatusOK

		if err := client.Ping(pingCtx); err != nil {
			resp.Status = "degraded"
			resp.Mongo = "unreachable"
			code = http.StatusServiceUnavailable
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(code)
		_ = json.NewEncoder(w).Encode(resp)
	}
}
