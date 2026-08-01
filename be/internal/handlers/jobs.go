package handlers

import (
	"github.com/anisharaz/incus-k8s-manager/be/internal/jobs"
	"github.com/anisharaz/incus-k8s-manager/be/internal/models"
	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

// TaskHandlers handles long-running job endpoints.
type TaskHandlers struct {
	manager *jobs.Manager
}

// NewTaskHandlers creates a new task handler collection.
func NewTaskHandlers(manager *jobs.Manager) *TaskHandlers {
	return &TaskHandlers{manager: manager}
}

// ListJobs returns all known jobs.
func (h *TaskHandlers) ListJobs(c fiber.Ctx) error {
	jobs, err := h.manager.List()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{
			Error:   "database error",
			Message: err.Error(),
			Code:    fiber.StatusInternalServerError,
		})
	}

	return c.JSON(models.JobListResponse{Jobs: jobs})
}

// GetJob returns a single job by id.
func (h *TaskHandlers) GetJob(c fiber.Ctx) error {
	job, err := h.manager.Get(c.Params("id"))
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(fiber.StatusNotFound).JSON(models.ErrorResponse{
				Error:   "job not found",
				Message: "No job exists with the requested id",
				Code:    fiber.StatusNotFound,
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{
			Error:   "database error",
			Message: err.Error(),
			Code:    fiber.StatusInternalServerError,
		})
	}

	return c.JSON(models.JobResponse{Job: *job})
}
