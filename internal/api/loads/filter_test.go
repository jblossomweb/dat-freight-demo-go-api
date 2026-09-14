package loads

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestBuildMongoFilter(t *testing.T) {
	t.Run("empty request", func(t *testing.T) {
		got, err := BuildMongoFilter(QueryRequest{})
		if err != nil {
			t.Fatalf("BuildMongoFilter() error = %v", err)
		}
		if !reflect.DeepEqual(got, bson.M{}) {
			t.Fatalf("filter = %#v, want empty filter", got)
		}
	})

	t.Run("empty quoted quicksearch is ignored", func(t *testing.T) {
		got, err := BuildMongoFilter(QueryRequest{QuickSearch: `""`})
		if err != nil {
			t.Fatalf("BuildMongoFilter() error = %v", err)
		}
		if !reflect.DeepEqual(got, bson.M{}) {
			t.Fatalf("filter = %#v, want empty filter", got)
		}
	})

	t.Run("combines filters and quicksearch", func(t *testing.T) {
		got, err := BuildMongoFilter(QueryRequest{
			Filters: map[string]FilterModelEntry{
				"status": {FilterType: "text", Type: "equals", Filter: "Available"},
			},
			QuickSearch: "chicago",
		})
		if err != nil {
			t.Fatalf("BuildMongoFilter() error = %v", err)
		}

		and, ok := got["$and"].([]bson.M)
		if !ok {
			t.Fatalf("$and = %#v, want []bson.M", got["$and"])
		}
		if len(and) != 2 {
			t.Fatalf("len($and) = %d, want 2", len(and))
		}
	})

	t.Run("returns filter error", func(t *testing.T) {
		_, err := BuildMongoFilter(QueryRequest{
			Filters: map[string]FilterModelEntry{
				"status": {FilterType: "text", Type: "unsupported", Filter: "Available"},
			},
		})
		if err == nil || !strings.Contains(err.Error(), "unsupported text filter type") {
			t.Fatalf("error = %v, want unsupported text filter error", err)
		}
	})
}

func TestBuildTextFilter(t *testing.T) {
	tests := []struct {
		name       string
		filterType string
		want       bson.M
	}{
		{name: "equals", filterType: "equals", want: bson.M{"status": bson.Regex{Pattern: "^Available$", Options: "i"}}},
		{name: "not equal", filterType: "notEqual", want: bson.M{"status": bson.M{"$not": bson.Regex{Pattern: "^Available$", Options: "i"}}}},
		{name: "contains", filterType: "contains", want: bson.M{"status": bson.Regex{Pattern: "Available", Options: "i"}}},
		{name: "empty type defaults to contains", filterType: "", want: bson.M{"status": bson.Regex{Pattern: "Available", Options: "i"}}},
		{name: "not contains", filterType: "notContains", want: bson.M{"status": bson.M{"$not": bson.Regex{Pattern: "Available", Options: "i"}}}},
		{name: "starts with", filterType: "startsWith", want: bson.M{"status": bson.Regex{Pattern: "^Available", Options: "i"}}},
		{name: "ends with", filterType: "endsWith", want: bson.M{"status": bson.Regex{Pattern: "Available$", Options: "i"}}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := buildTextFilter("status", FilterModelEntry{Type: test.filterType, Filter: "Available"})
			if err != nil {
				t.Fatalf("buildTextFilter() error = %v", err)
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Fatalf("filter = %#v, want %#v", got, test.want)
			}
		})
	}

	t.Run("escapes regex characters", func(t *testing.T) {
		got, err := buildTextFilter("status", FilterModelEntry{Type: "contains", Filter: "A+B"})
		if err != nil {
			t.Fatalf("buildTextFilter() error = %v", err)
		}
		want := bson.M{"status": bson.Regex{Pattern: "A\\+B", Options: "i"}}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("filter = %#v, want %#v", got, want)
		}
	})

	_, err := buildTextFilter("status", FilterModelEntry{Type: "unsupported", Filter: "Available"})
	if err == nil || !strings.Contains(err.Error(), "unsupported text filter type") {
		t.Fatalf("error = %v, want unsupported text filter error", err)
	}
}

