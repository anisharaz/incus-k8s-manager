package jobs

import (
	"os/exec"
	"strings"
	"time"

	"github.com/anisharaz/incus-k8s-manager/be/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Manager runs and tracks long-running background jobs.
// Job state is persisted to the database (source of truth).
type Manager struct {
	db *gorm.DB
}

// NewManager creates a new job manager backed by the given database.
func NewManager(db *gorm.DB) *Manager {
	return &Manager{db: db}
}

// CreateClusterJob creates a cluster creation job and runs it in the background.
// It inserts a queued job row, spawns a goroutine, and returns the persisted job.
func (m *Manager) CreateClusterJob(clusterID, clusterName string) (*models.Job, error) {
	now := time.Now().UTC()
	job := &models.Job{
		ID:        uuid.NewString(),
		Type:      "cluster_creation",
		Name:      "Cluster Creation: " + clusterName,
		Status:    models.JobStatusQueued,
		Progress:  0,
		Stage:     "queued",
		Message:   "Cluster creation job accepted",
		Metadata:  map[string]string{"clusterId": clusterID},
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := m.db.Create(job).Error; err != nil {
		return nil, err
	}

	go m.runClusterJob(job.ID, clusterID, clusterName)

	return job, nil
}

// List returns all jobs in reverse chronological order.
func (m *Manager) List() ([]models.Job, error) {
	var jobs []models.Job
	if err := m.db.Order("created_at DESC").Find(&jobs).Error; err != nil {
		return nil, err
	}
	return jobs, nil
}

// Get retrieves a job by id.
func (m *Manager) Get(id string) (*models.Job, error) {
	var job models.Job
	if err := m.db.Where("id = ?", id).First(&job).Error; err != nil {
		return nil, err
	}
	return &job, nil
}

// runClusterJob executes the cluster provisioning steps, updating the job row
// and the cluster row in the database at every stage.
//
// The steps below are SAMPLE placeholders (sleep) that simulate a long-running
// provisioning process. Replace each step's command/args with your real
// provisioning commands (e.g. incus launch, incus exec ...) as needed. The
// final step is expected to print details like "IP: 10.0.0.5" / "Name: vm-1"
// which are parsed into the job result and cluster record.
func (m *Manager) runClusterJob(jobID, clusterID, clusterName string) {
	m.updateJob(jobID, func(job *models.Job) {
		job.Status = models.JobStatusRunning
		job.Stage = "initializing"
		job.Progress = 5
		job.Message = "Starting cluster provisioning..."
	})

	m.updateCluster(clusterID, map[string]interface{}{
		"status":  string(models.ClusterStatusCreating),
		"message": "Provisioning cluster",
	})

	// Sample provisioning steps. Replace with real commands later.
	steps := []provisioningStep{
		{stage: "creating", message: "Creating VM...", command: "sleep", args: []string{"3"}},
		{stage: "configuring", message: "Configuring VM...", command: "sleep", args: []string{"3"}},
		{stage: "fetching-details", message: "Fetching VM details...", command: "bash", args: []string{"-c", "sleep 2; echo 'IP: 192.168.1.100'; echo 'Name: " + clusterName + "'"}},
	}

	var outputs []string
	for i, step := range steps {
		// Report the step before running it
		m.updateJob(jobID, func(job *models.Job) {
			job.Stage = step.stage
			job.Progress = 10 + (i+1)*(90/len(steps))
			job.Message = step.message
		})

		cmd := exec.Command(step.command, step.args...)
		output, err := cmd.CombinedOutput()
		outputs = append(outputs, string(output))

		if err != nil {
			m.failClusterJob(jobID, clusterID, outputs, err)
			return
		}
	}

	outputStr := strings.Join(outputs, "\n")

	// Parse "key: value" lines from the combined output (e.g. IP, Name, ...).
	details := extractDetails(outputStr)
	ip := details["ip"]
	if name := details["name"]; name != "" {
		clusterName = name
	}

	completedAt := time.Now().UTC()
	m.updateJob(jobID, func(job *models.Job) {
		job.Status = models.JobStatusSucceeded
		job.Progress = 100
		job.Stage = "complete"
		job.Message = "Cluster created successfully"
		job.Result = map[string]any{
			"name":    clusterName,
			"ip":      ip,
			"details": details,
			"output":  outputStr,
		}
		job.CompletedAt = &completedAt
	})

	m.updateCluster(clusterID, map[string]interface{}{
		"status":  string(models.ClusterStatusReady),
		"ip":      ip,
		"message": "Cluster is ready",
	})
}

// failClusterJob marks the job and cluster as failed.
func (m *Manager) failClusterJob(jobID, clusterID string, outputs []string, runErr error) {
	outputStr := strings.Join(outputs, "\n")
	completedAt := time.Now().UTC()

	m.updateJob(jobID, func(job *models.Job) {
		job.Status = models.JobStatusFailed
		job.Progress = 100
		job.Stage = "failed"
		job.Message = "Cluster provisioning failed"
		job.Error = runErr.Error()
		job.Result = map[string]any{
			"output": outputStr,
			"error":  runErr.Error(),
		}
		job.CompletedAt = &completedAt
	})

	m.updateCluster(clusterID, map[string]interface{}{
		"status":  string(models.ClusterStatusFailed),
		"message": "Cluster provisioning failed: " + runErr.Error(),
	})
}

// updateJob loads a job, applies the mutation, and persists the row.
func (m *Manager) updateJob(id string, mutate func(*models.Job)) {
	var job models.Job
	if err := m.db.Where("id = ?", id).First(&job).Error; err != nil {
		return
	}

	mutate(&job)
	job.UpdatedAt = time.Now().UTC()
	m.db.Save(&job)
}

// updateCluster applies partial updates to a cluster row.
func (m *Manager) updateCluster(clusterID string, updates map[string]interface{}) {
	m.db.Model(&models.Cluster{}).Where("id = ?", clusterID).Updates(updates)
}

// provisioningStep describes a single command run during cluster provisioning.
// Replace the sample sleep commands with your real provisioning commands.
type provisioningStep struct {
	stage   string // job stage reported to the UI
	message string // human-readable progress message
	command string // executable to run
	args    []string
}

// extractDetails parses "Key: Value" lines from command output into a map.
// Keys are lowercased for case-insensitive lookups (e.g. "IP:" -> "ip").
func extractDetails(output string) map[string]string {
	details := make(map[string]string)
	for _, line := range strings.Split(output, "\n") {
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(parts[0]))
		value := strings.TrimSpace(parts[1])
		if key != "" && value != "" {
			details[key] = value
		}
	}
	return details
}
