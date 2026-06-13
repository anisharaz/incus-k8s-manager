package handlers

import (
	"os/exec"

	"github.com/anisharaz/incus-k8s-manager/be/internal/models"
	"github.com/gofiber/fiber/v3"
)

// HealthHandler handles the health check endpoint
func HealthHandler(c fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"status":  "ok",
		"message": "Server is running",
	})
}

// StatusHandler handles the status endpoint
func StatusHandler(c fiber.Ctx) error {
	incusStatus := getIncusStatus()
	response := models.StatusResponse{
		Status: map[string]string{
			"incus": incusStatus,
		},
	}
	return c.JSON(response)
}

// getIncusStatus checks the status of Incus service
func getIncusStatus() string {
	cmd := exec.Command("incus", "version")
	err := cmd.Run()
	if err != nil {
		return "not found"
	}

	cmd = exec.Command("incus", "list")
	err = cmd.Run()
	if err != nil {
		return "stopped"
	}

	return "running"
}
