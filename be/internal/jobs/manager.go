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

// List returns ownerID's jobs in reverse chronological order.
func (m *Manager) List(ownerID string) ([]models.Job, error) {
	var jobs []models.Job
	if err := m.db.Where("owner_id = ?", ownerID).Order("created_at DESC").Find(&jobs).Error; err != nil {
		return nil, err
	}
	return jobs, nil
}

// Get retrieves a job by id, scoped to ownerID — a job owned by someone
// else looks identical to a nonexistent one (gorm.ErrRecordNotFound).
func (m *Manager) Get(id, ownerID string) (*models.Job, error) {
	var job models.Job
	if err := m.db.Where("id = ? AND owner_id = ?", id, ownerID).First(&job).Error; err != nil {
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
