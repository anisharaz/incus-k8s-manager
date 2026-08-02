package handlers

import (
	"encoding/json"
	"fmt"

	"github.com/anisharaz/incus-k8s-manager/be/internal/jobs"
	"github.com/anisharaz/incus-k8s-manager/be/internal/middleware"
	"github.com/anisharaz/incus-k8s-manager/be/internal/models"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// NodeHandlers handles node endpoints.
type NodeHandlers struct {
	db      *gorm.DB
	manager *jobs.Manager
}

// NewNodeHandlers creates a new node handler.
func NewNodeHandlers(db *gorm.DB, manager *jobs.Manager) *NodeHandlers {
	return &NodeHandlers{db: db, manager: manager}
}

// ListNodesForCluster returns all nodes belonging to a cluster (master
// first, then workers in creation order), so callers can poll VM status
// (and, via jobId, the underlying job's progress) while a node is created.
// The cluster must belong to the authenticated user.
func (h *NodeHandlers) ListNodesForCluster(c fiber.Ctx) error {
	ownerID := middleware.ClaimsFromContext(c).UserID
	clusterID := c.Params("id")

	var cluster models.Cluster
	if err := h.db.Where("id = ? AND owner_id = ?", clusterID, ownerID).First(&cluster).Error; err != nil {
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

	var nodes []models.Node
	if err := h.db.Where("cluster_id = ?", clusterID).Order("created_at ASC").Find(&nodes).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{
			Error:   "database error",
			Message: err.Error(),
			Code:    fiber.StatusInternalServerError,
		})
	}

	return c.JSON(models.NodeListResponse{Nodes: nodes})
}

// CreateNode adds a worker node to a cluster: it launches a VM on the
// cluster's network, fetches a fresh join token from the cluster's master
// (kubeadm token create --print-join-command — not the one-time token
// kubeadm init printed, which may be long expired), and runs kubeadm join
// on the new VM. The request body is optional; cpu/memory/disk each
// default if omitted, and are validated the same way as cluster creation.
// The cluster must belong to the authenticated user.
func (h *NodeHandlers) CreateNode(c fiber.Ctx) error {
	ownerID := middleware.ClaimsFromContext(c).UserID
	clusterID := c.Params("id")

	var cluster models.Cluster
	if err := h.db.Where("id = ? AND owner_id = ?", clusterID, ownerID).First(&cluster).Error; err != nil {
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

	if cluster.Status != string(models.ClusterStatusReady) {
		return c.Status(fiber.StatusConflict).JSON(models.ErrorResponse{
			Error:   "cluster not ready",
			Message: "cluster must be ready before adding workers",
			Code:    fiber.StatusConflict,
		})
	}

	var master models.Node
	if err := h.db.Where("cluster_id = ? AND role = ?", clusterID, string(models.NodeRoleMaster)).First(&master).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{
			Error:   "database error",
			Message: "cluster has no master node: " + err.Error(),
			Code:    fiber.StatusInternalServerError,
		})
	}

	if master.Status != string(models.NodeStatusRunning) {
		return c.Status(fiber.StatusConflict).JSON(models.ErrorResponse{
			Error:   "master not running",
			Message: "master node must be running before adding workers",
			Code:    fiber.StatusConflict,
		})
	}

	var network models.ClusterNetwork
	if err := h.db.Where("id = ?", cluster.NetworkID).First(&network).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{
			Error:   "database error",
			Message: err.Error(),
			Code:    fiber.StatusInternalServerError,
		})
	}

	var req models.CreateNodeRequest
	if body := c.Body(); len(body) > 0 {
		if err := json.Unmarshal(body, &req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{
				Error:   "invalid request body",
				Message: err.Error(),
				Code:    fiber.StatusBadRequest,
			})
		}
	}

	size, err := validateNodeSize(req.CPU, req.Memory, req.Disk)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{
			Error:   "validation error",
			Message: err.Error(),
			Code:    fiber.StatusBadRequest,
		})
	}

	var workerCount int64
	h.db.Model(&models.Node{}).Where("cluster_id = ? AND role = ?", clusterID, string(models.NodeRoleWorker)).Count(&workerCount)

	nodeID := uuid.New().String()
	node := models.Node{
		ID:        nodeID,
		ClusterID: clusterID,
		Name:      fmt.Sprintf("worker-%d", workerCount+1),
		IncusName: generateIncusNodeName(string(models.NodeRoleWorker), nodeID),
		Role:      string(models.NodeRoleWorker),
		Status:    string(models.NodeStatusCreating),
		Message:   "Node creation started",
	}

	if err := h.db.Create(&node).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{
			Error:   "database error",
			Message: err.Error(),
			Code:    fiber.StatusInternalServerError,
		})
	}

	job, err := h.manager.CreateNodeJob(ownerID, node.ID, node.IncusName, network.IncusName, node.Role, master.IncusName, size)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{
			Error:   "job creation error",
			Message: err.Error(),
			Code:    fiber.StatusInternalServerError,
		})
	}
	h.db.Model(&node).Update("job_id", job.ID)

	return c.Status(fiber.StatusAccepted).JSON(models.NodeResponse{Node: node})
}
