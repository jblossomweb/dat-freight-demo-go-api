package loads

import (
	"context"
	"errors"
	"log"
	"net/http"
	"time"
)

// GetResponse is the GET /load payload.
type GetResponse struct {
	RequestURL string  `json:"requestURL"`
	Meta       GetMeta `json:"meta"`
	Result     Load    `json:"result"`
}

// GetMeta holds response timing metadata for GET /load.
type GetMeta struct {
	ResponseTimeMs int64     `json:"responseTimeMs"`
	Timestamp      time.Time `json:"timestamp"`
}

// GetHandler returns the GET /load/{id} and GET /load?id=... handler backed by
// the given service. Responds 404 with a safe error message if not found.
//
// @Summary      Fetch a single load by id
// @Tags         loads
// @Produce      json
// @Param        id path     string false "Load id (path form)"
// @Param        id query    string false "Load id (query form); required if path id is omitted"
// @Success      200 {object} loads.GetResponse
// @Failure      400 {object} loads.ErrorResponse "no id provided"
// @Failure      404 {object} loads.ErrorResponse "no load found"
// @Router       /load/{id} [get]
// @Router       /load [get]
func GetHandler(service LoadService) http.HandlerFunc {
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

		writeJSON(w, http.StatusOK, GetResponse{
			RequestURL: requestedURL(r),
			Meta: GetMeta{
				ResponseTimeMs: time.Since(start).Milliseconds(),
				Timestamp:      time.Now().UTC(),
			},
			Result: load,
		})
	}
}
