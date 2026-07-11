//go:build !(js && wasm)

package config_test

import (
	"os"
	"strings"
	"testing"

	"github.com/v-proc/task-manager-echo-api/config"
	"github.com/v-proc/task-manager-echo-api/internal/testutil"
)

func TestLoad_ReadsEnvValues(t *testing.T) {
	testutil.SetEnv(t, "SERVER_PORT", "9999")
	testutil.SetEnv(t, "DB_HOST", "db.example.com")
	testutil.SetEnv(t, "DB_PORT", "6543")
	testutil.SetEnv(t, "DB_USER", "u")
	testutil.SetEnv(t, "DB_PASSWORD", "p")
	testutil.SetEnv(t, "DB_NAME", "n")
	testutil.SetEnv(t, "DB_SSLMODE", "require")
	testutil.SetEnv(t, "DB_TIMEZONE", "Europe/Berlin")
	testutil.SetEnv(t, "JWT_SECRET", "topsecret")
	testutil.SetEnv(t, "JWT_EXPIRE_HOURS", "12")

	// godotenv.Load() won't overwrite real env vars, so this exercises the
	// os.LookupEnv paths in config.Load.
	cfg := config.Load()

	if cfg.ServerPort != "9999" {
		t.Errorf("ServerPort = %q, want 9999", cfg.ServerPort)
	}
	if cfg.DBHost != "db.example.com" {
		t.Errorf("DBHost = %q", cfg.DBHost)
	}
	if cfg.JWTSecret != "topsecret" {
		t.Errorf("JWTSecret = %q", cfg.JWTSecret)
	}
	if cfg.JWTExpireHrs != 12 {
		t.Errorf("JWTExpireHrs = %d, want 12", cfg.JWTExpireHrs)
	}
}

func TestLoad_FallsBackOnDefaults(t *testing.T) {
	for _, k := range []string{
		"SERVER_PORT", "DB_HOST", "DB_PORT", "DB_USER", "DB_PASSWORD",
		"DB_NAME", "DB_SSLMODE", "DB_TIMEZONE", "JWT_SECRET", "JWT_EXPIRE_HOURS",
	} {
		os.Unsetenv(k)
	}

	cfg := config.Load()

	if cfg.ServerPort != "8080" {
		t.Errorf("default ServerPort = %q, want 8080", cfg.ServerPort)
	}
	if cfg.JWTExpireHrs != 24 {
		t.Errorf("default JWTExpireHrs = %d, want 24", cfg.JWTExpireHrs)
	}
}

func TestLoad_BadExpireFallsBackToInt(t *testing.T) {
	testutil.SetEnv(t, "JWT_EXPIRE_HOURS", "not-a-number")
	cfg := config.Load()
	if cfg.JWTExpireHrs != 24 {
		t.Errorf("JWTExpireHrs = %d, want fallback 24", cfg.JWTExpireHrs)
	}
}

func TestDSN_ContainsAllFields(t *testing.T) {
	cfg := testutil.Config()
	dsn := cfg.DSN()

	for _, want := range []string{
		"host=localhost", "port=5432", "user=postgres", "password=postgres",
		"dbname=task_manager_test", "sslmode=disable", "TimeZone=UTC",
	} {
		if !strings.Contains(dsn, want) {
			t.Errorf("DSN missing %q; got %q", want, dsn)
		}
	}
}
