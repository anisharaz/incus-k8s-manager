package handlers

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/anisharaz/incus-k8s-manager/be/internal/jobs"
	"github.com/anisharaz/incus-k8s-manager/be/internal/models"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/lxc/incus/v7/shared/units"
	"gorm.io/gorm"
)

// Minimum node VM allocation. CPU/Memory match kubeadm's own hard preflight
// requirements (confirmed live: kubeadm rejects <2 CPUs or <1700MB RAM).
// Disk isn't kubeadm-enforced, but 20GiB is the commonly recommended floor
// for a control-plane node (etcd + images) and comfortably exceeds the base
// VM image's own 4GiB virtual size.
const (
	minNodeCPU    = 2
	minNodeMemory = "1700MB"
	minNodeDisk   = "20GiB"
)

// defaultNodeMemory is used when the request omits memory. It's above
// minNodeMemory, not equal to it: virtualization overhead means the guest
// can see slightly less RAM than the configured limit, and kubeadm's check
// is a hard cutoff, so sitting exactly on the minimum risks failing it.
const defaultNodeMemory = "2GiB"

// ClusterHandlers handles cluster endpoints.
type ClusterHandlers struct {
	db      *gorm.DB
	manager *jobs.Manager
}

// NewClusterHandlers creates a new cluster handler.
func NewClusterHandlers(db *gorm.DB, manager *jobs.Manager) *ClusterHandlers {
	return &ClusterHandlers{db: db, manager: manager}
}

// CreateCluster creates a cluster and its master node, then starts a
// background job to launch the master's VM on the chosen network. Only the
// VM is launched here — bootstrapping Kubernetes on it is a later step.
func (h *ClusterHandlers) CreateCluster(c fiber.Ctx) error {
	var req models.CreateClusterRequest
	if err := json.Unmarshal(c.Body(), &req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{
			Error:   "invalid request body",
			Message: err.Error(),
			Code:    fiber.StatusBadRequest,
		})
	}

	req.OwnerID = strings.TrimSpace(req.OwnerID)
	if req.OwnerID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{
			Error:   "validation error",
			Message: "ownerId is required",
			Code:    fiber.StatusBadRequest,
		})
	}

	req.NetworkID = strings.TrimSpace(req.NetworkID)
	if req.NetworkID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{
			Error:   "validation error",
			Message: "networkId is required",
			Code:    fiber.StatusBadRequest,
		})
	}

	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" || len(req.Name) > 63 {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{
			Error:   "validation error",
			Message: "name must be between 1 and 63 characters",
			Code:    fiber.StatusBadRequest,
		})
	}

	size, err := validateNodeSize(req.CPU, req.Memory, req.Disk)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{
			Error:   "validation error",
			Message: err.Error(),
			Code:    fiber.StatusBadRequest,
		})
	}

	var network models.ClusterNetwork
	if err := h.db.Where("id = ?", req.NetworkID).First(&network).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(fiber.StatusNotFound).JSON(models.ErrorResponse{
				Error:   "not found",
				Message: "cluster network not found",
				Code:    fiber.StatusNotFound,
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{
			Error:   "database error",
			Message: err.Error(),
			Code:    fiber.StatusInternalServerError,
		})
	}

	// Fast pre-check for a friendlier duplicate-name error than the DB's own.
	var count int64
	h.db.Model(&models.Cluster{}).Where("owner_id = ? AND name = ?", req.OwnerID, req.Name).Count(&count)
	if count > 0 {
		return c.Status(fiber.StatusConflict).JSON(models.ErrorResponse{
			Error:   "cluster already exists",
			Message: "you already have a cluster with this name",
			Code:    fiber.StatusConflict,
		})
	}

	cluster := models.Cluster{
		ID:        uuid.New().String(),
		OwnerID:   req.OwnerID,
		NetworkID: req.NetworkID,
		Name:      req.Name,
		Status:    string(models.ClusterStatusCreating),
		Message:   "Cluster creation started",
	}

	nodeID := uuid.New().String()
	node := models.Node{
		ID:        nodeID,
		ClusterID: cluster.ID,
		Name:      "master",
		IncusName: generateIncusNodeName(string(models.NodeRoleMaster), nodeID),
		Role:      string(models.NodeRoleMaster),
		Status:    string(models.NodeStatusCreating),
		Message:   "Node creation started",
	}

	if err := h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&cluster).Error; err != nil {
			return err
		}
		return tx.Create(&node).Error
	}); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{
			Error:   "database error",
			Message: err.Error(),
			Code:    fiber.StatusInternalServerError,
		})
	}

	job, err := h.manager.CreateNodeJob(node.ID, node.IncusName, network.IncusName, node.Role, size)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{
			Error:   "job creation error",
			Message: err.Error(),
			Code:    fiber.StatusInternalServerError,
		})
	}
	h.db.Model(&node).Update("job_id", job.ID)

	return c.Status(fiber.StatusAccepted).JSON(models.ClusterResponse{Cluster: cluster})
}

// ListClusters returns all clusters.
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

// GetCluster returns a single cluster by ID.
func (h *ClusterHandlers) GetCluster(c fiber.Ctx) error {
	var cluster models.Cluster
	if err := h.db.Where("id = ?", c.Params("id")).First(&cluster).Error; err != nil {
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

// validateNodeSize applies the minimum to any unset field (cpu == 0 or
// memory/disk == "") and rejects anything explicitly set below it.
func validateNodeSize(cpu int, memory, disk string) (jobs.NodeSize, error) {
	if cpu == 0 {
		cpu = minNodeCPU
	} else if cpu < minNodeCPU {
		return jobs.NodeSize{}, fmt.Errorf("cpu must be at least %d, got %d", minNodeCPU, cpu)
	}

	memory = strings.TrimSpace(memory)
	if memory == "" {
		memory = defaultNodeMemory
	} else if err := checkMinByteSize("memory", memory, minNodeMemory); err != nil {
		return jobs.NodeSize{}, err
	}

	disk = strings.TrimSpace(disk)
	if disk == "" {
		disk = minNodeDisk
	} else if err := checkMinByteSize("disk", disk, minNodeDisk); err != nil {
		return jobs.NodeSize{}, err
	}

	return jobs.NodeSize{CPU: strconv.Itoa(cpu), Memory: memory, Disk: disk}, nil
}

// checkMinByteSize parses value and min using Incus's size format (e.g.
// "2GiB") and errors if value parses below min or doesn't parse at all.
func checkMinByteSize(field, value, min string) error {
	got, err := units.ParseByteSizeString(value)
	if err != nil {
		return fmt.Errorf("%s: %w", field, err)
	}

	minBytes, err := units.ParseByteSizeString(min)
	if err != nil {
		return fmt.Errorf("%s: %w", field, err)
	}

	if got < minBytes {
		return fmt.Errorf("%s must be at least %s, got %s", field, min, value)
	}

	return nil
}

// generateIncusNodeName derives a globally-unique, Incus-safe VM instance
// name (<=63 chars, alphanumeric/hyphen, satisfying validate.IsHostname)
// from a role and resource ID, so the user-facing display name is free of
// Incus's naming rules and per-cluster uniqueness scope.
func generateIncusNodeName(role, id string) string {
	compact := strings.ReplaceAll(id, "-", "")
	if len(compact) > 12 {
		compact = compact[:12]
	}
	return role + "-" + compact
}
