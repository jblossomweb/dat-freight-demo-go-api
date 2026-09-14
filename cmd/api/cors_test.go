package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCORSMiddleware(t *testing.T) {
	const allowedOrigin = "http://localhost:5173"
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := corsMiddleware([]string{allowedOrigin}, next)

	t.Run("allowed origin", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/loads", nil)
		request.Header.Set("Origin", allowedOrigin)

		handler.ServeHTTP(recorder, request)

		if recorder.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
		}
		if got := recorder.Header().Get("Access-Control-Allow-Origin"); got != allowedOrigin {
			t.Fatalf("Access-Control-Allow-Origin = %q, want %q", got, allowedOrigin)
		}
		if got := recorder.Header().Get("Vary"); got != "Origin" {
			t.Fatalf("Vary = %q, want Origin", got)
		}
	})

	t.Run("allowed preflight", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodOptions, "/loads/stats", nil)
		request.Header.Set("Origin", allowedOrigin)
		request.Header.Set("Access-Control-Request-Method", http.MethodGet)

		handler.ServeHTTP(recorder, request)

		if recorder.Code != http.StatusNoContent {
			t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNoContent)
		}
		if got := recorder.Header().Get("Access-Control-Allow-Origin"); got != allowedOrigin {
			t.Fatalf("Access-Control-Allow-Origin = %q, want %q", got, allowedOrigin)
		}
		if got := recorder.Header().Get("Access-Control-Allow-Methods"); got != "GET, OPTIONS" {
			t.Fatalf("Access-Control-Allow-Methods = %q, want GET, OPTIONS", got)
		}
	})

	t.Run("disallowed preflight", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodOptions, "/loads", nil)
		request.Header.Set("Origin", "https://example.com")
		request.Header.Set("Access-Control-Request-Method", http.MethodGet)

		handler.ServeHTTP(recorder, request)

		if recorder.Code != http.StatusForbidden {
			t.Fatalf("status = %d, want %d", recorder.Code, http.StatusForbidden)
		}
		if got := recorder.Header().Get("Access-Control-Allow-Origin"); got != "" {
			t.Fatalf("Access-Control-Allow-Origin = %q, want empty", got)
		}
	})

	t.Run("request without origin", func(t *testing.T) {
		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/health", nil))

		if recorder.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
		}
		if got := recorder.Header().Get("Access-Control-Allow-Origin"); got != "" {
			t.Fatalf("Access-Control-Allow-Origin = %q, want empty", got)
		}
	})
}