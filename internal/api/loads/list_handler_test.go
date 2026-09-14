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

type fakeLoadService struct {
	listLoads func(context.Context, QueryRequest) ([]Load, int64, int64, error)
	getLoad   func(context.Context, string) (Load, error)
	getStats  func(context.Context, string) (LoadStats, error)
}

func (f fakeLoadService) ListLoads(ctx context.Context, req QueryRequest) ([]Load, int64, int64, error) {
	return f.listLoads(ctx, req)
}

func (f fakeLoadService) GetLoad(ctx context.Context, id string) (Load, error) {
	return f.getLoad(ctx, id)
}

func (f fakeLoadService) GetLoadStats(ctx context.Context, quickSearch string) (LoadStats, error) {
	return f.getStats(ctx, quickSearch)
}

func TestListHandler(t *testing.T) {
	row := Load{ID: "LD-000001", Status: "Available"}

	t.Run("success", func(t *testing.T) {
		var received QueryRequest
		service := fakeLoadService{
			listLoads: func(_ context.Context, req QueryRequest) ([]Load, int64, int64, error) {
				received = req
				return []Load{row}, 100, 40, nil
			},
		}
		req := httptest.NewRequest(http.MethodGet, "/loads?startRow=25&endRow=50", nil)
		recorder := httptest.NewRecorder()

		ListHandler(service)(recorder, req)

		if recorder.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
		}
		var response ListResponse
		if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if received.StartRow != 25 || received.EndRow != 50 {
			t.Fatalf("service request range = [%d, %d), want [25, 50)", received.StartRow, received.EndRow)
		}
		if response.RequestURL != "/loads?startRow=25&endRow=50" {
			t.Fatalf("requestURL = %q, want original request URL", response.RequestURL)
		}
		if response.Meta.NumTotal != 100 || response.Meta.NumResults != 40 || response.LastRow != 40 {
			t.Fatalf("response counts = total %d, results %d, lastRow %d; want 100, 40, 40", response.Meta.NumTotal, response.Meta.NumResults, response.LastRow)
		}
		if !reflect.DeepEqual(response.Rows, []Load{row}) {
			t.Fatalf("rows = %#v, want %#v", response.Rows, []Load{row})
		}
	})

	t.Run("invalid query returns bad request without calling service", func(t *testing.T) {
		called := false
		service := fakeLoadService{
			listLoads: func(context.Context, QueryRequest) ([]Load, int64, int64, error) {
				called = true
				return nil, 0, 0, nil
			},
		}
		req := httptest.NewRequest(http.MethodGet, "/loads?startRow=nope", nil)
		recorder := httptest.NewRecorder()

		ListHandler(service)(recorder, req)

		assertErrorResponse(t, recorder, http.StatusBadRequest, "LOAD_QUERY_FAILED")
		if called {
			t.Fatal("service was called for an invalid query")
		}
	})

	t.Run("invalid service query returns bad request", func(t *testing.T) {
		service := fakeLoadService{
			listLoads: func(context.Context, QueryRequest) ([]Load, int64, int64, error) {
				return nil, 0, 0, &InvalidQueryError{err: errors.New("bad filter")}
			},
		}
		recorder := httptest.NewRecorder()

		ListHandler(service)(recorder, httptest.NewRequest(http.MethodGet, "/loads", nil))

		assertErrorResponse(t, recorder, http.StatusBadRequest, "LOAD_QUERY_FAILED")
	})

	t.Run("service failure returns internal server error", func(t *testing.T) {
		service := fakeLoadService{
			listLoads: func(context.Context, QueryRequest) ([]Load, int64, int64, error) {
				return nil, 0, 0, errors.New("database unavailable")
			},
		}
		recorder := httptest.NewRecorder()

		ListHandler(service)(recorder, httptest.NewRequest(http.MethodGet, "/loads", nil))

		assertErrorResponse(t, recorder, http.StatusInternalServerError, "LOAD_QUERY_FAILED")
	})

	t.Run("success with next page", func(t *testing.T) {
		service := fakeLoadService{
			listLoads: func(context.Context, QueryRequest) ([]Load, int64, int64, error) {
				return nil, 100, 100, nil
			},
		}
		recorder := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/loads?quickSearch=chicago&status=Available&startRow=25&endRow=50", nil)

		ListHandler(service)(recorder, req)

		if recorder.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
		}
		var response ListResponse
		if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if !response.Meta.HasNextPage {
			t.Fatal("hasNextPage = false, want true")
		}
		if response.Meta.NextPageURL == nil {
			t.Fatal("nextPageURL = nil, want next page URL")
		}
		if *response.Meta.NextPageURL != "/loads?quickSearch=chicago&status=Available&startRow=50&endRow=75" {
			t.Fatalf("nextPageURL = %q, want next page URL", *response.Meta.NextPageURL)
		}
		if response.Meta.NumPages != 4 || response.Meta.CurrentPage != 2 {
			t.Fatalf("page metadata = pages %d, current %d; want 4, 2", response.Meta.NumPages, response.Meta.CurrentPage)
		}
	})
}

