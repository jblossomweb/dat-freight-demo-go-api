package loads

import (
	"context"
	"log"
	"net/http"
	"time"
)

// StatsResponse is the GET /loads/stats payload.
type StatsResponse struct {
	RequestURL string         `json:"requestURL"`
	Query      StatsQueryEcho `json:"query"`
	Meta       StatsMeta      `json:"meta"`
	Stats      StatsBody      `json:"stats"`
}

// StatsQueryEcho reflects the effective stats query parameters.
type StatsQueryEcho struct {
	QuickSearch string `json:"quickSearch"`
}

// StatsMeta contains full and filtered counts plus request timing metadata.
type StatsMeta struct {
	NumResults     int64     `json:"numResults"`
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
// @Summary      Get freight load statistics
// @Description  Returns fixed equipment type and status counts across all matching loads.
// @Tags         loads
// @Produce      json
// @Param        quickSearch query string false "Trimmed OR substring match across every field; double quotes group phrases; alias: q"
// @Param        q query string false "Trimmed alias for quickSearch; ignored when trimmed quickSearch is non-empty"
// @Success      200 {object} loads.StatsResponse
// @Failure      500 {object} loads.ErrorResponse
// @Router       /loads/stats [get]
func StatsHandler(service LoadService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		quickSearch := parseQuickSearch(r.URL.Query())

		ctx, cancel := context.WithTimeout(r.Context(), queryTimeout)
		defer cancel()

		stats, err := service.GetLoadStats(ctx, quickSearch)
		if err != nil {
			log.Printf("load stats failed: %v", err)
			writeError(w, http.StatusInternalServerError, "LOAD_STATS_FAILED", "Unable to load freight statistics.")
			return
		}

		writeJSON(w, http.StatusOK, StatsResponse{
			RequestURL: requestedURL(r),
			Query:      StatsQueryEcho{QuickSearch: quickSearch},
			Meta: StatsMeta{
				NumResults:     stats.NumResults,
				NumTotal:       stats.NumTotal,
				ResponseTimeMs: max(1, time.Since(start).Milliseconds()),
				Timestamp:      time.Now().UTC(),
			},
			Stats: StatsBody{Totals: stats},
		})
	}
}
