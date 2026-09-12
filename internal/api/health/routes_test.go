package health

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

type routePinger struct{}

func (routePinger) Ping(context.Context) error {
	return nil
}

func TestRegisterRoutes(t *testing.T) {
	mux := http.NewServeMux()
	RegisterRoutes(mux, routePinger{})

	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/health", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
}

func TestRegisterRoutesRejectsUnsupportedMethods(t *testing.T) {
	mux := http.NewServeMux()
	RegisterRoutes(mux, routePinger{})

	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/health", nil))
	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusMethodNotAllowed)
	}
}
