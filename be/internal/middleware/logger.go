package middleware

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v3"
)

// LoggerMiddleware logs all incoming requests
func LoggerMiddleware() fiber.Handler {
	return func(c fiber.Ctx) error {
		start := time.Now()
		path := c.Path()
		method := c.Method()

		err := c.Next()

		latency := time.Since(start)
		statusCode := c.Response().StatusCode()

		fmt.Printf("[%s] %s %s - %d (%v)\n", time.Now().Format("2006-01-02 15:04:05"), method, path, statusCode, latency)

		return err
	}
}
