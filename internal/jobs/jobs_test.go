package jobs

import (
	"testing"
	"time"
)

func TestRetentionPreservesActiveAndRecentFailedJobs(t *testing.T) {
	now := time.Now().UTC()
	seed := []Job{
		{ID: "job-1", Status: StatusRunning, UpdatedAt: now.Add(-365 * 24 * time.Hour)},
		{ID: "job-2", Status: StatusCompleted, UpdatedAt: now.Add(-time.Hour)},
		{ID: "job-3", Status: StatusCompleted, UpdatedAt: now.Add(-2 * time.Hour)},
		{ID: "job-4", Status: StatusCompleted, UpdatedAt: now.Add(-3 * time.Hour)},
		{ID: "job-5", Status: StatusFailed, UpdatedAt: now.Add(-4 * time.Hour)},
		{ID: "job-6", Status: StatusFailed, UpdatedAt: now.Add(-100 * 24 * time.Hour)},
	}
	manager := NewManagerWithSeed(seed, nil, nil)
	var pruned []string
	removed := manager.SetRetention(RetentionPolicy{
		MaxTerminal: 2, MaxRecentFailed: 1, TerminalMaxAge: 30 * 24 * time.Hour, FailedMaxAge: 90 * 24 * time.Hour,
	}, func(job Job) { pruned = append(pruned, job.ID) })
	if len(removed) != 2 || len(pruned) != 2 {
		t.Fatalf("removed = %+v, callbacks = %+v", removed, pruned)
	}
	for _, id := range []string{"job-1", "job-2", "job-3", "job-5"} {
		if _, ok := manager.Get(id); !ok {
			t.Fatalf("expected retained job %s", id)
		}
	}
	for _, id := range []string{"job-4", "job-6"} {
		if _, ok := manager.Get(id); ok {
			t.Fatalf("expected pruned job %s", id)
		}
	}
}

func TestListPageBoundsHistory(t *testing.T) {
	manager := NewManager()
	for i := 0; i < 5; i++ {
		manager.Create("test", "test")
	}
	page, total := manager.ListPage(1, 2)
	if total != 5 || len(page) != 2 {
		t.Fatalf("page = %+v, total = %d", page, total)
	}
	empty, total := manager.ListPage(20, 2)
	if total != 5 || len(empty) != 0 {
		t.Fatalf("empty page = %+v, total = %d", empty, total)
	}
}
