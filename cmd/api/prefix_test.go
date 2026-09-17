package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestStripAPIPrefix(t *testing.T) {
	tests := []struct {
		name      string
		path      string
		query     string
		wantPath  string
		wantQuery string
	}{
		{name: "prefixed path", path: "/api/health", wantPath: "/health"},
		{name: "unprefixed path passes through", path: "/health", wantPath: "/health"},
		{name: "prefixed path with query", path: "/api/loads", query: "startRow=0&endRow=10", wantPath: "/loads", wantQuery: "startRow=0&endRow=10"},
		{name: "bare api rewrites to root", path: "/api", wantPath: "/"},
		{name: "api with trailing slash rewrites to root", path: "/api/", wantPath: "/"},
		{name: "look-alike prefix left untouched", path: "/apix/foo", wantPath: "/apix/foo"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var gotPath, gotQuery string
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotPath = r.URL.Path
				gotQuery = r.URL.RawQuery
				w.WriteHeader(http.StatusOK)
			})

			target := test.path
			if test.query != "" {
				target += "?" + test.query
			}
			recorder := httptest.NewRecorder()
			stripAPIPrefix(next).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, target, nil))

			if gotPath != test.wantPath {
				t.Fatalf("path = %q, want %q", gotPath, test.wantPath)
			}
			if gotQuery != test.wantQuery {
				t.Fatalf("query = %q, want %q", gotQuery, test.wantQuery)
			}
			if recorder.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
			}
		})
	}
}

func TestHadAPIPrefix(t *testing.T) {
	tests := []struct {
		name string
		path string
		want bool
	}{
		{name: "prefixed path", path: "/api/health", want: true},
		{name: "bare api", path: "/api", want: true},
		{name: "unprefixed path", path: "/health", want: false},
		{name: "look-alike prefix", path: "/apix/foo", want: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var got bool
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				got = hadAPIPrefix(r)
				w.WriteHeader(http.StatusOK)
			})

			recorder := httptest.NewRecorder()
			stripAPIPrefix(next).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, test.path, nil))

			if got != test.want {
				t.Fatalf("hadAPIPrefix = %v, want %v", got, test.want)
			}
		})
	}
}
