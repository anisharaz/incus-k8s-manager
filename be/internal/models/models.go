package models

import "time"

// StatusResponse represents the API status response
type StatusResponse struct {
	Status map[string]string `json:"status"`
}

// User owns cluster networks, clusters, and nodes. There is no
// authentication yet — this only tracks resource ownership.
type User struct {
	ID        string    `gorm:"primaryKey" json:"id"`
	Username  string    `gorm:"uniqueIndex" json:"username"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// CreateUserRequest represents the request to create a user.
type CreateUserRequest struct {
	Username string `json:"username"`
}

// UserResponse wraps a single user.
type UserResponse struct {
	User User `json:"user"`
}

// UserListResponse wraps a list of users.
type UserListResponse struct {
	Users []User `json:"users"`
}

// ClusterNetworkStatus represents the state of a cluster network.
type ClusterNetworkStatus string

const (
	ClusterNetworkStatusCreating ClusterNetworkStatus = "creating"
	ClusterNetworkStatusReady    ClusterNetworkStatus = "ready"
	ClusterNetworkStatusFailed   ClusterNetworkStatus = "failed"
)

// ClusterNetwork represents an Incus bridge network that cluster VMs are
// launched onto. Name is a user-facing display name, unique per owner;
// IncusName is a system-generated name satisfying Incus's bridge interface
// naming rules (<=15 chars) and is globally unique, since it lives in
// Incus's single namespace regardless of owner.
type ClusterNetwork struct {
	ID        string    `gorm:"primaryKey" json:"id"`
	OwnerID   string    `gorm:"column:owner_id;index" json:"ownerId"`
	Name      string    `json:"name"`
	IncusName string    `gorm:"column:incus_name;uniqueIndex" json:"incusName"`
	CIDR      string    `gorm:"column:cidr" json:"cidr"`
	Gateway   string    `json:"gateway"`
	Status    string    `gorm:"type:varchar(20)" json:"status"`
	Message   string    `json:"message,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// CreateClusterNetworkRequest represents the request to create a cluster network.
type CreateClusterNetworkRequest struct {
	OwnerID string `json:"ownerId"`
	Name    string `json:"name"`
	CIDR    string `json:"cidr"`
}

// ClusterNetworkResponse wraps a single cluster network.
type ClusterNetworkResponse struct {
	Network ClusterNetwork `json:"network"`
}

// ClusterNetworkListResponse wraps a list of cluster networks.
type ClusterNetworkListResponse struct {
	Networks []ClusterNetwork `json:"networks"`
}

// ClusterStatus represents the state of a cluster.
type ClusterStatus string

const (
	ClusterStatusCreating ClusterStatus = "creating"
	ClusterStatusReady    ClusterStatus = "ready"
	ClusterStatusFailed   ClusterStatus = "failed"
	ClusterStatusDeleting ClusterStatus = "deleting"
)

// Cluster represents a Kubernetes cluster: a named group of nodes (one
// master, zero or more workers) launched onto a single cluster network.
// Name is a display name, unique per owner; a cluster isn't itself an Incus
// resource, so it has no separate Incus-facing name (its nodes do).
type Cluster struct {
	ID        string    `gorm:"primaryKey" json:"id"`
	OwnerID   string    `gorm:"column:owner_id;index" json:"ownerId"`
	NetworkID string    `gorm:"column:network_id;index" json:"networkId"`
	Name      string    `json:"name"`
	Status    string    `gorm:"type:varchar(20)" json:"status"`
	Message   string    `json:"message,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// CreateClusterRequest represents the request to create a cluster. Creating
// a cluster launches its master node.
type CreateClusterRequest struct {
	OwnerID   string `json:"ownerId"`
	NetworkID string `json:"networkId"`
	Name      string `json:"name"`
}

// ClusterResponse wraps a single cluster.
type ClusterResponse struct {
	Cluster Cluster `json:"cluster"`
}

// ClusterListResponse wraps a list of clusters.
type ClusterListResponse struct {
	Clusters []Cluster `json:"clusters"`
}

// NodeRole distinguishes a cluster's control-plane node from its workers.
type NodeRole string

const (
	NodeRoleMaster NodeRole = "master"
	NodeRoleWorker NodeRole = "worker"
)

// NodeStatus represents the state of a node's underlying Incus VM.
type NodeStatus string

const (
	NodeStatusCreating NodeStatus = "creating"
	NodeStatusRunning  NodeStatus = "running"
	NodeStatusStopped  NodeStatus = "stopped"
	NodeStatusFailed   NodeStatus = "failed"
	NodeStatusDeleting NodeStatus = "deleting"
)

// Node represents a single Incus VM backing a cluster's master or a worker.
// Name is a display name, unique within its cluster (e.g. "master",
// "worker-1"); IncusName is a system-generated name satisfying Incus's
// instance hostname rules (<=63 chars) and is globally unique, since it
// lives in Incus's single namespace regardless of cluster/owner.
type Node struct {
	ID        string    `gorm:"primaryKey" json:"id"`
	ClusterID string    `gorm:"column:cluster_id;index" json:"clusterId"`
	JobID     *string   `gorm:"column:job_id" json:"jobId,omitempty"`
	Name      string    `json:"name"`
	IncusName string    `gorm:"column:incus_name;uniqueIndex" json:"incusName"`
	Role      string    `gorm:"type:varchar(10)" json:"role"`
	Status    string    `gorm:"type:varchar(20)" json:"status"`
	IP        string    `json:"ip,omitempty"`
	Message   string    `json:"message,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// CreateNodeRequest represents the request to add a worker node to a cluster.
type CreateNodeRequest struct {
	ClusterID string `json:"clusterId"`
}

// NodeResponse wraps a single node.
type NodeResponse struct {
	Node Node `json:"node"`
}

// NodeListResponse wraps a list of nodes.
type NodeListResponse struct {
	Nodes []Node `json:"nodes"`
}

// JobStatus represents the lifecycle state of a long-running job.
type JobStatus string

const (
	JobStatusQueued    JobStatus = "queued"
	JobStatusRunning   JobStatus = "running"
	JobStatusSucceeded JobStatus = "succeeded"
	JobStatusFailed    JobStatus = "failed"
)

// Job represents a long-running background task. It is persisted to the jobs table.
type Job struct {
	ID          string            `gorm:"primaryKey" json:"id"`
	Type        string            `gorm:"index" json:"type"`
	Name        string            `json:"name,omitempty"`
	Status      JobStatus         `gorm:"type:varchar(20);index" json:"status"`
	Progress    int               `json:"progress"`
	Stage       string            `json:"stage,omitempty"`
	Message     string            `json:"message,omitempty"`
	Result      map[string]any    `gorm:"type:jsonb" json:"result,omitempty"`
	Error       string            `json:"error,omitempty"`
	Metadata    map[string]string `gorm:"type:jsonb" json:"metadata,omitempty"`
	CreatedAt   time.Time         `json:"createdAt"`
	UpdatedAt   time.Time         `json:"updatedAt"`
	CompletedAt *time.Time        `json:"completedAt,omitempty"`
}

// JobListResponse wraps a list of jobs.
type JobListResponse struct {
	Jobs []Job `json:"jobs"`
}

// JobResponse wraps a single job.
type JobResponse struct {
	Job Job `json:"job"`
}

// IncusContainer represents an Incus container
type IncusContainer struct {
	Name      string `json:"name"`
	Status    string `json:"status"`
	IPv4      string `json:"ipv4,omitempty"`
	IPv6      string `json:"ipv6,omitempty"`
	Type      string `json:"type"`
	Ephemeral bool   `json:"ephemeral"`
}

// IncusListResponse represents the response from Incus list command
type IncusListResponse struct {
	Containers []IncusContainer `json:"containers"`
	Status     string           `json:"status"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Code    int    `json:"code"`
}
