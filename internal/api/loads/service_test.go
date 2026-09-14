package loads

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type fakeLoadRepository struct {
	countAll      func(context.Context) (int64, error)
	countMatching func(context.Context, bson.M) (int64, error)
	find          func(context.Context, bson.M, *SortModelEntry, int, int) ([]Load, error)
	findByID      func(context.Context, string) (Load, error)
	getStats      func(context.Context, bson.M) (LoadStats, error)
}

func (f fakeLoadRepository) CountAll(ctx context.Context) (int64, error) {
	return f.countAll(ctx)
}

func (f fakeLoadRepository) CountMatching(ctx context.Context, filter bson.M) (int64, error) {
	return f.countMatching(ctx, filter)
}

func (f fakeLoadRepository) Find(ctx context.Context, filter bson.M, sort *SortModelEntry, skip, limit int) ([]Load, error) {
	return f.find(ctx, filter, sort, skip, limit)
}

func (f fakeLoadRepository) FindByID(ctx context.Context, id string) (Load, error) {
	return f.findByID(ctx, id)
}

func (f fakeLoadRepository) GetStats(ctx context.Context, filter bson.M) (LoadStats, error) {
	return f.getStats(ctx, filter)
}

func TestServiceListLoads(t *testing.T) {
	row := Load{ID: "LD-000001", Status: "Available"}
	req := QueryRequest{
		StartRow: 25,
		EndRow:   50,
		Sort:     &SortModelEntry{ColID: "price", Sort: "desc"},
		Filters: map[string]FilterModelEntry{
			"status": {FilterType: "text", Type: "equals", Filter: "Available"},
		},
	}

	t.Run("success", func(t *testing.T) {
		var receivedFilter bson.M
		var receivedSort *SortModelEntry
		var receivedSkip, receivedLimit int
		repo := fakeLoadRepository{
			countAll: func(context.Context) (int64, error) { return 100, nil },
			countMatching: func(_ context.Context, filter bson.M) (int64, error) {
				receivedFilter = filter
				return 40, nil
			},
			find: func(_ context.Context, filter bson.M, sort *SortModelEntry, skip, limit int) ([]Load, error) {
				receivedFilter = filter
				receivedSort = sort
				receivedSkip = skip
				receivedLimit = limit
				return []Load{row}, nil
			},
		}

		rows, total, filtered, err := NewService(repo, 0).ListLoads(context.Background(), req)
		if err != nil {
			t.Fatalf("ListLoads() error = %v", err)
		}
		if !reflect.DeepEqual(rows, []Load{row}) || total != 100 || filtered != 40 {
			t.Fatalf("ListLoads() = rows %#v, total %d, filtered %d; want row, 100, 40", rows, total, filtered)
		}
		if receivedFilter == nil || receivedSort == nil || receivedSort.ColID != "price" || receivedSort.Sort != "desc" {
			t.Fatalf("repository sort/filter = %#v/%#v, want status filter and price desc", receivedFilter, receivedSort)
		}
		if receivedSkip != 25 || receivedLimit != 25 {
			t.Fatalf("repository page = skip %d, limit %d; want 25, 25", receivedSkip, receivedLimit)
		}
	})

	t.Run("invalid filter returns InvalidQueryError", func(t *testing.T) {
		wantErr := errors.New("unsupported filter")
		repo := fakeLoadRepository{
			countAll:      func(context.Context) (int64, error) { t.Fatal("CountAll called"); return 0, nil },
			countMatching: func(context.Context, bson.M) (int64, error) { t.Fatal("CountMatching called"); return 0, nil },
			find: func(context.Context, bson.M, *SortModelEntry, int, int) ([]Load, error) {
				t.Fatal("Find called")
				return nil, nil
			},
		}
		_, _, _, err := NewService(repo, 0).ListLoads(context.Background(), QueryRequest{Filters: map[string]FilterModelEntry{
			"status": {FilterType: "text", Type: "unsupported", Filter: "Available"},
		}})
		var invalid *InvalidQueryError
		if !errors.As(err, &invalid) {
			t.Fatalf("error = %v, want InvalidQueryError", err)
		}
		if !errors.Is(&InvalidQueryError{err: wantErr}, wantErr) {
			t.Fatal("InvalidQueryError did not unwrap its underlying error")
		}
	})

	t.Run("CountAll failure", func(t *testing.T) {
		wantErr := errors.New("count all failed")
		repo := fakeLoadRepository{
			countAll: func(context.Context) (int64, error) { return 0, wantErr },
		}
		_, _, _, err := NewService(repo, 0).ListLoads(context.Background(), QueryRequest{})
		if !errors.Is(err, wantErr) {
			t.Fatalf("error = %v, want %v", err, wantErr)
		}
	})

	t.Run("CountMatching failure", func(t *testing.T) {
		wantErr := errors.New("count matching failed")
		repo := fakeLoadRepository{
			countAll:      func(context.Context) (int64, error) { return 100, nil },
			countMatching: func(context.Context, bson.M) (int64, error) { return 0, wantErr },
		}
		_, _, _, err := NewService(repo, 0).ListLoads(context.Background(), QueryRequest{})
		if !errors.Is(err, wantErr) {
			t.Fatalf("error = %v, want %v", err, wantErr)
		}
	})

	t.Run("Find failure", func(t *testing.T) {
		wantErr := errors.New("find failed")
		repo := fakeLoadRepository{
			countAll:      func(context.Context) (int64, error) { return 100, nil },
			countMatching: func(context.Context, bson.M) (int64, error) { return 40, nil },
			find:          func(context.Context, bson.M, *SortModelEntry, int, int) ([]Load, error) { return nil, wantErr },
		}
		_, _, _, err := NewService(repo, 0).ListLoads(context.Background(), QueryRequest{})
		if !errors.Is(err, wantErr) {
			t.Fatalf("error = %v, want %v", err, wantErr)
		}
	})
}

