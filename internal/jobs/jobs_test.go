package jobs

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func TestServeEventsInitialSnapshotLargerThanBufferDoesNotBlockManager(t *testing.T) {
	manager := NewManager()
	for i := 0; i < 20; i++ {
		manager.Create("test", "Test job")
	}

	ctx, cancel := context.WithCancel(context.Background())
	req := httptest.NewRequest(http.MethodGet, "/api/jobs/events", nil).WithContext(ctx)
	writer := newNotifyWriter()
	done := make(chan struct{})
	go func() {
		defer close(done)
		manager.ServeEvents(writer, req)
	}()

	select {
	case <-writer.wrote:
	case <-time.After(250 * time.Millisecond):
		t.Fatal("event stream did not write its initial snapshot")
	}

	listed := make(chan []Job, 1)
	go func() {
		listed <- manager.List()
	}()
	select {
	case jobs := <-listed:
		if len(jobs) != 20 {
			t.Fatalf("jobs = %d, want 20", len(jobs))
		}
	case <-time.After(250 * time.Millisecond):
		t.Fatal("job manager List blocked while event stream was open")
	}

	cancel()
	select {
	case <-done:
	case <-time.After(250 * time.Millisecond):
		t.Fatal("event stream did not stop after request cancellation")
	}
}

type notifyWriter struct {
	header http.Header
	once   sync.Once
	wrote  chan struct{}
}

func newNotifyWriter() *notifyWriter {
	return &notifyWriter{
		header: http.Header{},
		wrote:  make(chan struct{}),
	}
}

func (w *notifyWriter) Header() http.Header {
	return w.header
}

func (w *notifyWriter) WriteHeader(statusCode int) {}

func (w *notifyWriter) Write(p []byte) (int, error) {
	w.once.Do(func() {
		close(w.wrote)
	})
	return len(p), nil
}

func (w *notifyWriter) Flush() {}