func TestBuildNumberFilter(t *testing.T) {
	tests := []struct {
		name       string
		filterType string
		want       bson.M
	}{
		{name: "equals", filterType: "equals", want: bson.M{"price": float64(100)}},
		{name: "not equal", filterType: "notEqual", want: bson.M{"price": bson.M{"$ne": float64(100)}}},
		{name: "less than", filterType: "lessThan", want: bson.M{"price": bson.M{"$lt": float64(100)}}},
		{name: "less than or equal", filterType: "lessThanOrEqual", want: bson.M{"price": bson.M{"$lte": float64(100)}}},
		{name: "greater than", filterType: "greaterThan", want: bson.M{"price": bson.M{"$gt": float64(100)}}},
		{name: "greater than or equal", filterType: "greaterThanOrEqual", want: bson.M{"price": bson.M{"$gte": float64(100)}}},
		{name: "in range", filterType: "inRange", want: bson.M{"price": bson.M{"$gte": float64(100), "$lte": float64(200)}}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			entry := FilterModelEntry{Type: test.filterType, Filter: float64(100)}
			if test.filterType == "inRange" {
				entry.FilterTo = float64(200)
			}
			got, err := buildNumberFilter("price", entry)
			if err != nil {
				t.Fatalf("buildNumberFilter() error = %v", err)
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Fatalf("filter = %#v, want %#v", got, test.want)
			}
		})
	}

	testsWithErrors := []struct {
		name  string
		entry FilterModelEntry
		want  string
	}{
		{name: "invalid value", entry: FilterModelEntry{Type: "equals", Filter: "100"}, want: "invalid numeric filter value"},
		{name: "invalid range end", entry: FilterModelEntry{Type: "inRange", Filter: float64(100), FilterTo: "200"}, want: "invalid filterTo"},
		{name: "unsupported type", entry: FilterModelEntry{Type: "unsupported", Filter: float64(100)}, want: "unsupported number filter type"},
	}
	for _, test := range testsWithErrors {
		t.Run(test.name, func(t *testing.T) {
			_, err := buildNumberFilter("price", test.entry)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want substring %q", err, test.want)
			}
		})
	}
}

func TestBuildDateFilter(t *testing.T) {
	tests := []struct {
		name       string
		filterType string
		want       bson.M
	}{
		{name: "equals", filterType: "equals", want: bson.M{"date": "2024-01-01"}},
		{name: "not equal", filterType: "notEqual", want: bson.M{"date": bson.M{"$ne": "2024-01-01"}}},
		{name: "less than", filterType: "lessThan", want: bson.M{"date": bson.M{"$lt": "2024-01-01"}}},
		{name: "greater than", filterType: "greaterThan", want: bson.M{"date": bson.M{"$gt": "2024-01-01"}}},
		{name: "in range", filterType: "inRange", want: bson.M{"date": bson.M{"$gte": "2024-01-01", "$lte": "2024-12-31"}}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := buildDateFilter("date", FilterModelEntry{Type: test.filterType, DateFrom: "2024-01-01", DateTo: "2024-12-31"})
			if err != nil {
				t.Fatalf("buildDateFilter() error = %v", err)
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Fatalf("filter = %#v, want %#v", got, test.want)
			}
		})
	}

	_, err := buildDateFilter("date", FilterModelEntry{Type: "unsupported", DateFrom: "2024-01-01"})
	if err == nil || !strings.Contains(err.Error(), "unsupported date filter type") {
		t.Fatalf("error = %v, want unsupported date filter error", err)
	}
}

func TestBuildSetFilter(t *testing.T) {
	tests := []struct {
		name   string
		field  string
		values []string
		want   bson.M
	}{
		{name: "strings", field: "status", values: []string{"Available", "In Transit"}, want: bson.M{"status": bson.M{"$in": []string{"Available", "In Transit"}}}},
		{name: "numbers are coerced", field: "price", values: []string{"100", "200.5"}, want: bson.M{"price": bson.M{"$in": []float64{100, 200.5}}}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := buildSetFilter(test.field, test.values)
			if err != nil {
				t.Fatalf("buildSetFilter() error = %v", err)
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Fatalf("filter = %#v, want %#v", got, test.want)
			}
		})
	}

	_, err := buildSetFilter("price", []string{"not-a-number"})
	if err == nil || !strings.Contains(err.Error(), "invalid numeric value") {
		t.Fatalf("error = %v, want invalid numeric value error", err)
	}
}

