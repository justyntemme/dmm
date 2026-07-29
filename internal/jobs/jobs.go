package jobs

import (
	"fmt"
	"sort"
	"sync"
	"time"
)

type Status string

const (
	StatusQueued    Status = "queued"
	StatusRunning   Status = "running"
	StatusWaiting   Status = "waiting"
	StatusCompleted Status = "completed"
	StatusFailed    Status = "failed"
	StatusCanceled  Status = "canceled"
)

type Job struct {
	ID        string     `json:"id"`
	Type      string     `json:"type"`
	Title     string     `json:"title"`
	Status    Status     `json:"status"`
	Message   string     `json:"message"`
	Payload   JobPayload `json:"payload,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

type JobPayload map[string]string

type Manager struct {
	mu       sync.Mutex
	seq      int
	jobs     map[string]Job
	onSave   func(Job)
	onDelete func(Job)
}

func NewManager() *Manager {
	return NewManagerWithSeed(nil, nil, nil)
}

func NewManagerWithSeed(seed []Job, onSave func(Job), onDelete func(Job)) *Manager {
	manager := &Manager{
		jobs:     map[string]Job{},
		onSave:   onSave,
		onDelete: onDelete,
	}
	for _, job := range seed {
		manager.jobs[job.ID] = job
		var idNum int
		if n, err := fmt.Sscanf(job.ID, "job-%d", &idNum); n == 1 && err == nil && idNum > manager.seq {
			manager.seq = idNum
		}
	}
	return manager
}

func (m *Manager) SetCallbacks(onSave func(Job), onDelete func(Job)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.onSave = onSave
	m.onDelete = onDelete
}

func (m *Manager) Snapshot(job Job) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.jobs[job.ID] = job
}

func (m *Manager) CreateWithID(job Job) Job {
	m.mu.Lock()
	defer m.mu.Unlock()
	if job.CreatedAt.IsZero() {
		job.CreatedAt = time.Now().UTC()
	}
	if job.UpdatedAt.IsZero() {
		job.UpdatedAt = job.CreatedAt
	}
	m.jobs[job.ID] = job
	m.persistLocked(job)
	return job
}

func (m *Manager) Create(jobType, title string) Job {
	return m.CreateWithPayload(jobType, title, nil)
}

func (m *Manager) CreateWithPayload(jobType, title string, payload JobPayload) Job {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.seq++
	now := time.Now().UTC()
	job := Job{
		ID:        fmt.Sprintf("job-%d", m.seq),
		Type:      jobType,
		Title:     title,
		Status:    StatusQueued,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if len(payload) > 0 {
		job.Payload = clonePayload(payload)
	}
	m.jobs[job.ID] = job
	m.persistLocked(job)
	return job
}

func (m *Manager) SetPayload(id string, payload JobPayload) (Job, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	job, ok := m.jobs[id]
	if !ok {
		return Job{}, false
	}
	job.Payload = clonePayload(payload)
	job.UpdatedAt = time.Now().UTC()
	m.jobs[id] = job
	m.persistLocked(job)
	return job, true
}

func (m *Manager) Complete(id, message string) (Job, bool) {
	return m.update(id, StatusCompleted, message)
}

func (m *Manager) Run(id, message string) (Job, bool) {
	return m.update(id, StatusRunning, message)
}

func (m *Manager) Wait(id, message string) (Job, bool) {
	return m.update(id, StatusWaiting, message)
}

func (m *Manager) Fail(id, message string) (Job, bool) {
	return m.update(id, StatusFailed, message)
}

func (m *Manager) Cancel(id, message string) (Job, bool) {
	return m.update(id, StatusCanceled, message)
}

func (m *Manager) update(id string, status Status, message string) (Job, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	job, ok := m.jobs[id]
	if !ok {
		return Job{}, false
	}
	job.Status = status
	job.Message = message
	job.UpdatedAt = time.Now().UTC()
	m.jobs[id] = job
	m.persistLocked(job)
	return job, true
}

func (m *Manager) List() []Job {
	m.mu.Lock()
	defer m.mu.Unlock()

	out := make([]Job, 0, len(m.jobs))
	for _, job := range m.jobs {
		out = append(out, job)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].UpdatedAt.After(out[j].UpdatedAt)
	})
	return out
}

func (m *Manager) Get(id string) (Job, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	job, ok := m.jobs[id]
	return job, ok
}

func (m *Manager) ClearType(jobType string) []Job {
	return m.ClearTypeWhere(jobType, nil)
}

func (m *Manager) ClearTypeWhere(jobType string, shouldClear func(Job) bool) []Job {
	m.mu.Lock()
	defer m.mu.Unlock()

	removed := []Job{}
	for id, job := range m.jobs {
		if job.Type != jobType {
			continue
		}
		if shouldClear != nil && !shouldClear(job) {
			continue
		}
		delete(m.jobs, id)
		job.Status = StatusCanceled
		job.Message = "Cleared"
		job.UpdatedAt = time.Now().UTC()
		removed = append(removed, job)
		m.deleteLocked(job)
	}
	return removed
}

func (m *Manager) persistLocked(job Job) {
	if m.onSave != nil {
		m.onSave(job)
	}
}

func (m *Manager) deleteLocked(job Job) {
	if m.onDelete != nil {
		m.onDelete(job)
	}
}

func clonePayload(payload JobPayload) JobPayload {
	if len(payload) == 0 {
		return nil
	}
	out := make(JobPayload, len(payload))
	for key, value := range payload {
		out[key] = value
	}
	return out
}
