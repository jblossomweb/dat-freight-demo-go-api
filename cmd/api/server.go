package main

import (
	"context"
	"net/http"

	_ "dat-freight-demo-api/internal/api/docs"
	"dat-freight-demo-api/internal/api/health"
	"dat-freight-demo-api/internal/api/loads"

	httpSwagger "github.com/swaggo/http-swagger/v2"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type mongoPinger struct {
	client *mongo.Client
}

func (p mongoPinger) Ping(ctx context.Context) error {
	return p.client.Ping(ctx, nil)
}

func newServer(client *mongo.Client, dbName string) *http.Server {
	mux := http.NewServeMux()

	health.RegisterRoutes(mux, mongoPinger{client: client})

	loadsRepo := loads.NewRepository(client.Database(dbName).Collection("loads"))
	loadsService := loads.NewService(loadsRepo)
	loads.RegisterRoutes(mux, loadsService)

	mux.Handle("/docs/", httpSwagger.WrapHandler)

	return &http.Server{
		Handler: mux,
	}
}
