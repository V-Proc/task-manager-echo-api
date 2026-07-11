//go:build !(js && wasm)

// Package testutil holds shared helpers for the unit tests across the
// task-manager-echo-api modules: an in-memory SQLite database swapped into the
// global database.DB, a deterministic test config, and an env-var helper that
// auto-restores on test cleanup.
package testutil

import (
	"os"
	"testing"

	"github.com/v-proc/task-manager-echo-api/config"
	"github.com/v-proc/task-manager-echo-api/database"
	"github.com/v-proc/task-manager-echo-api/models"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Config returns a config wired with deterministic test values. JWT settings
// are kept short so token-expiry-sensitive tests run fast.
func Config() *config.Config {
	return &config.Config{
		ServerPort:   "8080",
		DBHost:       "localhost",
		DBPort:       "5432",
		DBUser:       "postgres",
		DBPassword:   "postgres",
		DBName:       "task_manager_test",
		DBSSLMode:    "disable",
		DBTimezone:   "UTC",
		JWTSecret:    "test-secret",
		JWTExpireHrs: 1,
	}
}

// SetupDB swaps the global database.DB for an in-memory SQLite instance, runs
// migrations, and truncates any rows left over from previous tests (the shared
// cache keeps the in-memory DB alive across tests in the same process). It
// returns the *gorm.DB so callers can seed rows directly.
//
// glebarez/sqlite is pure-Go (modernc backend), so no CGO toolchain is needed.
func SetupDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared&_pragma=foreign_keys(1)"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.Task{}); err != nil {
		t.Fatalf("auto-migrate: %v", err)
	}
	db.Exec("DELETE FROM tasks")
	db.Exec("DELETE FROM users")
	database.DB = db
	return db
}

// SetEnv sets an env var for the duration of the test and restores the previous
// value (or unsets it) on cleanup.
func SetEnv(t *testing.T, key, value string) {
	t.Helper()
	prev, ok := os.LookupEnv(key)
	os.Setenv(key, value)
	t.Cleanup(func() {
		if ok {
			os.Setenv(key, prev)
		} else {
			os.Unsetenv(key)
		}
	})
}

// SeedUser creates a user row and returns its ID, for use as the authenticated
// user in subsequent request tests. It reuses the same database/models imports
// testutil already needs, so it adds no new dependencies to the package.
func SeedUser(t *testing.T, email string) uint {
	t.Helper()
	u := models.User{Name: email, Email: email, Password: "x"}
	if err := database.DB.Create(&u).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	return u.ID
}
