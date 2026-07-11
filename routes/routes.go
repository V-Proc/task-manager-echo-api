package routes

import (
	"github.com/v-proc/task-manager-echo-api/config"
	"github.com/v-proc/task-manager-echo-api/controllers"
	"github.com/v-proc/task-manager-echo-api/middleware"

	"github.com/labstack/echo/v4"
)

// Register wires every route onto the provided Echo instance. Public auth
// endpoints are grouped separately from the protected task API.
func Register(e *echo.Echo, cfg *config.Config) {
	e.GET("/health", func(c echo.Context) error {
		return c.JSON(200, echo.Map{"status": "ok"})
	})

	api := e.Group("/api")

	// Public auth routes.
	auth := api.Group("/auth")
	auth.POST("/register", controllers.Register(cfg))
	auth.POST("/login", controllers.Login(cfg))

	// Protected routes — require a valid JWT.
	protected := api.Group("", middleware.RequireAuth(cfg))
	protected.GET("/me", controllers.Me)

	tasks := protected.Group("/tasks")
	tasks.POST("", controllers.Create)
	tasks.GET("", controllers.List)
	tasks.GET("/:id", controllers.Get)
	tasks.PUT("/:id", controllers.Update)
	tasks.DELETE("/:id", controllers.Delete)
}
