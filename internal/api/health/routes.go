package health

import (
	"net/http"

	"go.mongodb.org/mongo-driver/v2/mongo"
)

// RegisterRoutes mounts the /health route on mux, backed by the given Mongo client.
func RegisterRoutes(mux *http.ServeMux, client *mongo.Client) {
	mux.HandleFunc("GET /health", Handler(client))
}
