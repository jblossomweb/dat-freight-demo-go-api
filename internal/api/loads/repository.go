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
	} else {
		findOpts.SetSort(bson.D{{Key: "id", Value: 1}}) // Default sort by ID ascending
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

// GetStats returns full and filtered counts for the fixed dashboard categories.
func (r *Repository) GetStats(ctx context.Context, filter bson.M) (LoadStats, error) {
	filteredPipeline := bson.A{}
	if len(filter) > 0 {
		filteredPipeline = append(filteredPipeline, bson.D{{Key: "$match", Value: filter}})
	}
	filteredPipeline = append(filteredPipeline, bson.D{{Key: "$group", Value: bson.D{
		{Key: "_id", Value: nil},
		{Key: "numResults", Value: bson.D{{Key: "$sum", Value: 1}}},
		{Key: "flatbed", Value: conditionalCount("equipmentType", "Flatbed")},
		{Key: "reefer", Value: conditionalCount("equipmentType", "Reefer")},
		{Key: "van", Value: conditionalCount("equipmentType", "Van")},
		{Key: "available", Value: conditionalCount("status", "Available")},
		{Key: "inTransit", Value: conditionalCount("status", "In Transit")},
		{Key: "delivered", Value: conditionalCount("status", "Delivered")},
	}}})

	pipeline := mongo.Pipeline{bson.D{{Key: "$facet", Value: bson.D{
		{Key: "total", Value: bson.A{bson.D{{Key: "$count", Value: "value"}}}},
		{Key: "filtered", Value: filteredPipeline},
	}}}}

	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return LoadStats{}, err
	}
	defer cursor.Close(ctx)

	var result struct {
		Total []struct {
			Value int64 `bson:"value"`
		} `bson:"total"`
		Filtered []struct {
			NumResults int64 `bson:"numResults"`
			Flatbed    int64 `bson:"flatbed"`
			Reefer     int64 `bson:"reefer"`
			Van        int64 `bson:"van"`
			Available  int64 `bson:"available"`
			InTransit  int64 `bson:"inTransit"`
			Delivered  int64 `bson:"delivered"`
		} `bson:"filtered"`
	}
	if cursor.Next(ctx) {
		if err := cursor.Decode(&result); err != nil {
			return LoadStats{}, err
		}
	}
	if err := cursor.Err(); err != nil {
		return LoadStats{}, err
	}

	stats := LoadStats{
		EquipmentType: []StatCount{
			{Label: "Flatbed"},
			{Label: "Reefer"},
			{Label: "Van"},
		},
		Status: []StatCount{
			{Label: "Available"},
			{Label: "In Transit"},
			{Label: "Delivered"},
		},
	}
	if len(result.Total) > 0 {
		stats.NumTotal = result.Total[0].Value
	}
	if len(result.Filtered) > 0 {
		filtered := result.Filtered[0]
		stats.NumResults = filtered.NumResults
		stats.EquipmentType[0].Value = filtered.Flatbed
		stats.EquipmentType[1].Value = filtered.Reefer
		stats.EquipmentType[2].Value = filtered.Van
		stats.Status[0].Value = filtered.Available
		stats.Status[1].Value = filtered.InTransit
		stats.Status[2].Value = filtered.Delivered
	}
	return stats, nil
}

func conditionalCount(field, value string) bson.D {
	return bson.D{{Key: "$sum", Value: bson.D{{Key: "$cond", Value: bson.A{
		bson.D{{Key: "$eq", Value: bson.A{"$" + field, value}}},
		1,
		0,
	}}}}}
}
