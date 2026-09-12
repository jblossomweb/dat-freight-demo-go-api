package loads

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestGetHandler(t *testing.T) {
	row := Load{ID: "LD-000001", Status: "Available"}

	t.Run("success with query id", func(t *testing.T) {
		var receivedID string
		service := fakeLoadService{
			getLoad: func(_ context.Context, id string) (Load, error) {
				receivedID = id
				return row, nil
			},
		}
		recorder := httptest.NewRecorder()

		GetHandler(service)(recorder, httptest.NewRequest(http.MethodGet, "/load?id=LD-000001", nil))

		if recorder.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
		}
		if receivedID != "LD-000001" {
			t.Fatalf("service id = %q, want LD-000001", receivedID)
		}
		var response GetResponse
		if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if !reflect.DeepEqual(response.Result, row) {
			t.Fatalf("result = %#v, want %#v", response.Result, row)
		}
	})

	t.Run("success with path id", func(t *testing.T) {
		var receivedID string
		service := fakeLoadService{
			getLoad: func(_ context.Context, id string) (Load, error) {
				receivedID = id
				return row, nil
			},
		}
		req := httptest.NewRequest(http.MethodGet, "/load/LD-000001", nil)
		req.SetPathValue("id", "LD-000001")
		recorder := httptest.NewRecorder()

		GetHandler(service)(recorder, req)

		if recorder.Code != http.StatusOK || receivedID != "LD-000001" {
			t.Fatalf("status = %d, id = %q; want 200 and LD-000001", recorder.Code, receivedID)
		}
	})

	t.Run("missing id returns bad request without calling service", func(t *testing.T) {
		called := false
		service := fakeLoadService{
			getLoad: func(context.Context, string) (Load, error) {
				called = true
				return Load{}, nil
			},
		}
		recorder := httptest.NewRecorder()

		GetHandler(service)(recorder, httptest.NewRequest(http.MethodGet, "/load", nil))

		assertErrorResponse(t, recorder, http.StatusBadRequest, "LOAD_ID_REQUIRED")
		if called {
			t.Fatal("service was called without an id")
		}
	})

	t.Run("not found returns not found", func(t *testing.T) {
		service := fakeLoadService{
			getLoad: func(context.Context, string) (Load, error) {
				return Load{}, ErrNotFound
			},
		}
		recorder := httptest.NewRecorder()

		GetHandler(service)(recorder, httptest.NewRequest(http.MethodGet, "/load?id=missing", nil))

		assertErrorResponse(t, recorder, http.StatusNotFound, "LOAD_NOT_FOUND")
	})

	t.Run("service failure returns internal server error", func(t *testing.T) {
		service := fakeLoadService{
			getLoad: func(context.Context, string) (Load, error) {
				return Load{}, errors.New("database unavailable")
			},
		}
		recorder := httptest.NewRecorder()

		GetHandler(service)(recorder, httptest.NewRequest(http.MethodGet, "/load?id=LD-000001", nil))

		assertErrorResponse(t, recorder, http.StatusInternalServerError, "LOAD_QUERY_FAILED")
	})
}
