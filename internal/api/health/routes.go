package health

import (
	"net/http"
)

// RegisterRoutes mounts the /health route on mux, backed by the given pinger.
func RegisterRoutes(mux *http.ServeMux, client MongoPinger) {
	mux.HandleFunc("GET /health", Handler(client))
}
