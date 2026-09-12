package loads

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// LoadRepository is the persistence boundary used by Service.
//
// The concrete Repository satisfies this interface; tests can provide a fake
// without connecting to MongoDB.
type LoadRepository interface {
	CountAll(context.Context) (int64, error)
	CountMatching(context.Context, bson.M) (int64, error)
	Find(context.Context, bson.M, *SortModelEntry, int, int) ([]Load, error)
	FindByID(context.Context, string) (Load, error)
}

// LoadService is the application boundary used by HTTP handlers.
//
// The concrete Service satisfies this interface; tests can provide a small
// fake without connecting to MongoDB.
type LoadService interface {
	ListLoads(context.Context, QueryRequest) ([]Load, int64, int64, error)
	GetLoad(context.Context, string) (Load, error)
}

// Service contains the business logic for listing and looking up loads,
// independent of HTTP transport.
type Service struct {
	repo LoadRepository
}

// NewService returns a Service backed by the given repository.
func NewService(repo LoadRepository) *Service {
	return &Service{repo: repo}
}

// InvalidQueryError marks an error as caused by the caller's request (bad
// input), as opposed to an infrastructure failure. Use errors.As to detect it.
type InvalidQueryError struct{ err error }

func (e *InvalidQueryError) Error() string { return e.err.Error() }
func (e *InvalidQueryError) Unwrap() error  { return e.err }

// ListLoads builds the Mongo filter for req and returns the matching page of
// loads alongside the total collection size and the filtered match count.
// A returned InvalidQueryError means req itself was invalid (bad request);
// any other error means the query failed for infrastructure reasons.
func (s *Service) ListLoads(ctx context.Context, req QueryRequest) (rows []Load, total, filtered int64, err error) {
	filter, err := BuildMongoFilter(req)
	if err != nil {
		return nil, 0, 0, &InvalidQueryError{err}
	}

	total, err = s.repo.CountAll(ctx)
	if err != nil {
		return nil, 0, 0, err
	}

	filtered, err = s.repo.CountMatching(ctx, filter)
	if err != nil {
		return nil, 0, 0, err
	}

	rows, err = s.repo.Find(ctx, filter, req.Sort, req.StartRow, req.EndRow-req.StartRow)
	if err != nil {
		return nil, 0, 0, err
	}

	return rows, total, filtered, nil
}

// GetLoad returns the load with the given id, or ErrNotFound if none matches.
func (s *Service) GetLoad(ctx context.Context, id string) (Load, error) {
	return s.repo.FindByID(ctx, id)
}
