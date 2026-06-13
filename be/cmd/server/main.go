package main

import (
	"log"

	"github.com/anisharaz/incus-k8s-manager/be/internal/config"
	"github.com/anisharaz/incus-k8s-manager/be/internal/jobs"
	"github.com/anisharaz/incus-k8s-manager/be/internal/models"
	"github.com/anisharaz/incus-k8s-manager/be/internal/routes"
	"github.com/gofiber/fiber/v3"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	// Load configuration
	cfg := config.NewConfig()

	// Initialize database
	db, err := initDatabase(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v\n", err)
	}

	// Create Fiber app
	app := fiber.New(fiber.Config{
		AppName: "Incus K8s Manager API",
	})

	jobManager := jobs.NewManagerWithDB(db)

	// Setup all routes
	routes.SetupRoutes(app, jobManager, db)

	// Start server
	log.Printf("Starting server on :%s\n", cfg.Port)
	if err := app.Listen(":" + cfg.Port); err != nil {
		log.Fatalf("Failed to start server: %v\n", err)
	}
}

// initDatabase initializes the PostgreSQL database connection and runs migrations
func initDatabase(cfg *config.Config) (*gorm.DB, error) {
	dsn := cfg.GetDatabaseDSN()
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	log.Println("Connected to database successfully")

	// Auto-migrate models
	err = db.AutoMigrate(&models.Cluster{})
	if err != nil {
		return nil, err
	}

	log.Println("Database migration completed")

	return db, nil
}
