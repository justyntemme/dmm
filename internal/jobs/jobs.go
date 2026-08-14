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

type RetentionPolicy struct {
	MaxTerminal     int
	MaxRecentFailed int
	TerminalMaxAge  time.Duration
	FailedMaxAge    time.Duration
}

var DefaultRetentionPolicy = RetentionPolicy{
	MaxTerminal:     500,
	MaxRecentFailed: 100,
	TerminalMaxAge:  30 * 24 * time.Hour,
	FailedMaxAge:    90 * 24 * time.Hour,
}

type Manager struct {
	mu        sync.Mutex
	seq       int
	jobs      map[string]Job
	onSave    func(Job)
	onDelete  func(Job)
	onPrune   func(Job)
	retention RetentionPolicy
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

func (m *Manager) SetRetention(policy RetentionPolicy, onPrune func(Job)) []Job {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.retention = normalizeRetentionPolicy(policy)
	m.onPrune = onPrune
	return m.pruneLocked(time.Now().UTC())
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
	m.pruneLocked(time.Now().UTC())
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
	m.pruneLocked(time.Now().UTC())
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
	m.pruneLocked(time.Now().UTC())
	return job, true
}

func (m *Manager) Complete(id, message string) (Job, bool) {
	return m.update(id, StatusCompleted, message)
}

func (m *Manager) Run(id, message string) (Job, bool) {
	return m.update(id, StatusRunning, message)
}

func (m *Manager) RunWithPayload(id, message string, payload JobPayload) (Job, bool) {
	return m.updateWithPayload(id, StatusRunning, message, payload)
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

func (m *Manager) TransitionIf(id string, allowed []Status, next Status, message string) (Job, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	job, ok := m.jobs[id]
	if !ok {
		return Job{}, false
	}
	matches := len(allowed) == 0
	for _, status := range allowed {
		if job.Status == status {
			matches = true
			break
		}
	}
	if !matches {
		return job, false
	}
	job.Status = next
	job.Message = message
	job.UpdatedAt = time.Now().UTC()
	m.jobs[id] = job
	m.persistLocked(job)
	m.pruneLocked(time.Now().UTC())
	return job, true
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
	m.pruneLocked(time.Now().UTC())
	return job, true
}

func (m *Manager) updateWithPayload(id string, status Status, message string, payload JobPayload) (Job, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	job, ok := m.jobs[id]
	if !ok {
		return Job{}, false
	}
	job.Status = status
	job.Message = message
	job.Payload = clonePayload(payload)
	job.UpdatedAt = time.Now().UTC()
	m.jobs[id] = job
	m.persistLocked(job)
	m.pruneLocked(time.Now().UTC())
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

func (m *Manager) ListPage(offset, limit int) ([]Job, int) {
	list := m.List()
	if offset < 0 {
		offset = 0
	}
	if limit <= 0 {
		limit = 100
	}
	if limit > 500 {
		limit = 500
	}
	if offset >= len(list) {
		return []Job{}, len(list)
	}
	end := min(offset+limit, len(list))
	return list[offset:end], len(list)
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

func (m *Manager) pruneLocked(now time.Time) []Job {
	policy := normalizeRetentionPolicy(m.retention)
	if policy.MaxTerminal == 0 {
		return nil
	}
	terminal := make([]Job, 0, len(m.jobs))
	for _, job := range m.jobs {
		if isTerminal(job.Status) {
			terminal = append(terminal, job)
		}
	}
	sort.Slice(terminal, func(i, j int) bool {
		return terminal[i].UpdatedAt.After(terminal[j].UpdatedAt)
	})
	keep := make(map[string]struct{}, policy.MaxTerminal+policy.MaxRecentFailed)
	terminalKept := 0
	for _, job := range terminal {
		maxAge := policy.TerminalMaxAge
		if job.Status == StatusFailed {
			maxAge = policy.FailedMaxAge
		}
		if maxAge > 0 && now.Sub(job.UpdatedAt) > maxAge {
			continue
		}
		if terminalKept < policy.MaxTerminal {
			keep[job.ID] = struct{}{}
			terminalKept++
		}
	}
	failedKept := 0
	for _, job := range terminal {
		if job.Status != StatusFailed || failedKept >= policy.MaxRecentFailed {
			continue
		}
		if policy.FailedMaxAge > 0 && now.Sub(job.UpdatedAt) > policy.FailedMaxAge {
			continue
		}
		keep[job.ID] = struct{}{}
		failedKept++
	}
	removed := make([]Job, 0)
	for _, job := range terminal {
		if _, ok := keep[job.ID]; ok {
			continue
		}
		delete(m.jobs, job.ID)
		removed = append(removed, job)
		if m.onPrune != nil {
			m.onPrune(job)
		}
	}
	return removed
}

func normalizeRetentionPolicy(policy RetentionPolicy) RetentionPolicy {
	if policy.MaxTerminal < 0 {
		policy.MaxTerminal = 0
	}
	if policy.MaxRecentFailed < 0 {
		policy.MaxRecentFailed = 0
	}
	return policy
}

func isTerminal(status Status) bool {
	return status == StatusCompleted || status == StatusFailed || status == StatusCanceled
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
