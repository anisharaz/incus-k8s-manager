package models

import "time"

// StatusResponse represents the API status response
type StatusResponse struct {
	Status map[string]string `json:"status"`
}

// ClusterNetworkStatus represents the state of a cluster network.
type ClusterNetworkStatus string

const (
	ClusterNetworkStatusCreating ClusterNetworkStatus = "creating"
	ClusterNetworkStatusReady    ClusterNetworkStatus = "ready"
	ClusterNetworkStatusFailed   ClusterNetworkStatus = "failed"
)

// ClusterNetwork represents an Incus bridge network that cluster VMs are
// launched onto. The name doubles as the underlying Incus/Linux bridge
// interface name, so it is constrained to Incus's interface naming rules.
type ClusterNetwork struct {
	ID        string    `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"uniqueIndex" json:"name"`
	CIDR      string    `gorm:"column:cidr" json:"cidr"`
	Gateway   string    `json:"gateway"`
	Status    string    `gorm:"type:varchar(20)" json:"status"`
	Message   string    `json:"message,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// CreateClusterNetworkRequest represents the request to create a cluster network.
type CreateClusterNetworkRequest struct {
	Name string `json:"name"`
	CIDR string `json:"cidr"`
}

// ClusterNetworkResponse wraps a single cluster network.
type ClusterNetworkResponse struct {
	Network ClusterNetwork `json:"network"`
}

// ClusterNetworkListResponse wraps a list of cluster networks.
type ClusterNetworkListResponse struct {
	Networks []ClusterNetwork `json:"networks"`
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
