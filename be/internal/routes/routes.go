package routes

import (
	"github.com/anisharaz/incus-k8s-manager/be/internal/handlers"
	"github.com/anisharaz/incus-k8s-manager/be/internal/incus"
	"github.com/anisharaz/incus-k8s-manager/be/internal/jobs"
	"github.com/anisharaz/incus-k8s-manager/be/internal/middleware"
	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

// SetupRoutes initializes all application routes
func SetupRoutes(app *fiber.App, jobManager *jobs.Manager, db *gorm.DB, incusClient *incus.Client) {
	taskHandlers := handlers.NewTaskHandlers(jobManager)
	networkHandlers := handlers.NewNetworkHandlers(db, incusClient)
	userHandlers := handlers.NewUserHandlers(db)
	clusterHandlers := handlers.NewClusterHandlers(db, jobManager)
	nodeHandlers := handlers.NewNodeHandlers(db)

	// Apply global middleware
	app.Use(middleware.LoggerMiddleware())
	app.Use(middleware.CORSMiddleware())

	// Health check route
	app.Get("/health", handlers.HealthHandler)

	// API v1 routes
	v1 := app.Group("/api/v1")

	// Status routes
	v1.Get("/status", handlers.StatusHandler)
	v1.Get("/jobs", taskHandlers.ListJobs)
	v1.Get("/jobs/:id", taskHandlers.GetJob)

	// User routes
	v1.Post("/users", userHandlers.CreateUser)
	v1.Get("/users", userHandlers.ListUsers)
	v1.Get("/users/:id", userHandlers.GetUser)

	// Cluster network routes
	v1.Post("/networks", networkHandlers.CreateNetwork)
	v1.Get("/networks", networkHandlers.ListNetworks)
	v1.Get("/networks/:id", networkHandlers.GetNetwork)
	v1.Delete("/networks/:id", networkHandlers.DeleteNetwork)

	// Cluster routes
	v1.Post("/clusters", clusterHandlers.CreateCluster)
	v1.Get("/clusters", clusterHandlers.ListClusters)
	v1.Get("/clusters/:id", clusterHandlers.GetCluster)
	v1.Get("/clusters/:id/nodes", nodeHandlers.ListNodesForCluster)

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
