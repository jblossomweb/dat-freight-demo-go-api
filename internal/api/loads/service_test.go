package loads

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type fakeLoadRepository struct {
	countAll      func(context.Context) (int64, error)
	countMatching func(context.Context, bson.M) (int64, error)
	find          func(context.Context, bson.M, *SortModelEntry, int, int) ([]Load, error)
	findByID      func(context.Context, string) (Load, error)
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

		rows, total, filtered, err := NewService(repo).ListLoads(context.Background(), req)
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
		_, _, _, err := NewService(repo).ListLoads(context.Background(), QueryRequest{Filters: map[string]FilterModelEntry{
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
		_, _, _, err := NewService(repo).ListLoads(context.Background(), QueryRequest{})
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
		_, _, _, err := NewService(repo).ListLoads(context.Background(), QueryRequest{})
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
		_, _, _, err := NewService(repo).ListLoads(context.Background(), QueryRequest{})
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
		got, err := NewService(repo).GetLoad(context.Background(), row.ID)
		if err != nil || !reflect.DeepEqual(got, row) || receivedID != row.ID {
			t.Fatalf("GetLoad() = %#v, %v, id %q; want %#v, nil, %q", got, err, receivedID, row, row.ID)
		}
	})

	t.Run("propagates repository error", func(t *testing.T) {
		wantErr := errors.New("find by id failed")
		repo := fakeLoadRepository{
			findByID: func(context.Context, string) (Load, error) { return Load{}, wantErr },
		}
		_, err := NewService(repo).GetLoad(context.Background(), row.ID)
		if !errors.Is(err, wantErr) {
			t.Fatalf("error = %v, want %v", err, wantErr)
		}
	})
}
