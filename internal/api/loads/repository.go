package loads

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// ErrNotFound indicates no load matched the given id.
var ErrNotFound = errors.New("load not found")

// Repository is the only place in this package that talks to MongoDB directly.
type Repository struct {
	collection *mongo.Collection
}

// NewRepository returns a Repository backed by the given loads collection.
func NewRepository(collection *mongo.Collection) *Repository {
	return &Repository{collection: collection}
}

// CountAll returns the full collection size, ignoring any filter.
func (r *Repository) CountAll(ctx context.Context) (int64, error) {
	return r.collection.EstimatedDocumentCount(ctx)
}

// CountMatching returns the exact number of documents matching filter.
func (r *Repository) CountMatching(ctx context.Context, filter bson.M) (int64, error) {
	return r.collection.CountDocuments(ctx, filter)
}

// Find returns the page of loads matching filter, sorted/skipped/limited per the given params.
func (r *Repository) Find(ctx context.Context, filter bson.M, sortBy *SortModelEntry, skip, limit int) ([]Load, error) {
	findOpts := options.Find().SetSkip(int64(skip)).SetLimit(int64(limit))
	if sortBy != nil {
		direction := 1
		if sortBy.Sort == "desc" {
			direction = -1
		}
		findOpts.SetSort(bson.D{{Key: sortBy.ColID, Value: direction}})
	}

	cursor, err := r.collection.Find(ctx, filter, findOpts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	rows := []Load{}
	if err := cursor.All(ctx, &rows); err != nil {
		return nil, err
	}
	return rows, nil
}

// FindByID returns the load with the given id, or ErrNotFound if none matches.
func (r *Repository) FindByID(ctx context.Context, id string) (Load, error) {
	var load Load
	err := r.collection.FindOne(ctx, bson.M{"id": id}).Decode(&load)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return Load{}, ErrNotFound
	}
	return load, err
}
