package loads

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRegisterRoutes(t *testing.T) {
	service := fakeLoadService{
		listLoads: func(context.Context, QueryRequest) ([]Load, int64, int64, error) {
			return nil, 0, 0, nil
		},
		getLoad: func(context.Context, string) (Load, error) {
			return Load{ID: "LD-000001"}, nil
		},
	}
	mux := http.NewServeMux()
	RegisterRoutes(mux, service)

	tests := []struct {
		name string
		path string
		want int
	}{
		{name: "list route", path: "/loads", want: http.StatusOK},
		{name: "load query route", path: "/load?id=LD-000001", want: http.StatusOK},
		{name: "load path route", path: "/load/LD-000001", want: http.StatusOK},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			mux.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, test.path, nil))
			if recorder.Code != test.want {
				t.Fatalf("status = %d, want %d", recorder.Code, test.want)
			}
		})
	}
}

func TestRegisterRoutesRejectsUnsupportedMethods(t *testing.T) {
	mux := http.NewServeMux()
	RegisterRoutes(mux, fakeLoadService{
		listLoads: func(context.Context, QueryRequest) ([]Load, int64, int64, error) {
			return nil, 0, 0, nil
		},
		getLoad: func(context.Context, string) (Load, error) {
			return Load{}, errors.New("unexpected call")
		},
	})

	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/loads", nil))
	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusMethodNotAllowed)
	}
}
