// Command api runs the freight demo HTTP server.
//
// @title        dat-freight-demo-go-api
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
	"os/signal"
	"syscall"
	"time"

	"dat-freight-demo-go-api/internal/db"
)

func main() {
	cfg, err := loadConfig()
	if err != nil {
		log.Fatalf("invalid configuration: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	client, err := db.Connect(ctx, cfg.mongoURI)
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

	srv := newServer(client, cfg.dbName, cfg.statsCacheTTL, cfg.allowedOrigins)
	srv.Addr = ":" + cfg.port

	go func() {
		log.Printf("listening on :%s", cfg.port)
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
