package middleware

import (
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
)

// CORSMiddleware configures CORS for the application
func CORSMiddleware() fiber.Handler {
	return cors.New(cors.Config{
		AllowOrigins: []string{
			"http://localhost:5173",
			"http://localhost:8000",
			"http://localhost:3000",
		},
		AllowHeaders: []string{
			"Origin",
			"Content-Type",
			"Accept",
			"Authorization",
		},
		AllowMethods: []string{
			"GET",
			"POST",
			"PUT",
			"DELETE",
			"PATCH",
			"OPTIONS",
		},
		// The session is a cookie (see internal/middleware/auth.go), which
		// browsers only attach to cross-origin requests when the request
		// opts in (fetch credentials: "include") AND the server allows it
		// here. Requires an explicit origin list above — incompatible with
		// AllowOrigins: ["*"].
		AllowCredentials: true,
	})
}
