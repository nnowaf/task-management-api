package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

// Config holds all runtime configuration sourced from environment variables.
type Config struct {
	AppPort        string
	AppEnv         string
	LogLevel       string
	DBHost         string
	DBPort         string
	DBUser         string
	DBPassword     string
	DBName         string
	DBSSLMode      string
	JWTSecret      string
	JWTExpiry      time.Duration
	IdempotencyTTL time.Duration
}

// Load reads configuration from the environment, optionally seeding from a .env file.
func Load() *Config {
	_ = godotenv.Load()

	return &Config{
		AppPort:        getEnv("APP_PORT", "3000"),
		AppEnv:         getEnv("APP_ENV", "development"),
		LogLevel:       getEnv("LOG_LEVEL", "info"),
		DBHost:         getEnv("DB_HOST", "localhost"),
		DBPort:         getEnv("DB_PORT", "5432"),
		DBUser:         getEnv("DB_USER", "postgres"),
		DBPassword:     getEnv("DB_PASSWORD", "postgres"),
		DBName:         getEnv("DB_NAME", "gdcpay_tasks"),
		DBSSLMode:      getEnv("DB_SSLMODE", "disable"),
		JWTSecret:      getEnv("JWT_SECRET", "jawa-adalah-kunci"),
		JWTExpiry:      getEnvDuration("JWT_EXPIRY", 24*time.Hour),
		IdempotencyTTL: getEnvDuration("IDEMPOTENCY_TTL", 24*time.Hour),
	}
}

// IsProduction reports whether the app runs in production mode (hides internal error detail).
func (c *Config) IsProduction() bool { return c.AppEnv == "production" }

// DSN builds the libpq-style PostgreSQL connection string consumed by pgx.
func (c *Config) DSN() string {
	return fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=%s",
		c.DBHost, c.DBUser, c.DBPassword, c.DBName, c.DBPort, c.DBSSLMode,
	)
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return fallback
	}
	// Accept either a Go duration ("24h") or a plain integer interpreted as hours.
	if d, err := time.ParseDuration(v); err == nil {
		return d
	}
	if n, err := strconv.Atoi(v); err == nil {
		return time.Duration(n) * time.Hour
	}
	return fallback
}
