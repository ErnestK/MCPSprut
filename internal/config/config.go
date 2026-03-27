package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	DBPath         string
	BatchSize      int
	BufferSize     int
	FlushInterval  time.Duration
	ConnectTimeout time.Duration
	RetryInterval  time.Duration
	MetricsAddr    string
}

func Load() Config {
	return Config{
		DBPath:         envString("SPRUT_DB_PATH", "sprut.db"),
		BatchSize:      envInt("SPRUT_BATCH_SIZE", 100),
		BufferSize:     envInt("SPRUT_BUFFER_SIZE", 256),
		FlushInterval:  envDuration("SPRUT_FLUSH_INTERVAL", 5*time.Second),
		ConnectTimeout: envDuration("SPRUT_CONNECT_TIMEOUT", 30*time.Second),
		RetryInterval:  envDuration("SPRUT_RETRY_INTERVAL", 10*time.Second),
		MetricsAddr:    envString("SPRUT_METRICS_ADDR", ":9100"),
	}
}

func envString(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

func envDuration(key string, fallback time.Duration) time.Duration {
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
