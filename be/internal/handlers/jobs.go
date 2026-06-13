package handlers

import (
	"encoding/json"
	"strings"

	"github.com/anisharaz/incus-k8s-manager/be/internal/jobs"
	"github.com/anisharaz/incus-k8s-manager/be/internal/models"
	"github.com/gofiber/fiber/v3"
)

// TaskHandlers handles long-running job endpoints.
type TaskHandlers struct {
	manager *jobs.Manager
}

// NewTaskHandlers creates a new task handler collection.
func NewTaskHandlers(manager *jobs.Manager) *TaskHandlers {
	return &TaskHandlers{manager: manager}
}

// CreateDemoJob starts a sample long-running job.
func (h *TaskHandlers) CreateDemoJob(c fiber.Ctx) error {
	var request models.DemoJobRequest
	if len(c.Body()) > 0 {
		if err := json.Unmarshal(c.Body(), &request); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{
				Error:   "invalid request body",
				Message: err.Error(),
				Code:    fiber.StatusBadRequest,
			})
		}
	}

	request.Name = strings.TrimSpace(request.Name)
	job := h.manager.CreateDemoJob(request)

	return c.Status(fiber.StatusAccepted).JSON(models.JobResponse{Job: job})
}

// ListJobs returns all known jobs.
func (h *TaskHandlers) ListJobs(c fiber.Ctx) error {
	return c.JSON(models.JobListResponse{Jobs: h.manager.List()})
}

// GetJob returns a single job by id.
func (h *TaskHandlers) GetJob(c fiber.Ctx) error {
	job, ok := h.manager.Get(c.Params("id"))
	if !ok {
		return c.Status(fiber.StatusNotFound).JSON(models.ErrorResponse{
			Error:   "job not found",
			Message: "No job exists with the requested id",
			Code:    fiber.StatusNotFound,
		})
	}

	return c.JSON(models.JobResponse{Job: job})
}
