package loads

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

func TestStatsHandler(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		var receivedQuickSearch string
		want := LoadStats{
			NumTotal:   100000,
			NumResults: 1200,
			EquipmentType: []StatCount{
				{Label: "Flatbed", Value: 33000},
				{Label: "Reefer", Value: 34000},
				{Label: "Van", Value: 33000},
			},
			Status: []StatCount{
				{Label: "Available", Value: 40000},
				{Label: "In Transit", Value: 30000},
				{Label: "Delivered", Value: 30000},
			},
		}
		service := fakeLoadService{getStats: func(_ context.Context, quickSearch string) (LoadStats, error) {
			receivedQuickSearch = quickSearch
			return want, nil
		}}
		recorder := httptest.NewRecorder()

		StatsHandler(service)(recorder, httptest.NewRequest(http.MethodGet, "/loads/stats?q=chicago", nil))

		if recorder.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
		}
		if got := recorder.Header().Get("Content-Type"); got != "application/json" {
			t.Fatalf("Content-Type = %q, want application/json", got)
		}
		responseBody := recorder.Body.String()
		resultsIndex := strings.Index(responseBody, `"numResults"`)
		totalIndex := strings.Index(responseBody, `"numTotal"`)
		if resultsIndex < 0 || totalIndex < 0 || resultsIndex > totalIndex {
			t.Fatalf("response metadata order = %s, want numResults before numTotal", responseBody)
		}
		var response StatsResponse
		if err := json.Unmarshal([]byte(responseBody), &response); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if response.RequestURL != "/loads/stats?q=chicago" || response.Query.QuickSearch != "chicago" || receivedQuickSearch != "chicago" || response.Meta.NumTotal != want.NumTotal || response.Meta.NumResults != want.NumResults || response.Meta.Timestamp.IsZero() {
			t.Fatalf("response metadata = %#v, want request URL, total, and timestamp", response)
		}
		if response.Meta.ResponseTimeMs < 1 {
			t.Fatalf("responseTimeMs = %d, want at least 1", response.Meta.ResponseTimeMs)
		}
		if !reflect.DeepEqual(response.Stats.Totals.EquipmentType, want.EquipmentType) || !reflect.DeepEqual(response.Stats.Totals.Status, want.Status) {
			t.Fatalf("response totals = %#v, want %#v", response.Stats.Totals, want)
		}
	})

	t.Run("quickSearch wins over q", func(t *testing.T) {
		var received string
		service := fakeLoadService{getStats: func(_ context.Context, quickSearch string) (LoadStats, error) {
			received = quickSearch
			return LoadStats{}, nil
		}}
		recorder := httptest.NewRecorder()

		StatsHandler(service)(recorder, httptest.NewRequest(http.MethodGet, "/loads/stats?q=denver&quickSearch=chicago", nil))

		var response StatsResponse
		if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if received != "chicago" || response.Query.QuickSearch != "chicago" {
			t.Fatalf("quickSearch = service %q, echo %q; want chicago", received, response.Query.QuickSearch)
		}
	})

	t.Run("empty query echo", func(t *testing.T) {
		service := fakeLoadService{getStats: func(context.Context, string) (LoadStats, error) {
			return LoadStats{}, nil
		}}
		recorder := httptest.NewRecorder()

		StatsHandler(service)(recorder, httptest.NewRequest(http.MethodGet, "/loads/stats", nil))

		var response StatsResponse
		if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if response.Query.QuickSearch != "" {
			t.Fatalf("quickSearch = %q, want empty", response.Query.QuickSearch)
		}
	})

	t.Run("service failure", func(t *testing.T) {
		service := fakeLoadService{getStats: func(context.Context, string) (LoadStats, error) {
			return LoadStats{}, errors.New("database credentials must not be returned")
		}}
		recorder := httptest.NewRecorder()

		StatsHandler(service)(recorder, httptest.NewRequest(http.MethodGet, "/loads/stats", nil))

		if recorder.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d, want %d", recorder.Code, http.StatusInternalServerError)
		}
		var response ErrorResponse
		if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
			t.Fatalf("decode error response: %v", err)
		}
		if response.Error.Code != "LOAD_STATS_FAILED" || strings.Contains(response.Error.Message, "credentials") {
			t.Fatalf("error response = %#v, want safe LOAD_STATS_FAILED response", response)
		}
	})
}
