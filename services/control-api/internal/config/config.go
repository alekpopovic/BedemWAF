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
