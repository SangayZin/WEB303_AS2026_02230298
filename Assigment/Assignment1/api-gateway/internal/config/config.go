package config

import "os"

type Config struct {
	HTTPPort     string
	MembersAddr  string
	CheckInsAddr string
}

func Load() *Config {
	return &Config{
		HTTPPort:     getEnv("HTTP_PORT", "8080"),
		MembersAddr:  getEnv("MEMBERS_ADDR", "localhost:50051"),
		CheckInsAddr: getEnv("CHECKINS_ADDR", "localhost:50052"),
	}
}

func getEnv(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
