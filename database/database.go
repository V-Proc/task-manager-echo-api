package database

import (
	"fmt"
	"log"
	"time"

	"github.com/v-proc/task-manager-echo-api/config"
	"github.com/v-proc/task-manager-echo-api/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// DB is the shared gorm instance used across the application.
var DB *gorm.DB

// Connect opens the PostgreSQL connection and runs auto-migrations. It
// retries for up to ~60s (with backoff) before giving up, so the app tolerates a
// database that isn't ready the instant the process starts — common under
// docker/podman-compose where the API container may come up before Postgres
// finishes initializing. Safe to call once at startup from main.
func Connect(cfg *config.Config) {
	const (
		maxAttempts = 30
		baseDelay   = 1 * time.Second
	)

	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		db, err := gorm.Open(postgres.Open(cfg.DSN()), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Warn),
		})
		if err == nil {
			// A successful gorm.Open doesn't guarantee the server is actually
			// accepting queries, so ping before trusting the connection.
			sqlDB, perr := db.DB()
			if perr == nil {
				if perr = sqlDB.Ping(); perr == nil {
					sqlDB.SetMaxIdleConns(10)
					sqlDB.SetMaxOpenConns(100)
					if err := db.AutoMigrate(&models.User{}, &models.Task{}); err != nil {
						log.Fatalf("failed to auto-migrate: %v", err)
					}
					DB = db
					fmt.Println("database connection established and migrations applied")
					return
				}
				lastErr = perr
				sqlDB.Close()
			} else {
				lastErr = perr
			}
		} else {
			lastErr = err
		}

		delay := baseDelay * time.Duration(attempt)
		if delay > 5*time.Second {
			delay = 5 * time.Second
		}
		log.Printf("waiting for database (attempt %d/%d): %v; retrying in %s",
			attempt, maxAttempts, lastErr, delay)
		time.Sleep(delay)
	}

	log.Fatalf("failed to connect database after %d attempts: %v", maxAttempts, lastErr)
}
