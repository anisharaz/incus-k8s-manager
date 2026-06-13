package routes

import (
	"github.com/anisharaz/incus-k8s-manager/be/internal/handlers"
	"github.com/anisharaz/incus-k8s-manager/be/internal/middleware"
	"github.com/gofiber/fiber/v3"
)

// SetupRoutes initializes all application routes
func SetupRoutes(app *fiber.App) {
	// Apply global middleware
	app.Use(middleware.LoggerMiddleware())
	app.Use(middleware.CORSMiddleware())

	// Health check route
	app.Get("/health", handlers.HealthHandler)

	// API v1 routes
	v1 := app.Group("/api/v1")

	// Status routes
	v1.Get("/status", handlers.StatusHandler)

	// Root API endpoint
	v1.Get("/", func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"message": "Incus K8s Manager API v1",
			"version": "1.0.0",
		})
	})

	// 404 handler
	app.Use(func(c fiber.Ctx) error {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Route not found",
			"path":  c.Path(),
		})
	})
}
