package main

import (
	"log"

	"github.com/anisharaz/incus-k8s-manager/be/internal/config"
	"github.com/anisharaz/incus-k8s-manager/be/internal/routes"
	"github.com/gofiber/fiber/v3"
)

func main() {
	// Load configuration
	cfg := config.NewConfig()

	// Create Fiber app
	app := fiber.New(fiber.Config{
		AppName: "Incus K8s Manager API",
	})

	// Setup all routes
	routes.SetupRoutes(app)

	// Start server
	log.Printf("Starting server on :%s\n", cfg.Port)
	if err := app.Listen(":" + cfg.Port); err != nil {
		log.Fatalf("Failed to start server: %v\n", err)
	}
}
