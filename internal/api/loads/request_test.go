package loads

import (
	"net/url"
	"reflect"
	"strings"
	"testing"
)

func TestParseQueryPagination(t *testing.T) {
	tests := []struct {
		name     string
		query    url.Values
		wantFrom int
		wantTo   int
	}{
		{
			name:     "defaults",
			query:    url.Values{},
			wantFrom: 0,
			wantTo:   defaultPageSize,
		},
		{
			name:     "explicit range",
			query:    url.Values{"startRow": {"10"}, "endRow": {"35"}},
			wantFrom: 10,
			wantTo:   35,
		},
		{
			name:     "offset alias",
			query:    url.Values{"offset": {"10"}},
			wantFrom: 10,
			wantTo:   35,
		},
		{
			name:     "startRow wins over offset",
			query:    url.Values{"startRow": {"10"}, "offset": {"20"}},
			wantFrom: 10,
			wantTo:   35,
		},
		{
			name:     "limit alias",
			query:    url.Values{"startRow": {"10"}, "limit": {"25"}},
			wantFrom: 10,
			wantTo:   35,
		},
		{
			name:     "endRow wins over limit",
			query:    url.Values{"startRow": {"10"}, "endRow": {"20"}, "limit": {"25"}},
			wantFrom: 10,
			wantTo:   20,
		},
		{
			name:     "equal range becomes one row",
			query:    url.Values{"startRow": {"10"}, "endRow": {"10"}},
			wantFrom: 10,
			wantTo:   11,
		},
		{
			name:     "page size is capped",
			query:    url.Values{"startRow": {"10"}, "endRow": {"1000"}},
			wantFrom: 10,
			wantTo:   10 + maxPageSize,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := ParseQuery(test.query)
			if err != nil {
				t.Fatalf("ParseQuery() error = %v", err)
			}
			if got.StartRow != test.wantFrom || got.EndRow != test.wantTo {
				t.Fatalf("ParseQuery() range = [%d, %d), want [%d, %d)", got.StartRow, got.EndRow, test.wantFrom, test.wantTo)
			}
		})
	}
}

func TestParseQueryQuickSearch(t *testing.T) {
	tests := []struct {
		name  string
		query url.Values
		want  string
	}{
		{name: "quickSearch", query: url.Values{"quickSearch": {"chicago"}}, want: "chicago"},
		{name: "q alias", query: url.Values{"q": {"chicago"}}, want: "chicago"},
		{name: "quickSearch wins over q", query: url.Values{"quickSearch": {"chicago"}, "q": {"denver"}}, want: "chicago"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := ParseQuery(test.query)
			if err != nil {
				t.Fatalf("ParseQuery() error = %v", err)
			}
			if got.QuickSearch != test.want {
				t.Fatalf("QuickSearch = %q, want %q", got.QuickSearch, test.want)
			}
		})
	}
}

func TestParseQuerySort(t *testing.T) {
	tests := []struct {
		name  string
		query url.Values
		want  *SortModelEntry
	}{
		{name: "sortModel", query: url.Values{"sortModel": {`[{"colId":"price","sort":"desc"},{"colId":"status","sort":"asc"}]`}}, want: &SortModelEntry{ColID: "price", Sort: "desc"}},
		{name: "sort alias ascending", query: url.Values{"sortBy": {"price"}}, want: &SortModelEntry{ColID: "price", Sort: "asc"}},
		{name: "sort alias descending", query: url.Values{"sortBy": {"price"}, "sortDir": {"DESCENDING"}}, want: &SortModelEntry{ColID: "price", Sort: "desc"}},
		{name: "invalid sort direction defaults ascending", query: url.Values{"sortBy": {"price"}, "sortDir": {"sideways"}}, want: &SortModelEntry{ColID: "price", Sort: "asc"}},
		{name: "sortModel wins over aliases", query: url.Values{"sortModel": {`[{"colId":"status","sort":"asc"}]`}, "sortBy": {"price"}, "sortDir": {"desc"}}, want: &SortModelEntry{ColID: "status", Sort: "asc"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := ParseQuery(test.query)
			if err != nil {
				t.Fatalf("ParseQuery() error = %v", err)
			}
			if !reflect.DeepEqual(got.Sort, test.want) {
				t.Fatalf("Sort = %#v, want %#v", got.Sort, test.want)
			}
		})
	}
}

func TestParseQueryFilters(t *testing.T) {
	query := url.Values{
		"filterModel": {`{"status":{"filterType":"text","type":"contains","filter":"Available"}}`},
		"status":      {"In Transit"},
		"companyName": {"Swift Transport"},
		"weight":      {"32000"},
		"price":       {"100", "200"},
		"date":        {"2024-07-05"},
	}

	got, err := ParseQuery(query)
	if err != nil {
		t.Fatalf("ParseQuery() error = %v", err)
	}

	want := map[string]FilterModelEntry{
		"status":      {FilterType: "text", Type: "contains", Filter: "Available"},
		"companyName": {FilterType: "text", Type: "equals", Filter: "Swift Transport"},
		"weight":      {FilterType: "number", Type: "equals", Filter: float64(32000)},
		"price":       {FilterType: "set", Values: []string{"100", "200"}},
		"date":        {FilterType: "date", Type: "equals", DateFrom: "2024-07-05"},
	}
	if !reflect.DeepEqual(got.Filters, want) {
		t.Fatalf("Filters = %#v, want %#v", got.Filters, want)
	}
}

func TestParseQueryInvalidInput(t *testing.T) {
	tests := []struct {
		name  string
		query url.Values
		want  string
	}{
		{name: "invalid startRow", query: url.Values{"startRow": {"nope"}}, want: "invalid startRow"},
		{name: "negative startRow", query: url.Values{"startRow": {"-1"}}, want: "invalid startRow"},
		{name: "invalid endRow", query: url.Values{"endRow": {"nope"}}, want: "invalid endRow"},
		{name: "endRow before startRow", query: url.Values{"startRow": {"10"}, "endRow": {"9"}}, want: "invalid endRow"},
		{name: "invalid limit", query: url.Values{"limit": {"nope"}}, want: "invalid limit"},
		{name: "negative limit", query: url.Values{"limit": {"-1"}}, want: "invalid limit"},
		{name: "invalid sortModel", query: url.Values{"sortModel": {"not-json"}}, want: "invalid sortModel"},
		{name: "invalid filterModel", query: url.Values{"filterModel": {"not-json"}}, want: "invalid filterModel"},
		{name: "invalid numeric alias", query: url.Values{"price": {"not-a-number"}}, want: "invalid numeric value"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := ParseQuery(test.query)
			if err == nil {
				t.Fatal("ParseQuery() error = nil, want an error")
			}
			if !strings.Contains(err.Error(), test.want) {
				t.Fatalf("ParseQuery() error = %q, want substring %q", err, test.want)
			}
		})
	}
}
