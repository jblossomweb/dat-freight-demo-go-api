package loads

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

// ListResponse is the GET /loads payload.
type ListResponse struct {
	RequestURL string    `json:"requestURL"` // the incoming path+query exactly as received, for debugging URL-encoding issues
	Query      QueryEcho `json:"query"`
	Meta       ListMeta  `json:"meta"`
	Rows       []Load    `json:"rows"`
	LastRow    int64     `json:"lastRow"`
}

// ListMeta holds derived/redundant values kept for human readability alongside the AG Grid contract fields.
type ListMeta struct {
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

// QueryEcho reflects the effective request params actually applied, including defaults.
type QueryEcho struct {
	QuickSearch string                      `json:"quickSearch"`
	SortModel   []SortModelEntry            `json:"sortModel"`
	FilterModel map[string]FilterModelEntry `json:"filterModel"`
	StartRow    int                         `json:"startRow"`
	EndRow      int                         `json:"endRow"`
}

// ErrorResponse is the standard error payload shape used across this package's endpoints.
type ErrorResponse struct {
	Error ErrorBody `json:"error"`
}

// ErrorBody carries a machine-readable code plus a human-readable message.
type ErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// ListHandler returns the GET /loads handler backed by the given service.
//
// @Summary      Query freight loads (AG Grid server-side row model contract)
// @Description  Supports pagination, single-column sort, multi-column filtering, and
// @Description  quicksearch, plus human-friendly aliases for each param.
// @Tags         loads
// @Produce      json
// @Param        startRow    query int    false "First row index (inclusive); alias: offset"
// @Param        endRow      query int    false "Row index one past the last requested (exclusive); alias: limit"
// @Param        quickSearch query string false "Substring match across every field; alias: q"
// @Param        sortModel   query string false "JSON-encoded [{colId,sort}]; alias: sortBy/sortDir"
// @Param        filterModel query string false "JSON-encoded AG Grid simple filter model (text/number/date/set)"
// @Success      200 {object} loads.ListResponse
// @Failure      400 {object} loads.ErrorResponse
// @Router       /loads [get]
func ListHandler(service LoadService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		req, err := ParseQuery(r.URL.Query())
		if err != nil {
			writeError(w, http.StatusBadRequest, "LOAD_QUERY_FAILED", err.Error())
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), queryTimeout)
		defer cancel()

		rows, totalRows, filteredRows, err := service.ListLoads(ctx, req)
		if err != nil {
			var invalidQuery *InvalidQueryError
			if errors.As(err, &invalidQuery) {
				writeError(w, http.StatusBadRequest, "LOAD_QUERY_FAILED", err.Error())
				return
			}
			log.Printf("list loads failed: %v", err)
			writeError(w, http.StatusInternalServerError, "LOAD_QUERY_FAILED", "Unable to load freight data.")
			return
		}

		pageSize := req.EndRow - req.StartRow
		hasNext := int64(req.EndRow) < filteredRows
		writeJSON(w, http.StatusOK, ListResponse{
			RequestURL: requestedURL(r),
			Query:      echoQuery(req),
			Meta: ListMeta{
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

func echoQuery(req QueryRequest) QueryEcho {
	sortModel := []SortModelEntry{}
	if req.Sort != nil {
		sortModel = append(sortModel, *req.Sort)
	}
	filterModel := req.Filters
	if filterModel == nil {
		filterModel = map[string]FilterModelEntry{}
	}
	return QueryEcho{
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
	writeJSON(w, status, ErrorResponse{Error: ErrorBody{Code: code, Message: message}})
}
