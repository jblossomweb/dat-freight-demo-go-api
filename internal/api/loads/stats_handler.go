package loads

import (
	"context"
	"log"
	"net/http"
	"time"
)

// StatsResponse is the GET /loads/stats payload.
type StatsResponse struct {
	RequestURL string    `json:"requestURL"`
	Meta       StatsMeta `json:"meta"`
	Stats      StatsBody `json:"stats"`
}

// StatsMeta contains full-dataset and request timing metadata.
type StatsMeta struct {
	NumTotal       int64     `json:"numTotal"`
	ResponseTimeMs int64     `json:"responseTimeMs"`
	Timestamp      time.Time `json:"timestamp"`
}

// StatsBody contains aggregate load statistics.
type StatsBody struct {
	Totals LoadStats `json:"totals"`
}

// StatsHandler returns the GET /loads/stats handler backed by the given service.
//
// @Summary      Get full-dataset freight load statistics
// @Description  Returns fixed equipment type and status counts across all loads.
// @Tags         loads
// @Produce      json
// @Success      200 {object} loads.StatsResponse
// @Failure      500 {object} loads.ErrorResponse
// @Router       /loads/stats [get]
func StatsHandler(service LoadService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		ctx, cancel := context.WithTimeout(r.Context(), queryTimeout)
		defer cancel()

		stats, err := service.GetLoadStats(ctx)
		if err != nil {
			log.Printf("load stats failed: %v", err)
			writeError(w, http.StatusInternalServerError, "LOAD_STATS_FAILED", "Unable to load freight statistics.")
			return
		}

		writeJSON(w, http.StatusOK, StatsResponse{
			RequestURL: requestedURL(r),
			Meta: StatsMeta{
				NumTotal:       stats.NumTotal,
				ResponseTimeMs: max(1, time.Since(start).Milliseconds()),
				Timestamp:      time.Now().UTC(),
			},
			Stats: StatsBody{Totals: stats},
		})
	}
}
