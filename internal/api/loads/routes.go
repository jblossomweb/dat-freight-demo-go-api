package loads

import "net/http"

// RegisterRoutes mounts all /loads and /load routes on mux, backed by service.
// Callers (e.g. main.go) don't need to know the individual route paths.
func RegisterRoutes(mux *http.ServeMux, service LoadService) {
	mux.HandleFunc("GET /loads", ListHandler(service))
	mux.HandleFunc("GET /loads/stats", StatsHandler(service))
	mux.HandleFunc("GET /load/{id}", GetHandler(service))
	mux.HandleFunc("GET /load", GetHandler(service))
}
