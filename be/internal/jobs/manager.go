package jobs

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/anisharaz/incus-k8s-manager/be/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const defaultDemoDurationSeconds = 10

// Manager stores and runs in-memory jobs.
type Manager struct {
	mu   sync.RWMutex
	jobs map[string]*models.Job
	db   *gorm.DB
}

// NewManager creates a new job manager.
func NewManager() *Manager {
	return &Manager{jobs: make(map[string]*models.Job)}
}

// NewManagerWithDB creates a new job manager with database connection
func NewManagerWithDB(db *gorm.DB) *Manager {
	return &Manager{
		jobs: make(map[string]*models.Job),
		db:   db,
	}
}

// SetDatabase sets the database connection for the manager
func (m *Manager) SetDatabase(db *gorm.DB) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.db = db
}

// CreateDemoJob creates a demo long-running job and starts it in the background.
func (m *Manager) CreateDemoJob(request models.DemoJobRequest) models.Job {
	durationSeconds := request.DurationSeconds
	if durationSeconds <= 0 {
		durationSeconds = defaultDemoDurationSeconds
	}

	now := time.Now().UTC()
	job := &models.Job{
		ID:              uuid.NewString(),
		Type:            "demo",
		Name:            strings.TrimSpace(request.Name),
		Status:          models.JobStatusQueued,
		Progress:        0,
		Stage:           "queued",
		Message:         "Job accepted and waiting to start",
		DurationSeconds: durationSeconds,
		Metadata:        request.Metadata,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	m.mu.Lock()
	m.jobs[job.ID] = job
	m.mu.Unlock()

	go m.runDemoJob(job.ID)

	return cloneJob(job)
}

// CreateClusterJob creates a cluster creation job and runs init.sh in the background
func (m *Manager) CreateClusterJob(clusterID, clusterName string) models.Job {
	now := time.Now().UTC()
	job := &models.Job{
		ID:       uuid.NewString(),
		Type:     "cluster_creation",
		Name:     "Cluster Creation: " + clusterName,
		Status:   models.JobStatusQueued,
		Progress: 0,
		Stage:    "queued",
		Message:  "Cluster creation job accepted",
		Metadata: map[string]string{
			"clusterId": clusterID,
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	m.mu.Lock()
	m.jobs[job.ID] = job
	m.mu.Unlock()

	go m.runClusterJob(job.ID, clusterID, clusterName)

	return cloneJob(job)
}

// List returns all jobs in reverse chronological order.
func (m *Manager) List() []models.Job {
	m.mu.RLock()
	defer m.mu.RUnlock()

	jobs := make([]models.Job, 0, len(m.jobs))
	for _, job := range m.jobs {
		jobs = append(jobs, cloneJob(job))
	}

	sort.Slice(jobs, func(i, j int) bool {
		return jobs[i].CreatedAt.After(jobs[j].CreatedAt)
	})

	return jobs
}

// Get retrieves a job by id.
func (m *Manager) Get(id string) (models.Job, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	job, ok := m.jobs[id]
	if !ok {
		return models.Job{}, false
	}

	return cloneJob(job), true
}

func (m *Manager) runDemoJob(id string) {
	job, ok := m.Get(id)
	if !ok {
		return
	}

	durationSeconds := job.DurationSeconds
	if durationSeconds <= 0 {
		durationSeconds = defaultDemoDurationSeconds
	}

	_ = m.update(id, func(job *models.Job) {
		job.Status = models.JobStatusRunning
		job.Stage = "preparing"
		job.Message = "Starting background work"
		job.UpdatedAt = time.Now().UTC()
	})

	steps := 10
	stepDuration := time.Duration(durationSeconds) * time.Second / time.Duration(steps)
	if stepDuration < 250*time.Millisecond {
		stepDuration = 250 * time.Millisecond
	}

	for step := 1; step <= steps; step++ {
		time.Sleep(stepDuration)

		if !m.update(id, func(job *models.Job) {
			job.Stage = "processing"
			job.Progress = step * 100 / steps
			job.Message = fmt.Sprintf("Processing step %d of %d", step, steps)
			job.UpdatedAt = time.Now().UTC()
		}) {
			return
		}
	}

	completedAt := time.Now().UTC()
	_ = m.update(id, func(job *models.Job) {
		job.Status = models.JobStatusSucceeded
		job.Progress = 100
		job.Stage = "complete"
		job.Message = "Job finished successfully"
		job.Result = map[string]any{
			"summary":         "Demo job completed",
			"durationSeconds": durationSeconds,
			"steps":           steps,
		}
		job.CompletedAt = &completedAt
		job.UpdatedAt = completedAt
	})
}

func (m *Manager) runClusterJob(jobID, clusterID, clusterName string) {
	_ = m.update(jobID, func(job *models.Job) {
		job.Status = models.JobStatusRunning
		job.Stage = "initializing"
		job.Progress = 5
		job.Message = "Starting cluster initialization..."
		job.UpdatedAt = time.Now().UTC()
	})

	// Update cluster status in database
	if m.db != nil {
		m.db.Model(&models.Cluster{}).Where("id = ?", clusterID).Updates(map[string]interface{}{
			"status":  string(models.ClusterStatusCreating),
			"message": "Initializing cluster creation",
		})
	}

	// Find init.sh script
	initScriptPath := "./init.sh"
	if _, err := os.Stat(initScriptPath); err != nil {
		// Try relative to the binary location
		ex, _ := os.Executable()
		initScriptPath = filepath.Join(filepath.Dir(ex), "..", "..", "init.sh")
	}

	// Execute init.sh with cluster name and VM name parameters
	cmd := exec.Command("bash", initScriptPath, "--name", clusterName)

	_ = m.update(jobID, func(job *models.Job) {
		job.Progress = 10
		job.Stage = "executing"
		job.Message = fmt.Sprintf("Executing: %s", strings.Join(cmd.Args, " "))
		job.UpdatedAt = time.Now().UTC()
	})

	// Run the command
	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	if err != nil {
		completedAt := time.Now().UTC()
		_ = m.update(jobID, func(job *models.Job) {
			job.Status = models.JobStatusFailed
			job.Progress = 100
			job.Stage = "failed"
			job.Message = "Cluster creation failed"
			job.Error = err.Error()
			job.Result = map[string]any{
				"output": outputStr,
				"error":  err.Error(),
			}
			job.CompletedAt = &completedAt
			job.UpdatedAt = completedAt
		})

		// Update cluster status to failed
		if m.db != nil {
			m.db.Model(&models.Cluster{}).Where("id = ?", clusterID).Updates(map[string]interface{}{
				"status":  string(models.ClusterStatusFailed),
				"message": "Cluster creation failed: " + err.Error(),
			})
		}
		return
	}

	// Extract IP from output if available
	ip := extractIP(outputStr)

	completedAt := time.Now().UTC()
	_ = m.update(jobID, func(job *models.Job) {
		job.Status = models.JobStatusSucceeded
		job.Progress = 100
		job.Stage = "complete"
		job.Message = "Cluster created successfully"
		job.Result = map[string]any{
			"output": outputStr,
			"ip":     ip,
		}
		job.CompletedAt = &completedAt
		job.UpdatedAt = completedAt
	})

	// Update cluster status to ready
	if m.db != nil {
		m.db.Model(&models.Cluster{}).Where("id = ?", clusterID).Updates(map[string]interface{}{
			"status":  string(models.ClusterStatusReady),
			"ip":      ip,
			"message": "Cluster is ready",
		})
	}
}

// extractIP extracts the master IP from init.sh output
func extractIP(output string) string {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "Master IP:") {
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				return parts[2]
			}
		}
	}
	return ""
}

func (m *Manager) update(id string, mutate func(*models.Job)) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	job, ok := m.jobs[id]
	if !ok {
		return false
	}

	mutate(job)
	return true
}

func cloneJob(job *models.Job) models.Job {
	if job == nil {
		return models.Job{}
	}

	clone := *job
	if job.Metadata != nil {
		clone.Metadata = make(map[string]string, len(job.Metadata))
		for key, value := range job.Metadata {
			clone.Metadata[key] = value
		}
	}
	if job.Result != nil {
		clone.Result = make(map[string]any, len(job.Result))
		for key, value := range job.Result {
			clone.Result[key] = value
		}
	}

	return clone
}
