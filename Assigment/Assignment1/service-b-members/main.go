package main

import (
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"

	"github.com/sangay/web303/members/internal/config"
	"github.com/sangay/web303/members/internal/db"
	"github.com/sangay/web303/members/internal/handler"
	membersv1 "github.com/sangay/web303/proto/members/v1"
)

func main() {
	cfg := config.Load()

	log.Printf("Service B (Members) starting on gRPC port %s", cfg.GRPCPort)

	// --- DB connection (retry for up to 30s in case Postgres is still booting) ---
	store, err := connectWithRetry(cfg.DSN(), 30*time.Second)
	if err != nil {
		log.Fatalf("failed to connect to members_db: %v", err)
	}
	defer store.Close()
	log.Println("connected to members_db")

	// --- gRPC server ---
	lis, err := net.Listen("tcp", ":"+cfg.GRPCPort)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	membersv1.RegisterMemberServiceServer(grpcServer, handler.New(store))

	// health check (useful for Docker healthchecks)
	healthServer := health.NewServer()
	healthServer.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
	healthpb.RegisterHealthServer(grpcServer, healthServer)

	// reflection (lets grpcurl/Postman inspect the API)
	reflection.Register(grpcServer)

	// --- graceful shutdown ---
	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
		<-sig
		log.Println("shutting down gRPC server...")
		grpcServer.GracefulStop()
	}()

	log.Printf("listening on :%s", cfg.GRPCPort)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("grpc server error: %v", err)
	}
}

// connectWithRetry keeps trying to connect to Postgres until timeout elapses.
func connectWithRetry(dsn string, timeout time.Duration) (*db.Store, error) {
	deadline := time.Now().Add(timeout)
	var lastErr error
	for time.Now().Before(deadline) {
		store, err := db.New(dsn)
		if err == nil {
			return store, nil
		}
		lastErr = err
		log.Printf("waiting for members_db... (%v)", err)
		time.Sleep(2 * time.Second)
	}
	return nil, lastErr
}
