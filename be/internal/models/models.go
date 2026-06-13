package models

import "time"

// StatusResponse represents the API status response
type StatusResponse struct {
	Status map[string]string `json:"status"`
}

// ClusterStatus represents the state of a Kubernetes cluster
type ClusterStatus string

const (
	ClusterStatusCreating ClusterStatus = "creating"
	ClusterStatusReady    ClusterStatus = "ready"
	ClusterStatusFailed   ClusterStatus = "failed"
	ClusterStatusDeleting ClusterStatus = "deleting"
)

// Cluster represents a Kubernetes cluster
type Cluster struct {
	ID        string    `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"index" json:"name"`
	Status    string    `gorm:"type:varchar(50)" json:"status"`
	JobID     *string   `json:"jobId,omitempty"`
	IP        string    `json:"ip,omitempty"`
	Message   string    `json:"message,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// CreateClusterRequest represents the request to create a cluster
type CreateClusterRequest struct {
	Name string `json:"name" validate:"required,min=3,max=255"`
}

// ClusterResponse wraps a single cluster
type ClusterResponse struct {
	Cluster Cluster `json:"cluster"`
}

// ClusterListResponse wraps a list of clusters
type ClusterListResponse struct {
	Clusters []Cluster `json:"clusters"`
}

// JobStatus represents the lifecycle state of a long-running job.
type JobStatus string

const (
	JobStatusQueued    JobStatus = "queued"
	JobStatusRunning   JobStatus = "running"
	JobStatusSucceeded JobStatus = "succeeded"
	JobStatusFailed    JobStatus = "failed"
)

// Job represents a long-running background task.
type Job struct {
	ID              string            `json:"id"`
	Type            string            `json:"type"`
	Name            string            `json:"name,omitempty"`
	Status          JobStatus         `json:"status"`
	Progress        int               `json:"progress"`
	Stage           string            `json:"stage,omitempty"`
	Message         string            `json:"message,omitempty"`
	Result          map[string]any    `json:"result,omitempty"`
	Error           string            `json:"error,omitempty"`
	DurationSeconds int               `json:"durationSeconds,omitempty"`
	Metadata        map[string]string `json:"metadata,omitempty"`
	CreatedAt       time.Time         `json:"createdAt"`
	UpdatedAt       time.Time         `json:"updatedAt"`
	CompletedAt     *time.Time        `json:"completedAt,omitempty"`
}

// DemoJobRequest represents a sample long-running task submission.
type DemoJobRequest struct {
	Name            string            `json:"name"`
	DurationSeconds int               `json:"durationSeconds"`
	Metadata        map[string]string `json:"metadata"`
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
