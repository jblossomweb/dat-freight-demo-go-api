package loads

import (
	"fmt"
	"regexp"
	"strconv"

	"encoding/json"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// BuildMongoFilter translates the parsed filter model and quicksearch term into a Mongo query.
func BuildMongoFilter(req QueryRequest) (bson.M, error) {
	var and []bson.M

	for field, entry := range req.Filters {
		cond, err := buildFieldFilter(field, entry)
		if err != nil {
			return nil, err
		}
		and = append(and, cond)
	}

	if req.QuickSearch != "" {
		and = append(and, buildQuickSearch(req.QuickSearch))
	}

	if len(and) == 0 {
		return bson.M{}, nil
	}
	return bson.M{"$and": and}, nil
}

// buildQuickSearch matches the client's AG Grid quick filter, which searches every
// column by default. String columns get a direct regex match; numeric columns
// (stored as BSON numbers) are matched via $expr/$toString since Mongo regex only
// applies to strings.
func buildQuickSearch(term string) bson.M {
	escaped := regexp.QuoteMeta(term)
	pattern := bson.Regex{Pattern: escaped, Options: "i"}

	or := make([]bson.M, 0, len(stringFields)+len(numberFields))
	for _, field := range stringFields {
		or = append(or, bson.M{field: pattern})
	}
	for field := range numberFields {
		or = append(or, bson.M{
			"$expr": bson.M{
				"$regexMatch": bson.M{
					"input":   bson.M{"$toString": "$" + field},
					"regex":   escaped,
					"options": "i",
				},
			},
		})
	}
	return bson.M{"$or": or}
}

func buildFieldFilter(field string, entry FilterModelEntry) (bson.M, error) {
	if entry.FilterType == "set" {
		return buildSetFilter(field, entry.Values)
	}
	if numberFields[field] || entry.FilterType == "number" {
		return buildNumberFilter(field, entry)
	}
	if entry.FilterType == "date" {
		return buildDateFilter(field, entry)
	}
	return buildTextFilter(field, entry)
}

// buildSetFilter builds an "$in" match. Numeric fields get their string values
// (from the alias query params) coerced to numbers so type comparisons work.
func buildSetFilter(field string, values []string) (bson.M, error) {
	if !numberFields[field] {
		return bson.M{field: bson.M{"$in": values}}, nil
	}
	nums := make([]float64, len(values))
	for i, v := range values {
		n, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid numeric value %q for field %q", v, field)
		}
		nums[i] = n
	}
	return bson.M{field: bson.M{"$in": nums}}, nil
}

func buildTextFilter(field string, entry FilterModelEntry) (bson.M, error) {
	value, _ := entry.Filter.(string)
	quoted := regexp.QuoteMeta(value)

	switch entry.Type {
	case "equals":
		return bson.M{field: bson.Regex{Pattern: "^" + quoted + "$", Options: "i"}}, nil
	case "notEqual":
		return bson.M{field: bson.M{"$not": bson.Regex{Pattern: "^" + quoted + "$", Options: "i"}}}, nil
	case "contains", "":
		return bson.M{field: bson.Regex{Pattern: quoted, Options: "i"}}, nil
	case "notContains":
		return bson.M{field: bson.M{"$not": bson.Regex{Pattern: quoted, Options: "i"}}}, nil
	case "startsWith":
		return bson.M{field: bson.Regex{Pattern: "^" + quoted, Options: "i"}}, nil
	case "endsWith":
		return bson.M{field: bson.Regex{Pattern: quoted + "$", Options: "i"}}, nil
	default:
		return nil, fmt.Errorf("unsupported text filter type %q for field %q", entry.Type, field)
	}
}

func buildNumberFilter(field string, entry FilterModelEntry) (bson.M, error) {
	value, ok := toFloat64(entry.Filter)
	if !ok {
		return nil, fmt.Errorf("invalid numeric filter value for field %q", field)
	}

	switch entry.Type {
	case "equals":
		return bson.M{field: value}, nil
	case "notEqual":
		return bson.M{field: bson.M{"$ne": value}}, nil
	case "lessThan":
		return bson.M{field: bson.M{"$lt": value}}, nil
	case "lessThanOrEqual":
		return bson.M{field: bson.M{"$lte": value}}, nil
	case "greaterThan":
		return bson.M{field: bson.M{"$gt": value}}, nil
	case "greaterThanOrEqual":
		return bson.M{field: bson.M{"$gte": value}}, nil
	case "inRange":
		to, ok := toFloat64(entry.FilterTo)
		if !ok {
			return nil, fmt.Errorf("invalid filterTo for range filter on field %q", field)
		}
		return bson.M{field: bson.M{"$gte": value, "$lte": to}}, nil
	default:
		return nil, fmt.Errorf("unsupported number filter type %q for field %q", entry.Type, field)
	}
}

func buildDateFilter(field string, entry FilterModelEntry) (bson.M, error) {
	switch entry.Type {
	case "equals":
		return bson.M{field: entry.DateFrom}, nil
	case "notEqual":
		return bson.M{field: bson.M{"$ne": entry.DateFrom}}, nil
	case "lessThan":
		return bson.M{field: bson.M{"$lt": entry.DateFrom}}, nil
	case "greaterThan":
		return bson.M{field: bson.M{"$gt": entry.DateFrom}}, nil
	case "inRange":
		return bson.M{field: bson.M{"$gte": entry.DateFrom, "$lte": entry.DateTo}}, nil
	default:
		return nil, fmt.Errorf("unsupported date filter type %q for field %q", entry.Type, field)
	}
}

func toFloat64(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case int:
		return float64(n), true
	case json.Number:
		f, err := n.Float64()
		return f, err == nil
	default:
		return 0, false
	}
}
