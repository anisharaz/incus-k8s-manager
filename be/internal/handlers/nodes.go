package handlers

import (
	"github.com/anisharaz/incus-k8s-manager/be/internal/models"
	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

// NodeHandlers handles node endpoints.
type NodeHandlers struct {
	db *gorm.DB
}

// NewNodeHandlers creates a new node handler.
func NewNodeHandlers(db *gorm.DB) *NodeHandlers {
	return &NodeHandlers{db: db}
}

// ListNodesForCluster returns all nodes belonging to a cluster (master
// first, then workers in creation order), so callers can poll VM status
// (and, via jobId, the underlying job's progress) while a node is created.
func (h *NodeHandlers) ListNodesForCluster(c fiber.Ctx) error {
	var nodes []models.Node
	if err := h.db.Where("cluster_id = ?", c.Params("id")).Order("created_at ASC").Find(&nodes).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{
			Error:   "database error",
			Message: err.Error(),
			Code:    fiber.StatusInternalServerError,
		})
	}

	return c.JSON(models.NodeListResponse{Nodes: nodes})
}