func TestBuildQuickSearch(t *testing.T) {
	got := buildQuickSearch(`  "A+B C.D"  E?F  `)
	or, ok := got["$or"].([]bson.M)
	if !ok {
		t.Fatalf("$or = %#v, want []bson.M", got["$or"])
	}
	wantAlternatives := 2 * (len(stringFields) + len(numberFields))
	if len(or) != wantAlternatives {
		t.Fatalf("len($or) = %d, want %d", len(or), wantAlternatives)
	}

	for _, pattern := range []string{"A\\+B C\\.D", "E\\?F"} {
		for _, field := range stringFields {
			if !containsBSONRegex(or, field, bson.Regex{Pattern: pattern, Options: "i"}) {
				t.Errorf("quicksearch missing string regex %q for %q", pattern, field)
			}
		}
		for field := range numberFields {
			if !containsNumericSearch(or, field, pattern) {
				t.Errorf("quicksearch missing numeric expression %q for %q", pattern, field)
			}
		}
	}
}

func TestTokenizeQuickSearch(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  []string
	}{
		{name: "unquoted terms", value: "miami denver", want: []string{"miami", "denver"}},
		{name: "quoted phrase and term", value: `"new york" miami`, want: []string{"new york", "miami"}},
		{name: "escaped quote in phrase", value: `"J.B. \"Hunt\"" reefer`, want: []string{`J.B. "Hunt"`, "reefer"}},
		{name: "empty quotes ignored", value: `"" miami ""`, want: []string{"miami"}},
		{name: "unicode whitespace separates terms", value: "miami\u2003denver", want: []string{"miami", "denver"}},
		{name: "whitespace preserved in phrase", value: `"new  york"`, want: []string{"new  york"}},
		{name: "unfinished quote groups remainder", value: `miami "new york`, want: []string{"miami", "new york"}},
		{name: "literal backslash preserved", value: `"a\b"`, want: []string{`a\b`}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := tokenizeQuickSearch(test.value); !reflect.DeepEqual(got, test.want) {
				t.Fatalf("tokenizeQuickSearch(%q) = %#v, want %#v", test.value, got, test.want)
			}
		})
	}
}

func TestBuildFieldFilterDispatchesByType(t *testing.T) {
	tests := []struct {
		name  string
		field string
		entry FilterModelEntry
		want  bson.M
	}{
		{name: "set takes precedence", field: "price", entry: FilterModelEntry{FilterType: "set", Values: []string{"100"}}, want: bson.M{"price": bson.M{"$in": []float64{100}}}},
		{name: "number field", field: "price", entry: FilterModelEntry{FilterType: "number", Type: "equals", Filter: float64(100)}, want: bson.M{"price": float64(100)}},
		{name: "date field", field: "date", entry: FilterModelEntry{FilterType: "date", Type: "equals", DateFrom: "2024-01-01"}, want: bson.M{"date": "2024-01-01"}},
		{name: "text field", field: "status", entry: FilterModelEntry{FilterType: "text", Type: "equals", Filter: "Available"}, want: bson.M{"status": bson.Regex{Pattern: "^Available$", Options: "i"}}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := buildFieldFilter(test.field, test.entry)
			if err != nil {
				t.Fatalf("buildFieldFilter() error = %v", err)
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Fatalf("filter = %#v, want %#v", got, test.want)
			}
		})
	}
}

func containsBSONRegex(filters []bson.M, field string, want bson.Regex) bool {
	for _, filter := range filters {
		if got, ok := filter[field].(bson.Regex); ok && got == want {
			return true
		}
	}
	return false
}

func containsNumericSearch(filters []bson.M, field, wantRegex string) bool {
	for _, filter := range filters {
		expr, ok := filter["$expr"].(bson.M)
		if !ok {
			continue
		}
		regexMatch, ok := expr["$regexMatch"].(bson.M)
		if !ok {
			continue
		}
		input, inputOK := regexMatch["input"].(bson.M)
		regex, regexOK := regexMatch["regex"].(string)
		if inputOK && regexOK && regex == wantRegex && input["$toString"] == "$"+field {
			return true
		}
	}
	return false
}

func TestToFloat64(t *testing.T) {
	tests := []struct {
		name  string
		value any
		want  float64
		ok    bool
	}{
		{name: "float64", value: float64(1.5), want: 1.5, ok: true},
		{name: "int", value: 2, want: 2, ok: true},
		{name: "json number", value: json.Number("3.5"), want: 3.5, ok: true},
		{name: "invalid json number", value: json.Number("bad"), ok: false},
		{name: "unsupported type", value: "4", ok: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, ok := toFloat64(test.value)
			if got != test.want || ok != test.ok {
				t.Fatalf("toFloat64(%#v) = (%v, %v), want (%v, %v)", test.value, got, ok, test.want, test.ok)
			}
		})
	}
}
