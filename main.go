package main

import (
	"fmt"
	"log"

	"github.com/v-proc/task-manager-echo-api/config"
	"github.com/v-proc/task-manager-echo-api/database"
	"github.com/v-proc/task-manager-echo-api/routes"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func main() {
	cfg := config.Load()

	database.Connect(cfg)

	e := echo.New()

	// Global middleware.
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())

	routes.Register(e, cfg)

	addr := ":" + cfg.ServerPort
	fmt.Printf("starting server on %s\n", addr)
	if err := e.Start(addr); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
