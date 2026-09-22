package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDocsRedirect(t *testing.T) {
	tests := []struct {
		name         string
		path         string
		wantLocation string
	}{
		{name: "unprefixed request redirects to docs subtree", path: "/docs", wantLocation: "/docs/"},
		{name: "api-prefixed request redirects with api prefix preserved", path: "/api/docs", wantLocation: "/api/docs/"},
	}

	mux := http.NewServeMux()
	registerDocsRoutes(mux)
	handler := stripAPIPrefix(mux)

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, test.path, nil))

			if recorder.Code != http.StatusFound {
				t.Fatalf("status = %d, want %d", recorder.Code, http.StatusFound)
			}
			if got := recorder.Header().Get("Location"); got != test.wantLocation {
				t.Fatalf("Location = %q, want %q", got, test.wantLocation)
			}
		})
	}
}
