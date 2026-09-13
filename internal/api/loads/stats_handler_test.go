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
		want := LoadStats{
			NumTotal: 100000,
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
		service := fakeLoadService{getStats: func(context.Context) (LoadStats, error) {
			return want, nil
		}}
		recorder := httptest.NewRecorder()

		StatsHandler(service)(recorder, httptest.NewRequest(http.MethodGet, "/loads/stats", nil))

		if recorder.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
		}
		if got := recorder.Header().Get("Content-Type"); got != "application/json" {
			t.Fatalf("Content-Type = %q, want application/json", got)
		}
		var response StatsResponse
		if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if response.RequestURL != "/loads/stats" || response.Meta.NumTotal != want.NumTotal || response.Meta.Timestamp.IsZero() {
			t.Fatalf("response metadata = %#v, want request URL, total, and timestamp", response)
		}
		if response.Meta.ResponseTimeMs < 1 {
			t.Fatalf("responseTimeMs = %d, want at least 1", response.Meta.ResponseTimeMs)
		}
		if !reflect.DeepEqual(response.Stats.Totals.EquipmentType, want.EquipmentType) || !reflect.DeepEqual(response.Stats.Totals.Status, want.Status) {
			t.Fatalf("response totals = %#v, want %#v", response.Stats.Totals, want)
		}
	})

	t.Run("service failure", func(t *testing.T) {
		service := fakeLoadService{getStats: func(context.Context) (LoadStats, error) {
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
