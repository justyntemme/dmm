package jobs

import (
	"encoding/json"
	"fmt"
	"net/http"
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
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	Title     string    `json:"title"`
	Status    Status    `json:"status"`
	Message   string    `json:"message"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Manager struct {
	mu          sync.Mutex
	seq         int
	jobs        map[string]Job
	subscribers map[chan Job]struct{}
}

func NewManager() *Manager {
	return &Manager{
		jobs:        map[string]Job{},
		subscribers: map[chan Job]struct{}{},
	}
}

func (m *Manager) Create(jobType, title string) Job {
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
	m.jobs[job.ID] = job
	m.publishLocked(job)
	return job
}

func (m *Manager) Complete(id, message string) (Job, bool) {
	return m.update(id, StatusCompleted, message)
}

func (m *Manager) Wait(id, message string) (Job, bool) {
	return m.update(id, StatusWaiting, message)
}

func (m *Manager) Fail(id, message string) (Job, bool) {
	return m.update(id, StatusFailed, message)
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
	m.publishLocked(job)
	return job, true
}

func (m *Manager) List() []Job {
	m.mu.Lock()
	defer m.mu.Unlock()

	out := make([]Job, 0, len(m.jobs))
	for _, job := range m.jobs {
		out = append(out, job)
	}
	return out
}

func (m *Manager) ClearType(jobType string) []Job {
	m.mu.Lock()
	defer m.mu.Unlock()

	removed := []Job{}
	for id, job := range m.jobs {
		if job.Type != jobType {
			continue
		}
		delete(m.jobs, id)
		job.Status = StatusCanceled
		job.Message = "Cleared"
		job.UpdatedAt = time.Now().UTC()
		removed = append(removed, job)
		m.publishLocked(job)
	}
	return removed
}

func (m *Manager) ServeEvents(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ch := make(chan Job, 16)
	m.mu.Lock()
	m.subscribers[ch] = struct{}{}
	for _, job := range m.jobs {
		ch <- job
	}
	m.mu.Unlock()

	defer func() {
		m.mu.Lock()
		delete(m.subscribers, ch)
		m.mu.Unlock()
		close(ch)
	}()

	for {
		select {
		case <-r.Context().Done():
			return
		case job := <-ch:
			b, _ := json.Marshal(job)
			_, _ = fmt.Fprintf(w, "event: job\ndata: %s\n\n", b)
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		}
	}
}

func (m *Manager) publishLocked(job Job) {
	for ch := range m.subscribers {
		select {
		case ch <- job:
		default:
		}
	}
}
