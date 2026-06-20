package handlers

import (
	"encoding/json"
	"strings"

	"github.com/anisharaz/incus-k8s-manager/be/internal/jobs"
	"github.com/anisharaz/incus-k8s-manager/be/internal/models"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ClusterHandlers handles cluster-related endpoints
type ClusterHandlers struct {
	db      *gorm.DB
	manager *jobs.Manager
}

// NewClusterHandlers creates a new cluster handler
func NewClusterHandlers(db *gorm.DB, manager *jobs.Manager) *ClusterHandlers {
	return &ClusterHandlers{
		db:      db,
		manager: manager,
	}
}

// CreateCluster creates a new cluster
func (h *ClusterHandlers) CreateCluster(c fiber.Ctx) error {
	var req models.CreateClusterRequest
	if err := json.Unmarshal(c.Body(), &req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{
			Error:   "invalid request body",
			Message: err.Error(),
			Code:    fiber.StatusBadRequest,
		})
	}

	// Validate cluster name
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{
			Error:   "validation error",
			Message: "cluster name is required",
			Code:    fiber.StatusBadRequest,
		})
	}

	if len(req.Name) < 3 {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{
			Error:   "validation error",
			Message: "cluster name must be at least 3 characters",
			Code:    fiber.StatusBadRequest,
		})
	}

	// Check if cluster with this name already exists
	var count int64
	h.db.Model(&models.Cluster{}).Where("name = ?", req.Name).Count(&count)
	if count > 0 {
		return c.Status(fiber.StatusConflict).JSON(models.ErrorResponse{
			Error:   "cluster already exists",
			Message: "a cluster with this name already exists",
			Code:    fiber.StatusConflict,
		})
	}

	// Create cluster record
	clusterID := uuid.New().String()
	cluster := models.Cluster{
		ID:      clusterID,
		Name:    req.Name,
		Status:  string(models.ClusterStatusCreating),
		Message: "Cluster creation started",
	}

	if err := h.db.Create(&cluster).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{
			Error:   "database error",
			Message: err.Error(),
			Code:    fiber.StatusInternalServerError,
		})
	}

	// Create a background job for cluster creation
	job := h.manager.CreateClusterJob(clusterID, req.Name)

	// Update cluster with job ID
	cluster.JobID = &job.ID
	h.db.Save(&cluster)

	return c.Status(fiber.StatusAccepted).JSON(models.ClusterResponse{Cluster: cluster})
}

// ListClusters returns all clusters
func (h *ClusterHandlers) ListClusters(c fiber.Ctx) error {
	var clusters []models.Cluster
	if err := h.db.Order("created_at DESC").Find(&clusters).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{
			Error:   "database error",
			Message: err.Error(),
			Code:    fiber.StatusInternalServerError,
		})
	}

	return c.JSON(models.ClusterListResponse{Clusters: clusters})
}

// GetCluster returns a single cluster by ID
func (h *ClusterHandlers) GetCluster(c fiber.Ctx) error {
	clusterID := c.Params("id")
	var cluster models.Cluster

	if err := h.db.Where("id = ?", clusterID).First(&cluster).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(fiber.StatusNotFound).JSON(models.ErrorResponse{
				Error:   "not found",
				Message: "cluster not found",
				Code:    fiber.StatusNotFound,
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{
			Error:   "database error",
			Message: err.Error(),
			Code:    fiber.StatusInternalServerError,
		})
	}

	return c.JSON(models.ClusterResponse{Cluster: cluster})
}
