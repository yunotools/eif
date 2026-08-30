package config

import (
	"fmt"
	"net/url"
	"time"
)

type Config struct {
	ServerConfig  ServerConfig
	LoggerConfig  LoggerConfig
	CORSConfig    CORSConfig
	HDDTGDTConfig HDDTGDTConfig
}

type ServerConfig struct {
	Port              int
	StaticDir         string
	ReadHeaderTimeout time.Duration
	ShutdownTimeout   time.Duration
}

func (c ServerConfig) ServerAddress() string {
	return fmt.Sprintf(":%d", c.Port)
}

type LoggerConfig struct {
	Level  string
	Format string
}

type CORSConfig struct {
	AllowedOrigins []string
}

type HDDTGDTConfig struct {
	Endpoint             string
	Timeout              time.Duration
	MaxQueryDays         int
	MaxExportDays        int
	SessionSkew          time.Duration
	SessionStorePath     string
	SessionEncryptionKey []byte
}

func (c HDDTGDTConfig) Validate() error {
	u, err := url.Parse(c.Endpoint)

	if err != nil || u.Scheme == "" || u.Host == "" {
		return fmt.Errorf("invalid EIF_HDDT_GDT_ENDPOINT")
	}

	if c.MaxQueryDays <= 0 {
		return fmt.Errorf("EIF_HDDT_GDT_MAX_QUERY_DAYS must be > 0")
	}

	if c.MaxExportDays <= 0 {
		return fmt.Errorf("EIF_HDDT_GDT_MAX_EXPORT_DAYS must be > 0")
	}

	if c.SessionStorePath == "" {
		return fmt.Errorf("EIF_SESSION_STORE_PATH is required")
	}

	// AES-256 yêu cầu key đúng 32 bytes.
	if len(c.SessionEncryptionKey) != 32 {
		return fmt.Errorf("EIF_SESSION_ENCRYPTION_KEY must decode to exactly 32 bytes")
	}

	return nil
}
