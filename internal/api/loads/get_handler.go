package loads

import (
	"context"
	"errors"
	"log"
	"net/http"
	"time"
)

type getResponse struct {
	RequestURL string  `json:"requestURL"`
	Meta       getMeta `json:"meta"`
	Result     Load    `json:"result"`
}

type getMeta struct {
	ResponseTimeMs int64     `json:"responseTimeMs"`
	Timestamp      time.Time `json:"timestamp"`
}

// GetHandler returns the GET /load/{id} and GET /load?id=... handler backed by
// the given service. Responds 404 with a safe error message if not found.
func GetHandler(service *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		id := r.PathValue("id")
		if id == "" {
			id = r.URL.Query().Get("id")
		}
		if id == "" {
			writeError(w, http.StatusBadRequest, "LOAD_ID_REQUIRED", "A load id is required.")
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), queryTimeout)
		defer cancel()

		load, err := service.GetLoad(ctx, id)
		if errors.Is(err, ErrNotFound) {
			writeError(w, http.StatusNotFound, "LOAD_NOT_FOUND", "No load was found with the given id.")
			return
		}
		if err != nil {
			log.Printf("find load %q failed: %v", id, err)
			writeError(w, http.StatusInternalServerError, "LOAD_QUERY_FAILED", "Unable to load freight data.")
			return
		}

		writeJSON(w, http.StatusOK, getResponse{
			RequestURL: requestedURL(r),
			Meta: getMeta{
				ResponseTimeMs: time.Since(start).Milliseconds(),
				Timestamp:      time.Now().UTC(),
			},
			Result: load,
		})
	}
}
