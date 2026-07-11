package database

import (
	"fmt"
	"log"

	"github.com/v-proc/task-manager-echo-api/config"
	"github.com/v-proc/task-manager-echo-api/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// DB is the shared gorm instance used across the application.
var DB *gorm.DB

// Connect opens the PostgreSQL connection and runs auto-migrations. It is
// safe to call once at startup from main.
func Connect(cfg *config.Config) {
	var err error
	DB, err = gorm.Open(postgres.Open(cfg.DSN()), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}

	sqlDB, err := DB.DB()
	if err != nil {
		log.Fatalf("failed to obtain underlying *sql.DB: %v", err)
	}
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)

	if err := DB.AutoMigrate(&models.User{}, &models.Task{}); err != nil {
		log.Fatalf("failed to auto-migrate: %v", err)
	}

	fmt.Println("database connection established and migrations applied")
}
