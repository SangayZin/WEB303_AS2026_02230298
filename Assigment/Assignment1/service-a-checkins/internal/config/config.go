package config

import (
	"fmt"
	"os"
	"time"
)

// Config holds runtime config for Service A (CheckIns).
type Config struct {
	GRPCPort string

	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	DBSSLMode  string

	// Service B (Members) gRPC target + resilience settings
	MembersAddr         string
	MembersTimeout      time.Duration
	MembersMaxAttempts  int
	MembersBaseBackoff  time.Duration
	MembersMaxBackoff   time.Duration

	// Circuit breaker thresholds
	CBMaxRequests uint32
	CBInterval    time.Duration
	CBTimeout     time.Duration
	CBMinRequests uint32
	CBFailureRate float64
}

func Load() *Config {
	return &Config{
		GRPCPort: getEnv("GRPC_PORT", "50052"),

		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBUser:     getEnv("DB_USER", "checkins_user"),
		DBPassword: getEnv("DB_PASSWORD", "checkins_pass"),
		DBName:     getEnv("DB_NAME", "checkins_db"),
		DBSSLMode:  getEnv("DB_SSLMODE", "disable"),

		MembersAddr:        getEnv("MEMBERS_ADDR", "localhost:50051"),
		MembersTimeout:     getEnvDuration("MEMBERS_TIMEOUT", 2*time.Second),
		MembersMaxAttempts: getEnvInt("MEMBERS_MAX_ATTEMPTS", 3),
		MembersBaseBackoff: getEnvDuration("MEMBERS_BASE_BACKOFF", 100*time.Millisecond),
		MembersMaxBackoff:  getEnvDuration("MEMBERS_MAX_BACKOFF", 1*time.Second),

		CBMaxRequests: getEnvUint("CB_MAX_REQUESTS", 1),
		CBInterval:    getEnvDuration("CB_INTERVAL", 10*time.Second),
		CBTimeout:     getEnvDuration("CB_TIMEOUT", 15*time.Second),
		CBMinRequests: getEnvUint("CB_MIN_REQUESTS", 3),
		CBFailureRate: getEnvFloat("CB_FAILURE_RATE", 0.6),
	}
}

func (c *Config) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.DBHost, c.DBPort, c.DBUser, c.DBPassword, c.DBName, c.DBSSLMode,
	)
}

// ---- env helpers ----

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	var n int
	if _, err := fmt.Sscanf(v, "%d", &n); err == nil {
		return n
	}
	return fallback
}

func getEnvUint(key string, fallback uint32) uint32 {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	var n uint32
	if _, err := fmt.Sscanf(v, "%d", &n); err == nil {
		return n
	}
	return fallback
}

func getEnvFloat(key string, fallback float64) float64 {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	var n float64
	if _, err := fmt.Sscanf(v, "%f", &n); err == nil {
		return n
	}
	return fallback
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return fallback
	}
	return d
}
