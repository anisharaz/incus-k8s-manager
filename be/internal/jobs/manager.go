package jobs

import (
	"time"

	"github.com/anisharaz/incus-k8s-manager/be/internal/incus"
	"github.com/anisharaz/incus-k8s-manager/be/internal/models"
	"gorm.io/gorm"
)

// Manager runs and tracks long-running background jobs.
// Job state is persisted to the database (source of truth).
type Manager struct {
	db    *gorm.DB
	incus *incus.Client
}

// NewManager creates a new job manager backed by the given database.
func NewManager(db *gorm.DB, incusClient *incus.Client) *Manager {
	return &Manager{db: db, incus: incusClient}
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
