package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"

	"github.com/sangay/web303/gateway/internal/clients"
	"github.com/sangay/web303/gateway/internal/config"
	"github.com/sangay/web303/gateway/internal/handlers"
)

func main() {
	cfg := config.Load()
	log.Printf("API Gateway starting on HTTP port %s", cfg.HTTPPort)

	// --- downstream gRPC clients ---
	cls, err := clients.New(cfg.MembersAddr, cfg.CheckInsAddr)
	if err != nil {
		log.Fatalf("failed to create gRPC clients: %v", err)
	}
	defer cls.Close()

	// --- router ---
	r := mux.NewRouter()
	handlers.NewMembersHandler(cls).Register(r)
	handlers.NewCheckInsHandler(cls).Register(r)

	// health
	r.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}).Methods(http.MethodGet)

	// --- server with sensible timeouts ---
	srv := &http.Server{
		Addr:              ":" + cfg.HTTPPort,
		Handler:           logRequest(r),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	// --- graceful shutdown ---
	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
		<-sig
		log.Println("shutting down HTTP server...")
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
	}()

	log.Printf("listening on :%s", cfg.HTTPPort)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("http server error: %v", err)
	}
}

// logRequest is a tiny request logger for the gateway.
func logRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
	})
}
