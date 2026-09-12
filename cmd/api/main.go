// Command api runs the freight demo HTTP server.
//
// @title        dat-freight-demo-api
// @version      1.0
// @description  REST API backing the DAT freight demo SPA (AG Grid frontend).
// @description  Implements the AG Grid server-side row model contract
// @description  (pagination, sort, filter, quicksearch) for freight loads,
// @description  plus a single-load lookup endpoint.
// @host         localhost:8080
// @BasePath     /
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"dat-freight-demo-api/internal/api/health"
	"dat-freight-demo-api/internal/api/loads"
	"dat-freight-demo-api/internal/db"

	_ "dat-freight-demo-api/internal/api/docs"

	httpSwagger "github.com/swaggo/http-swagger/v2"
)

func main() {
	port := getEnv("PORT", "8080")
	mongoURI := getEnv("MONGO_URI", "mongodb://localhost:27017")
	dbName := getEnv("MONGO_DB_NAME", "freight")

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	client, err := db.Connect(ctx, mongoURI)
	if err != nil {
		log.Fatalf("failed to connect to mongo at startup: %v", err)
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := client.Disconnect(shutdownCtx); err != nil {
			log.Printf("error disconnecting mongo client: %v", err)
		}
	}()

	mux := http.NewServeMux()
	health.RegisterRoutes(mux, client)
	loadsRepo := loads.NewRepository(client.Database(dbName).Collection("loads"))
	loadsService := loads.NewService(loadsRepo)
	loads.RegisterRoutes(mux, loadsService)
	mux.Handle("/swagger/", httpSwagger.WrapHandler)

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	go func() {
		log.Printf("listening on :%s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("shutting down...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("error during server shutdown: %v", err)
	}
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}
