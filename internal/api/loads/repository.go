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

// GetStats returns full-collection counts for the fixed dashboard categories.
func (r *Repository) GetStats(ctx context.Context) (LoadStats, error) {
	pipeline := mongo.Pipeline{bson.D{{Key: "$group", Value: bson.D{
		{Key: "_id", Value: nil},
		{Key: "numTotal", Value: bson.D{{Key: "$sum", Value: 1}}},
		{Key: "flatbed", Value: conditionalCount("equipmentType", "Flatbed")},
		{Key: "reefer", Value: conditionalCount("equipmentType", "Reefer")},
		{Key: "van", Value: conditionalCount("equipmentType", "Van")},
		{Key: "available", Value: conditionalCount("status", "Available")},
		{Key: "inTransit", Value: conditionalCount("status", "In Transit")},
		{Key: "delivered", Value: conditionalCount("status", "Delivered")},
	}}}}

	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return LoadStats{}, err
	}
	defer cursor.Close(ctx)

	var result struct {
		NumTotal  int64 `bson:"numTotal"`
		Flatbed   int64 `bson:"flatbed"`
		Reefer    int64 `bson:"reefer"`
		Van       int64 `bson:"van"`
		Available int64 `bson:"available"`
		InTransit int64 `bson:"inTransit"`
		Delivered int64 `bson:"delivered"`
	}
	if cursor.Next(ctx) {
		if err := cursor.Decode(&result); err != nil {
			return LoadStats{}, err
		}
	}
	if err := cursor.Err(); err != nil {
		return LoadStats{}, err
	}

	return LoadStats{
		NumTotal: result.NumTotal,
		EquipmentType: []StatCount{
			{Label: "Flatbed", Value: result.Flatbed},
			{Label: "Reefer", Value: result.Reefer},
			{Label: "Van", Value: result.Van},
		},
		Status: []StatCount{
			{Label: "Available", Value: result.Available},
			{Label: "In Transit", Value: result.InTransit},
			{Label: "Delivered", Value: result.Delivered},
		},
	}, nil
}

func conditionalCount(field, value string) bson.D {
	return bson.D{{Key: "$sum", Value: bson.D{{Key: "$cond", Value: bson.A{
		bson.D{{Key: "$eq", Value: bson.A{"$" + field, value}}},
		1,
		0,
	}}}}}
}
