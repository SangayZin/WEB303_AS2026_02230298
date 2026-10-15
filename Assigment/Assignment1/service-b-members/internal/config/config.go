package config

import (
	"fmt"
	"os"
)

// Config holds runtime configuration loaded from environment variables.
type Config struct {
	GRPCPort   string
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	DBSSLMode  string
}

// Load reads config from environment variables, applying safe defaults for local dev.
func Load() *Config {
	return &Config{
		GRPCPort:   getEnv("GRPC_PORT", "50051"),
		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBUser:     getEnv("DB_USER", "members_user"),
		DBPassword: getEnv("DB_PASSWORD", "members_pass"),
		DBName:     getEnv("DB_NAME", "members_db"),
		DBSSLMode:  getEnv("DB_SSLMODE", "disable"),
	}
}

// DSN returns the Postgres connection string.
func (c *Config) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.DBHost, c.DBPort, c.DBUser, c.DBPassword, c.DBName, c.DBSSLMode,
	)
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
