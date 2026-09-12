// Command api runs the freight demo HTTP server.
package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"dat-freight-demo-api/internal/db"
	"dat-freight-demo-api/internal/loads"

	"go.mongodb.org/mongo-driver/v2/mongo"
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
	mux.HandleFunc("GET /health", healthHandler(client))
	loadsCollection := client.Database(dbName).Collection("loads")
	mux.HandleFunc("GET /loads", loads.Handler(loadsCollection))
	mux.HandleFunc("GET /load/{id}", loads.GetHandler(loadsCollection))
	mux.HandleFunc("GET /load", loads.GetHandler(loadsCollection))

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

func healthHandler(client *mongo.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		pingCtx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()

		status := "ok"
		code := http.StatusOK
		mongoStatus := "ok"

		if err := client.Ping(pingCtx, nil); err != nil {
			status = "degraded"
			code = http.StatusServiceUnavailable
			mongoStatus = "unreachable"
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(code)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"status": status,
			"mongo":  mongoStatus,
		})
	}
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}