func TestServiceGetLoad(t *testing.T) {
	row := Load{ID: "LD-000001", Status: "Available"}

	t.Run("success", func(t *testing.T) {
		var receivedID string
		repo := fakeLoadRepository{
			findByID: func(_ context.Context, id string) (Load, error) {
				receivedID = id
				return row, nil
			},
		}
		got, err := NewService(repo, 0).GetLoad(context.Background(), row.ID)
		if err != nil || !reflect.DeepEqual(got, row) || receivedID != row.ID {
			t.Fatalf("GetLoad() = %#v, %v, id %q; want %#v, nil, %q", got, err, receivedID, row, row.ID)
		}
	})

	t.Run("propagates repository error", func(t *testing.T) {
		wantErr := errors.New("find by id failed")
		repo := fakeLoadRepository{
			findByID: func(context.Context, string) (Load, error) { return Load{}, wantErr },
		}
		_, err := NewService(repo, 0).GetLoad(context.Background(), row.ID)
		if !errors.Is(err, wantErr) {
			t.Fatalf("error = %v, want %v", err, wantErr)
		}
	})
}

func TestServiceGetLoadStats(t *testing.T) {
	want := LoadStats{
		NumTotal:      10,
		NumResults:    3,
		EquipmentType: []StatCount{{Label: "Flatbed", Value: 1}},
		Status:        []StatCount{{Label: "Available", Value: 2}},
	}

	t.Run("passes quick search filter", func(t *testing.T) {
		var received bson.M
		repo := fakeLoadRepository{getStats: func(_ context.Context, filter bson.M) (LoadStats, error) {
			received = filter
			return want, nil
		}}

		if _, err := NewService(repo, 0).GetLoadStats(context.Background(), "chicago"); err != nil {
			t.Fatalf("GetLoadStats() error = %v", err)
		}
		if len(received) == 0 {
			t.Fatal("repository filter is empty, want quick-search filter")
		}
	})

	t.Run("cache hit and expiry", func(t *testing.T) {
		var calls int
		now := time.Date(2026, time.September, 13, 8, 0, 0, 0, time.UTC)
		repo := fakeLoadRepository{getStats: func(context.Context, bson.M) (LoadStats, error) {
			calls++
			return want, nil
		}}
		service := NewService(repo, 5*time.Minute)
		service.now = func() time.Time { return now }

		first, err := service.GetLoadStats(context.Background(), "chicago")
		if err != nil {
			t.Fatalf("GetLoadStats() error = %v", err)
		}
		first.EquipmentType[0].Value = 99
		now = now.Add(4 * time.Minute)
		second, err := service.GetLoadStats(context.Background(), "chicago")
		if err != nil {
			t.Fatalf("GetLoadStats() cached error = %v", err)
		}
		if calls != 1 || second.EquipmentType[0].Value != 1 {
			t.Fatalf("cached result = %#v after %d calls, want isolated cached value after 1 call", second, calls)
		}

		now = now.Add(time.Minute)
		if _, err := service.GetLoadStats(context.Background(), "chicago"); err != nil {
			t.Fatalf("GetLoadStats() refresh error = %v", err)
		}
		if calls != 2 {
			t.Fatalf("repository calls = %d, want 2 after TTL expiry", calls)
		}
	})

	t.Run("disabled cache", func(t *testing.T) {
		var calls int
		repo := fakeLoadRepository{getStats: func(context.Context, bson.M) (LoadStats, error) {
			calls++
			return want, nil
		}}
		service := NewService(repo, 0)
		for range 2 {
			if _, err := service.GetLoadStats(context.Background(), ""); err != nil {
				t.Fatalf("GetLoadStats() error = %v", err)
			}
		}
		if calls != 2 {
			t.Fatalf("repository calls = %d, want 2 with cache disabled", calls)
		}
	})

	t.Run("errors are not cached", func(t *testing.T) {
		wantErr := errors.New("stats failed")
		var calls int
		repo := fakeLoadRepository{getStats: func(context.Context, bson.M) (LoadStats, error) {
			calls++
			if calls == 1 {
				return LoadStats{}, wantErr
			}
			return want, nil
		}}
		service := NewService(repo, time.Minute)
		if _, err := service.GetLoadStats(context.Background(), "chicago"); !errors.Is(err, wantErr) {
			t.Fatalf("first error = %v, want %v", err, wantErr)
		}
		if _, err := service.GetLoadStats(context.Background(), "chicago"); err != nil {
			t.Fatalf("second GetLoadStats() error = %v", err)
		}
		if calls != 2 {
			t.Fatalf("repository calls = %d, want 2 after uncached error", calls)
		}
	})

	t.Run("concurrent cold requests share refresh", func(t *testing.T) {
		var calls atomic.Int64
		repo := fakeLoadRepository{getStats: func(context.Context, bson.M) (LoadStats, error) {
			calls.Add(1)
			time.Sleep(10 * time.Millisecond)
			return want, nil
		}}
		service := NewService(repo, time.Minute)

		var waitGroup sync.WaitGroup
		for range 10 {
			waitGroup.Add(1)
			go func() {
				defer waitGroup.Done()
				if _, err := service.GetLoadStats(context.Background(), "chicago"); err != nil {
					t.Errorf("GetLoadStats() error = %v", err)
				}
			}()
		}
		waitGroup.Wait()
		if calls.Load() != 1 {
			t.Fatalf("repository calls = %d, want 1", calls.Load())
		}
	})

	t.Run("separate keys and LRU eviction", func(t *testing.T) {
		var calls int
		repo := fakeLoadRepository{getStats: func(context.Context, bson.M) (LoadStats, error) {
			calls++
			return want, nil
		}}
		service := NewService(repo, time.Minute)
		for i := range statsCacheMaxEntries {
			if _, err := service.GetLoadStats(context.Background(), fmt.Sprintf("term-%d", i)); err != nil {
				t.Fatalf("GetLoadStats() error = %v", err)
			}
		}
		if _, err := service.GetLoadStats(context.Background(), "term-0"); err != nil {
			t.Fatalf("GetLoadStats() recent hit error = %v", err)
		}
		if _, err := service.GetLoadStats(context.Background(), "term-50"); err != nil {
			t.Fatalf("GetLoadStats() overflow error = %v", err)
		}
		if _, err := service.GetLoadStats(context.Background(), "term-0"); err != nil {
			t.Fatalf("GetLoadStats() retained entry error = %v", err)
		}
		if _, err := service.GetLoadStats(context.Background(), "term-1"); err != nil {
			t.Fatalf("GetLoadStats() evicted entry error = %v", err)
		}
		if calls != statsCacheMaxEntries+2 {
			t.Fatalf("repository calls = %d, want %d", calls, statsCacheMaxEntries+2)
		}
	})
}
