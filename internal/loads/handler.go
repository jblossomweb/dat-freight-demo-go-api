package loads

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type queryResponse struct {
	RequestURL string    `json:"requestURL"` // the incoming path+query exactly as received, for debugging URL-encoding issues
	Query      queryEcho `json:"query"`
	Meta       queryMeta `json:"meta"`
	Rows       []Load    `json:"rows"`
	LastRow    int64     `json:"lastRow"`
}

// queryMeta holds derived/redundant values kept for human readability alongside the AG Grid contract fields.
type queryMeta struct {
	NumResults     int64     `json:"numResults"`     // rows matching the current filters/quicksearch; same value as lastRow
	NumTotal       int64     `json:"numTotal"`       // full collection size, ignoring filters/quicksearch
	PageSize       int       `json:"pageSize"`       // derived from endRow-startRow
	NumPages       int64     `json:"numPages"`       // ceil(filteredRows/pageSize)
	CurrentPage    int64     `json:"currentPage"`    // 1-indexed, derived from startRow/pageSize
	HasNextPage    bool      `json:"hasNextPage"`    // whether more filtered rows exist beyond this page
	NextPageURL    *string   `json:"nextPageURL"`    // path+query for the next page; null when there is no next page
	ResponseTimeMs int64     `json:"responseTimeMs"` // server-side processing time for this request
	Timestamp      time.Time `json:"timestamp"`      // server time the response was generated
}

// queryEcho reflects the effective request params actually applied, including defaults.
type queryEcho struct {
	QuickSearch string                      `json:"quickSearch"`
	SortModel   []SortModelEntry            `json:"sortModel"`
	FilterModel map[string]FilterModelEntry `json:"filterModel"`
	StartRow    int                         `json:"startRow"`
	EndRow      int                         `json:"endRow"`
}

type errorResponse struct {
	Error errorBody `json:"error"`
}

type errorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Handler returns the GET /loads handler backed by the given collection.
func Handler(collection *mongo.Collection) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		req, err := ParseQuery(r.URL.Query())
		if err != nil {
			writeError(w, http.StatusBadRequest, "LOAD_QUERY_FAILED", err.Error())
			return
		}

		filter, err := BuildMongoFilter(req)
		if err != nil {
			writeError(w, http.StatusBadRequest, "LOAD_QUERY_FAILED", err.Error())
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		totalRows, err := collection.EstimatedDocumentCount(ctx)
		if err != nil {
			log.Printf("estimated count failed: %v", err)
			writeError(w, http.StatusInternalServerError, "LOAD_QUERY_FAILED", "Unable to load freight data.")
			return
		}

		filteredRows, err := collection.CountDocuments(ctx, filter)
		if err != nil {
			log.Printf("count loads failed: %v", err)
			writeError(w, http.StatusInternalServerError, "LOAD_QUERY_FAILED", "Unable to load freight data.")
			return
		}

		findOpts := options.Find().
			SetSkip(int64(req.StartRow)).
			SetLimit(int64(req.EndRow - req.StartRow))
		if req.Sort != nil {
			direction := 1
			if req.Sort.Sort == "desc" {
				direction = -1
			}
			findOpts.SetSort(bson.D{{Key: req.Sort.ColID, Value: direction}})
		}

		cursor, err := collection.Find(ctx, filter, findOpts)
		if err != nil {
			log.Printf("find loads failed: %v", err)
			writeError(w, http.StatusInternalServerError, "LOAD_QUERY_FAILED", "Unable to load freight data.")
			return
		}
		defer cursor.Close(ctx)

		rows := []Load{}
		if err := cursor.All(ctx, &rows); err != nil {
			log.Printf("decode loads failed: %v", err)
			writeError(w, http.StatusInternalServerError, "LOAD_QUERY_FAILED", "Unable to load freight data.")
			return
		}

		pageSize := req.EndRow - req.StartRow
		hasNext := int64(req.EndRow) < filteredRows
		writeJSON(w, http.StatusOK, queryResponse{
			RequestURL: requestedURL(r),
			Query:      echoQuery(req),
			Meta: queryMeta{
				NumResults:     filteredRows,
				NumTotal:       totalRows,
				PageSize:       pageSize,
				NumPages:       numPages(filteredRows, pageSize),
				CurrentPage:    currentPage(req.StartRow, pageSize),
				HasNextPage:    hasNext,
				NextPageURL:    nextPageURL(r, req, hasNext),
				ResponseTimeMs: time.Since(start).Milliseconds(),
				Timestamp:      time.Now().UTC(),
			},
			Rows:    rows,
			LastRow: filteredRows,
		})
	}
}

func numPages(rows int64, pageSize int) int64 {
	if pageSize <= 0 {
		return 0
	}
	return (rows + int64(pageSize) - 1) / int64(pageSize)
}

func currentPage(startRow, pageSize int) int64 {
	if pageSize <= 0 {
		return 1
	}
	return int64(startRow/pageSize) + 1
}

// requestedURL returns the incoming path+query exactly as received (raw, undecoded),
// so URL-encoding mistakes are visible verbatim rather than normalized away.
func requestedURL(r *http.Request) string {
	if r.URL.RawQuery == "" {
		return r.URL.Path
	}
	return r.URL.Path + "?" + r.URL.RawQuery
}

// nextPageURL builds the path+query for the next page, advancing startRow/endRow.
// Params are ordered to match the 'query' payload: quickSearch, the enum alias
// filters (status, equipmentType), sortModel, filterModel, then any other
// params, with startRow/endRow last. Returns nil when there is no next page.
func nextPageURL(r *http.Request, req QueryRequest, hasNext bool) *string {
	if !hasNext {
		return nil
	}
	pageSize := req.EndRow - req.StartRow

	q := r.URL.Query()
	handled := map[string]bool{"startRow": true, "endRow": true}

	var parts []string

	for _, key := range []string{"quickSearch", "status", "equipmentType", "sortModel", "filterModel"} {
		values, ok := q[key]
		if !ok {
			continue
		}
		handled[key] = true
		for _, v := range values {
			parts = append(parts, key+"="+url.QueryEscape(v))
		}
	}

	var remainingKeys []string
	for key := range q {
		if !handled[key] {
			remainingKeys = append(remainingKeys, key)
		}
	}
	sort.Strings(remainingKeys)
	for _, key := range remainingKeys {
		for _, v := range q[key] {
			parts = append(parts, key+"="+url.QueryEscape(v))
		}
	}

	parts = append(parts,
		"startRow="+strconv.Itoa(req.EndRow),
		"endRow="+strconv.Itoa(req.EndRow+pageSize),
	)

	full := r.URL.Path + "?" + strings.Join(parts, "&")
	return &full
}

func echoQuery(req QueryRequest) queryEcho {
	sortModel := []SortModelEntry{}
	if req.Sort != nil {
		sortModel = append(sortModel, *req.Sort)
	}
	filterModel := req.Filters
	if filterModel == nil {
		filterModel = map[string]FilterModelEntry{}
	}
	return queryEcho{
		StartRow:    req.StartRow,
		EndRow:      req.EndRow,
		QuickSearch: req.QuickSearch,
		SortModel:   sortModel,
		FilterModel: filterModel,
	}
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, errorResponse{Error: errorBody{Code: code, Message: message}})
}
