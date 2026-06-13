package main

import (
	"log"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
)

func main() {
	app := fiber.New()

	v1 := app.Group("/api")
	v1.Use(cors.New(cors.Config{
		AllowOrigins: []string{"http://localhost:5173", "http://localhost:8000"},
		AllowHeaders: []string{"Origin", "Content-Type", "Accept"},
	}))

	v1.Get("/", func(c fiber.Ctx) error {
		return c.SendString("Hello, me")
	})

	log.Fatal(app.Listen(":8000"))
}
