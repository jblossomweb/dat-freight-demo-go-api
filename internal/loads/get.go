package loads

import (
	"context"
	"log"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
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
// the given collection. Responds 404 with a safe error message if not found.
func GetHandler(collection *mongo.Collection) http.HandlerFunc {
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

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		var load Load
		err := collection.FindOne(ctx, bson.M{"id": id}).Decode(&load)
		if err == mongo.ErrNoDocuments {
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
