package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Config holds all application configuration values loaded from the environment.
type Config struct {
	ServerPort    string
	DBHost        string
	DBPort        string
	DBUser        string
	DBPassword    string
	DBName        string
	DBSSLMode     string
	DBTimezone    string
	JWTSecret     string
	JWTExpireHrs  int
}

// Load reads the .env file (if present) and the surrounding environment, then
// returns a populated Config. Missing variables fall back to sensible defaults
// so the application can still boot in development.
func Load() *Config {
	// Ignore the error: .env is optional in production where env vars are
	// injected directly by the runtime.
	_ = godotenv.Load()

	cfg := &Config{
		ServerPort:   getEnv("SERVER_PORT", "8080"),
		DBHost:       getEnv("DB_HOST", "localhost"),
		DBPort:       getEnv("DB_PORT", "5432"),
		DBUser:       getEnv("DB_USER", "postgres"),
		DBPassword:   getEnv("DB_PASSWORD", "postgres"),
		DBName:       getEnv("DB_NAME", "task_manager"),
		DBSSLMode:    getEnv("DB_SSLMODE", "disable"),
		DBTimezone:   getEnv("DB_TIMEZONE", "UTC"),
		JWTSecret:    getEnv("JWT_SECRET", "change-me-in-production"),
		JWTExpireHrs: getEnvInt("JWT_EXPIRE_HOURS", 24),
	}

	return cfg
}

// DSN builds a PostgreSQL connection string understood by gorm's postgres driver.
func (c *Config) DSN() string {
	return "host=" + c.DBHost +
		" port=" + c.DBPort +
		" user=" + c.DBUser +
		" password=" + c.DBPassword +
		" dbname=" + c.DBName +
		" sslmode=" + c.DBSSLMode +
		" TimeZone=" + c.DBTimezone
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}
