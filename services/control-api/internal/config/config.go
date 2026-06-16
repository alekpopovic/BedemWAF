package config

import (
	"os"
	"time"
)

type Config struct {
	ListenAddr         string
	DatabaseURL        string
	AdminAPIKey        string
	GatewayAPIKey      string
	CORSAllowedOrigins []string
	RequestBodyLimit   int64
	ClickHouseURL      string
	ClickHouseDatabase string
	ClickHouseUsername string
	ClickHousePassword string

	DBPingTimeout time.Duration
}

func Load() Config {
	cfg := Config{
		ListenAddr:         getenv("BEDEMWAF_CONTROL_API_ADDR", ":8081"),
		DatabaseURL:        os.Getenv("BEDEMWAF_DATABASE_URL"),
		AdminAPIKey:        os.Getenv("BEDEMWAF_ADMIN_API_KEY"),
		GatewayAPIKey:      os.Getenv("BEDEMWAF_GATEWAY_API_KEY"),
		CORSAllowedOrigins: splitCSV(getenv("BEDEMWAF_CORS_ALLOWED_ORIGINS", "http://localhost:3000,http://127.0.0.1:3000")),
		RequestBodyLimit:   getenvInt64("BEDEMWAF_REQUEST_BODY_LIMIT_BYTES", 1<<20),
		ClickHouseURL:      getenv("BEDEMWAF_CLICKHOUSE_URL", "http://localhost:8123"),
		ClickHouseDatabase: getenv("BEDEMWAF_CLICKHOUSE_DATABASE", "bedemwaf"),
		ClickHouseUsername: os.Getenv("BEDEMWAF_CLICKHOUSE_USERNAME"),
		ClickHousePassword: os.Getenv("BEDEMWAF_CLICKHOUSE_PASSWORD"),
		DBPingTimeout:      2 * time.Second,
	}
	if cfg.DatabaseURL == "" {
		cfg.DatabaseURL = os.Getenv("DATABASE_URL")
	}
	return cfg
}

func getenv(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}

func getenvInt64(name string, fallback int64) int64 {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}
	var parsed int64
	for _, r := range value {
		if r < '0' || r > '9' {
			return fallback
		}
		parsed = parsed*10 + int64(r-'0')
	}
	if parsed <= 0 {
		return fallback
	}
	return parsed
}

func splitCSV(value string) []string {
	if value == "" {
		return nil
	}
	var out []string
	start := 0
	for i, r := range value {
		if r != ',' {
			continue
		}
		if item := trimSpace(value[start:i]); item != "" {
			out = append(out, item)
		}
		start = i + 1
	}
	if item := trimSpace(value[start:]); item != "" {
		out = append(out, item)
	}
	return out
}

func trimSpace(value string) string {
	start := 0
	end := len(value)
	for start < end && (value[start] == ' ' || value[start] == '\t' || value[start] == '\n' || value[start] == '\r') {
		start++
	}
	for end > start && (value[end-1] == ' ' || value[end-1] == '\t' || value[end-1] == '\n' || value[end-1] == '\r') {
		end--
	}
	return value[start:end]
}