func assertErrorResponse(t *testing.T, recorder *httptest.ResponseRecorder, wantStatus int, wantCode string) {
	t.Helper()
	if recorder.Code != wantStatus {
		t.Fatalf("status = %d, want %d", recorder.Code, wantStatus)
	}
	var response ErrorResponse
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	if response.Error.Code != wantCode {
		t.Fatalf("error code = %q, want %q", response.Error.Code, wantCode)
	}
}

func TestListHandlerHelpers(t *testing.T) {
	t.Run("pagination helpers", func(t *testing.T) {
		if got := numPages(0, 25); got != 0 {
			t.Errorf("numPages(0, 25) = %d, want 0", got)
		}
		if got := numPages(26, 25); got != 2 {
			t.Errorf("numPages(26, 25) = %d, want 2", got)
		}
		if got := numPages(1, 0); got != 0 {
			t.Errorf("numPages(1, 0) = %d, want 0", got)
		}
		if got := currentPage(0, 25); got != 1 {
			t.Errorf("currentPage(0, 25) = %d, want 1", got)
		}
		if got := currentPage(25, 25); got != 2 {
			t.Errorf("currentPage(25, 25) = %d, want 2", got)
		}
		if got := currentPage(25, 0); got != 1 {
			t.Errorf("currentPage(25, 0) = %d, want 1", got)
		}
	})

	t.Run("request URL preserves raw query", func(t *testing.T) {
		withQuery := httptest.NewRequest(http.MethodGet, "/loads?q=hello%20world", nil)
		if got := requestedURL(withQuery); got != "/loads?q=hello%20world" {
			t.Errorf("requestedURL() = %q, want raw query", got)
		}
		withoutQuery := httptest.NewRequest(http.MethodGet, "/loads", nil)
		if got := requestedURL(withoutQuery); got != "/loads" {
			t.Errorf("requestedURL() = %q, want path only", got)
		}
	})

	t.Run("echo query initializes nil values", func(t *testing.T) {
		got := echoQuery(QueryRequest{})
		if got.SortModel == nil || got.FilterModel == nil {
			t.Fatalf("echoQuery() = %#v, want initialized sort and filter values", got)
		}
		if len(got.SortModel) != 0 || len(got.FilterModel) != 0 {
			t.Fatalf("echoQuery() = %#v, want empty sort and filter values", got)
		}
	})

	t.Run("echo query includes sort", func(t *testing.T) {
		want := SortModelEntry{ColID: "price", Sort: "desc"}
		got := echoQuery(QueryRequest{Sort: &want})
		if len(got.SortModel) != 1 || got.SortModel[0] != want {
			t.Fatalf("SortModel = %#v, want [%#v]", got.SortModel, want)
		}
	})

	t.Run("next page URL orders known and remaining params", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/loads?z=last&sortModel=%5B%5D&status=Available&quickSearch=hello+world&a=first", nil)
		parsed := QueryRequest{StartRow: 0, EndRow: 25}
		got := nextPageURL(req, parsed, true)
		if got == nil {
			t.Fatal("nextPageURL() = nil, want URL")
		}
		want := "/loads?quickSearch=hello+world&status=Available&sortModel=%5B%5D&a=first&z=last&startRow=25&endRow=50"
		if *got != want {
			t.Fatalf("nextPageURL() = %q, want %q", *got, want)
		}
		if got := nextPageURL(req, parsed, false); got != nil {
			t.Fatalf("nextPageURL(false) = %q, want nil", *got)
		}
	})
}
