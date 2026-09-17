package main

import (
	"context"
	"net/http"
	"time"

	_ "dat-freight-demo-go-api/internal/api/docs"
	"dat-freight-demo-go-api/internal/api/health"
	"dat-freight-demo-go-api/internal/api/loads"

	httpSwagger "github.com/swaggo/http-swagger/v2"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type mongoPinger struct {
	client *mongo.Client
}

func (p mongoPinger) Ping(ctx context.Context) error {
	return p.client.Ping(ctx, nil)
}

func newServer(client *mongo.Client, dbName string, statsCacheTTL time.Duration, allowedOrigins []string) *http.Server {
	mux := http.NewServeMux()

	health.RegisterRoutes(mux, mongoPinger{client: client})

	loadsRepo := loads.NewRepository(client.Database(dbName).Collection("loads"))
	loadsService := loads.NewService(loadsRepo, statsCacheTTL)
	loads.RegisterRoutes(mux, loadsService)

	mux.Handle("/docs/", httpSwagger.WrapHandler)

	// Send the bare root to the docs UI, preserving the "/api" prefix in the redirect if the request arrived with one.
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		location := "/docs/index.html"
		if hadAPIPrefix(r) {
			location = "/api" + location
		}
		http.Redirect(w, r, location, http.StatusFound)
	})

	return &http.Server{
		Handler: corsMiddleware(allowedOrigins, stripAPIPrefix(mux)),
	}
}
