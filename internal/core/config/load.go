package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

func Load() (*Config, error) {
	port, err := envInt("EIF_PORT", 8080)
	if err != nil {
		return nil, err
	}

	maxQueryDays, err := envInt("EIF_HDDT_GDT_MAX_QUERY_DAYS", 30)
	if err != nil {
		return nil, err
	}

	maxExportDays, err := envInt("EIF_HDDT_GDT_MAX_EXPORT_DAYS", 30)
	if err != nil {
		return nil, err
	}

	cfg := &Config{
		ServerConfig: ServerConfig{
			Port:              port,
			StaticDir:         envString("EIF_STATIC_DIR", "./web/static"),
			ReadHeaderTimeout: envDuration("EIF_READ_HEADER_TIMEOUT", 10*time.Second),
			ShutdownTimeout:   envDuration("EIF_SHUTDOWN_TIMEOUT", 10*time.Second),
		},

		LoggerConfig: LoggerConfig{
			Level:  envString("EIF_LOG_LEVEL", "info"),
			Format: envString("EIF_LOG_FORMAT", "json"),
		},

		CORSConfig: CORSConfig{
			AllowedOrigins: envCSV("EIF_CORS_ALLOWED_ORIGINS", []string{"http://localhost:3000"}),
		},

		HDDTGDTConfig: HDDTGDTConfig{
			Endpoint:      strings.TrimRight(envString("EIF_HDDT_GDT_ENDPOINT", "https://hoadondientu.gdt.gov.vn"), "/"),
			Timeout:       envDuration("EIF_HDDT_GDT_TIMEOUT", 60*time.Second),
			MaxQueryDays:  maxQueryDays,
			MaxExportDays: maxExportDays,
			SessionSkew:   envDuration("EIF_SESSION_EXPIRY_SKEW", 30*time.Second),
		},
	}

	if err := cfg.HDDTGDTConfig.Validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func envString(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}

	return fallback
}

func envInt(key string, fallback int) (int, error) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback, nil
	}

	n, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer: %w", key, err)
	}

	return n, nil
}

func envDuration(key string, fallback time.Duration) time.Duration {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	d, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}

	return d
}

func envCSV(key string, fallback []string) []string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if item := strings.TrimSpace(part); item != "" {
			out = append(out, item)
		}
	}

	return out
}
