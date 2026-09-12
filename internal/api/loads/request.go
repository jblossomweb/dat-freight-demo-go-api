package loads

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Pagination and timeout tuning: change these values to adjust default/max page
// size and how long a single request may take to query MongoDB.
const (
	defaultPageSize = 25              // used when the request omits both startRow and endRow
	maxPageSize     = 500             // hard cap on rows per page, enforced regardless of requested endRow
	queryTimeout    = 10 * time.Second // max time allowed for a single request's Mongo queries
)

// aliasSetFields lists columns that accept a simpler query param (e.g.
// status=Available&status=In+Transit) as shorthand for a set filter. A single
// occurrence behaves as an exact match; repeated occurrences behave as "one of".
var aliasSetFields = []string{
	"id", "companyName", "origin", "destination", "weight",
	"equipmentType", "date", "price", "distance", "status",
}

// SortModelEntry mirrors a single entry of AG Grid's sortModel.
type SortModelEntry struct {
	ColID string `json:"colId"`
	Sort  string `json:"sort"` // "asc" or "desc"
}

// FilterModelEntry mirrors AG Grid's simple filter model for one column.
type FilterModelEntry struct {
	FilterType string   `json:"filterType"`
	Type       string   `json:"type,omitempty"`
	Filter     any      `json:"filter,omitempty"`
	FilterTo   any      `json:"filterTo,omitempty"`
	DateFrom   string   `json:"dateFrom,omitempty"`
	DateTo     string   `json:"dateTo,omitempty"`
	Values     []string `json:"values,omitempty"`
}

// QueryRequest holds the parsed AG Grid server-side row model request parameters.
type QueryRequest struct {
	StartRow    int
	EndRow      int
	QuickSearch string
	Sort        *SortModelEntry // only the first sortModel entry is honored (single-column sort)
	Filters     map[string]FilterModelEntry
}

// ParseQuery parses AG Grid's startRow/endRow/quickSearch/sortModel/filterModel query params.
func ParseQuery(q url.Values) (QueryRequest, error) {
	req := QueryRequest{StartRow: 0, EndRow: defaultPageSize}

	// offset is a human-friendly alias for startRow; ignored if startRow is set.
	startRowParam := q.Get("startRow")
	if startRowParam == "" {
		startRowParam = q.Get("offset")
	}
	if startRowParam != "" {
		n, err := strconv.Atoi(startRowParam)
		if err != nil || n < 0 {
			return req, fmt.Errorf("invalid startRow: %q", startRowParam)
		}
		req.StartRow = n
	}

	// limit is a human-friendly alias for endRow (startRow+limit); ignored if endRow is set.
	if v := q.Get("endRow"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < req.StartRow {
			return req, fmt.Errorf("invalid endRow: %q (must not be less than startRow)", v)
		}
		req.EndRow = n
	} else if v := q.Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			return req, fmt.Errorf("invalid limit: %q", v)
		}
		req.EndRow = req.StartRow + n
	} else {
		req.EndRow = req.StartRow + defaultPageSize
	}

	if req.EndRow == req.StartRow {
		// startRow == endRow is a zero-width range; treat it leniently as "at
		// least one row starting here" rather than rejecting or returning
		// everything (Mongo's SetLimit(0) means "no limit", not zero rows).
		req.EndRow = req.StartRow + 1
	}

	if req.EndRow-req.StartRow > maxPageSize {
		req.EndRow = req.StartRow + maxPageSize
	}

	req.QuickSearch = q.Get("quickSearch")
	if req.QuickSearch == "" {
		// q is a shorthand alias for quickSearch; ignored if quickSearch is already set.
		req.QuickSearch = q.Get("q")
	}

	if v := q.Get("sortModel"); v != "" {
		var entries []SortModelEntry
		if err := json.Unmarshal([]byte(v), &entries); err != nil {
			return req, fmt.Errorf("invalid sortModel: %w", err)
		}
		if len(entries) > 0 {
			req.Sort = &entries[0]
		}
	}

	// sortBy/sortDir are a human-friendly alias for a single-column sortModel
	// entry; ignored if sortModel already specified a sort. "desc"/"descending"
	// (case-insensitive) sort descending; any other value (including empty or
	// invalid) defaults to ascending, matching sortModel's own lenient handling.
	if req.Sort == nil {
		if sortBy := q.Get("sortBy"); sortBy != "" {
			sortDir := "asc"
			switch strings.ToLower(q.Get("sortDir")) {
			case "desc", "descending":
				sortDir = "desc"
			}
			req.Sort = &SortModelEntry{ColID: sortBy, Sort: sortDir}
		}
	}

	if v := q.Get("filterModel"); v != "" {
		var filters map[string]FilterModelEntry
		if err := json.Unmarshal([]byte(v), &filters); err != nil {
			return req, fmt.Errorf("invalid filterModel: %w", err)
		}
		req.Filters = filters
	}

	// Alias params only apply when filterModel didn't already specify that field.
	for _, field := range aliasSetFields {
		values := q[field]
		if len(values) == 0 {
			continue
		}
		if req.Filters == nil {
			req.Filters = map[string]FilterModelEntry{}
		}
		if _, exists := req.Filters[field]; exists {
			continue
		}
		entry, err := aliasFilterEntry(field, values)
		if err != nil {
			return req, err
		}
		req.Filters[field] = entry
	}

	return req, nil
}

// aliasFilterEntry builds the effective filter for an alias param: a single
// value becomes an exact-match equals filter (type-aware), multiple values
// become a set (membership) filter.
func aliasFilterEntry(field string, values []string) (FilterModelEntry, error) {
	if len(values) > 1 {
		return FilterModelEntry{FilterType: "set", Values: values}, nil
	}

	value := values[0]
	switch {
	case numberFields[field]:
		n, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return FilterModelEntry{}, fmt.Errorf("invalid numeric value %q for field %q", value, field)
		}
		return FilterModelEntry{FilterType: "number", Type: "equals", Filter: n}, nil
	case field == "date":
		return FilterModelEntry{FilterType: "date", Type: "equals", DateFrom: value}, nil
	default:
		return FilterModelEntry{FilterType: "text", Type: "equals", Filter: value}, nil
	}
}
