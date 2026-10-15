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

	checkinsv1 "github.com/sangay/web303/proto/checkins/v1"
	"github.com/sangay/web303/checkins/internal/client"
	"github.com/sangay/web303/checkins/internal/config"
	"github.com/sangay/web303/checkins/internal/db"
	"github.com/sangay/web303/checkins/internal/handler"
)

func main() {
	cfg := config.Load()
	log.Printf("Service A (CheckIns) starting on gRPC port %s", cfg.GRPCPort)

	// --- DB ---
	store, err := connectWithRetry(cfg.DSN(), 30*time.Second)
	if err != nil {
		log.Fatalf("failed to connect to checkins_db: %v", err)
	}
	defer store.Close()
	log.Println("connected to checkins_db")

	// --- Resilient client to Service B (Members) ---
	membersClient, err := client.New(cfg.MembersAddr, client.Config{
		Timeout:      cfg.MembersTimeout,
		MaxAttempts:  cfg.MembersMaxAttempts,
		BaseBackoff:  cfg.MembersBaseBackoff,
		MaxBackoff:   cfg.MembersMaxBackoff,
		CBMinReqs:    cfg.CBMinRequests,
		CBFailurePct: cfg.CBFailureRate,
		CBInterval:   cfg.CBInterval,
		CBTimeout:    cfg.CBTimeout,
	})
	if err != nil {
		log.Fatalf("failed to create members client: %v", err)
	}
	defer membersClient.Close()
	log.Printf("resilient client to members at %s ready", cfg.MembersAddr)

	// --- gRPC server ---
	lis, err := net.Listen("tcp", ":"+cfg.GRPCPort)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	checkinsv1.RegisterCheckInServiceServer(grpcServer, handler.New(store, membersClient))

	healthServer := health.NewServer()
	healthServer.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
	healthpb.RegisterHealthServer(grpcServer, healthServer)

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

func connectWithRetry(dsn string, timeout time.Duration) (*db.Store, error) {
	deadline := time.Now().Add(timeout)
	var lastErr error
	for time.Now().Before(deadline) {
		store, err := db.New(dsn)
		if err == nil {
			return store, nil
		}
		lastErr = err
		log.Printf("waiting for checkins_db... (%v)", err)
		time.Sleep(2 * time.Second)
	}
	return nil, lastErr
}
