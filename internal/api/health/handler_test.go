package health

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

type fakeMongoPinger struct {
	err    error
	called bool
}

func (f *fakeMongoPinger) Ping(context.Context) error {
	f.called = true
	return f.err
}

func TestHandler(t *testing.T) {
	t.Run("ping succeeds", func(t *testing.T) {
		pinger := &fakeMongoPinger{}
		recorder := httptest.NewRecorder()

		Handler(pinger)(recorder, httptest.NewRequest(http.MethodGet, "/health", nil))

		if recorder.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
		}
		if !pinger.called {
			t.Fatal("ping was not called")
		}
		var response Response
		if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if response != (Response{Status: "ok", Mongo: "ok"}) {
			t.Fatalf("response = %#v, want ok response", response)
		}
	})

	t.Run("ping fails", func(t *testing.T) {
		pinger := &fakeMongoPinger{err: errors.New("mongo unavailable")}
		recorder := httptest.NewRecorder()

		Handler(pinger)(recorder, httptest.NewRequest(http.MethodGet, "/health", nil))

		if recorder.Code != http.StatusServiceUnavailable {
			t.Fatalf("status = %d, want %d", recorder.Code, http.StatusServiceUnavailable)
		}
		var response UnavailableResponse
		if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if response != (UnavailableResponse{Status: "degraded", Mongo: "unreachable"}) {
			t.Fatalf("response = %#v, want degraded response", response)
		}
	})
}
