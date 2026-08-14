package server

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/coder/websocket"
	"github.com/justyntemme/decky-mod-manager/internal/events"
	"github.com/justyntemme/decky-mod-manager/internal/jobs"
)

func (s *Server) pruneDomainEvents(ctx context.Context, force bool) {
	s.retentionMu.Lock()
	if !force && time.Since(s.retentionAt) < retentionCheckInterval {
		s.retentionMu.Unlock()
		return
	}
	s.retentionAt = time.Now()
	s.retentionMu.Unlock()
	protectedAfterID := int64(0)
	if cursor, ok := s.events.MinSubscriberCursor(); ok {
		protectedAfterID = cursor + 1
	}
	removed, err := s.db.PruneDomainEvents(ctx, time.Now().UTC().Add(-domainEventRetentionAge), domainEventRetentionLimit, protectedAfterID)
	if err != nil {
		s.logger.Warn("domain event retention failed", "error", err)
		return
	}
	if removed > 0 {
		s.logger.Info("domain event retention completed", "removed", removed)
	}
}

func (s *Server) handleJobs(w http.ResponseWriter, r *http.Request) {
	s.cleanupOrphanedInstallerChoiceJobs(r.Context(), "jobs-list")
	limit := boundedQueryInt(r.URL.Query().Get("limit"), jobSnapshotLimit, 1, 500)
	offset := boundedQueryInt(r.URL.Query().Get("offset"), 0, 0, 1000000)
	page, total := s.jobs.ListPage(offset, limit)
	w.Header().Set("X-Total-Count", strconv.Itoa(total))
	if offset+len(page) < total {
		w.Header().Set("X-Next-Offset", strconv.Itoa(offset+len(page)))
	}
	writeJSON(w, http.StatusOK, jobAPIResponses(page))
}

func boundedQueryInt(raw string, fallback, minimum, maximum int) int {
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		return fallback
	}
	if value < minimum {
		return minimum
	}
	if value > maximum {
		return maximum
	}
	return value
}

func (s *Server) handleEventsWebSocket(w http.ResponseWriter, r *http.Request) {
	var afterID int64
	if raw := strings.TrimSpace(r.URL.Query().Get("after")); raw != "" {
		parsed, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || parsed < 0 {
			http.Error(w, "after must be a non-negative event id", http.StatusBadRequest)
			return
		}
		afterID = parsed
	}
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{OriginPatterns: []string{
		"steamloopback.host", "*.steamloopback.host", "localhost", "127.0.0.1",
	}})
	if err != nil {
		s.logger.Warn("event websocket accept failed", "remote", r.RemoteAddr, "error", err)
		return
	}
	defer conn.Close(websocket.StatusNormalClosure, "")
	ctx := conn.CloseRead(r.Context())
	s.logger.Info("event websocket opened", "remote", r.RemoteAddr, "after_id", afterID)
	defer s.logger.Info("event websocket closed", "remote", r.RemoteAddr, "error", ctx.Err())

	subscribeAfterID := afterID
	if subscribeAfterID <= 0 {
		subscribeAfterID = s.events.LastID()
	}
	subscription := s.events.Subscribe(subscribeAfterID)
	defer subscription.Close()
	snapshot := events.Event{
		Type: events.TypeJobsSnapshot, Payload: events.MustPayload(jobAPIResponses(jobSnapshot(s.jobs))), CreatedAt: time.Now().UTC(),
	}
	if err := writeWebSocketEvent(ctx, conn, snapshot); err != nil {
		s.logger.Warn("event websocket snapshot write failed", "remote", r.RemoteAddr, "error", err)
		return
	}
	replayedThrough := subscribeAfterID
	if afterID > 0 {
		replayedThrough, err = s.replayStoredEvents(ctx, conn, afterID)
		if err != nil {
			s.logger.Warn("event websocket replay failed", "remote", r.RemoteAddr, "after_id", afterID, "error", err)
			return
		}
		subscription.Advance(replayedThrough)
	}
	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-subscription.C:
			if !ok {
				return
			}
			if event.ID <= replayedThrough {
				continue
			}
			if err := writeWebSocketEvent(ctx, conn, event); err != nil {
				s.logger.Warn("event websocket write failed", "remote", r.RemoteAddr, "event_id", event.ID, "type", event.Type, "error", err)
				return
			}
			subscription.Advance(event.ID)
		}
	}
}

func jobSnapshot(manager *jobs.Manager) []jobs.Job {
	page, _ := manager.ListPage(0, jobSnapshotLimit)
	return page
}

func (s *Server) replayStoredEvents(ctx context.Context, conn *websocket.Conn, afterID int64) (int64, error) {
	const pageSize = 1000
	replayedThrough := afterID
	for {
		stored, err := s.db.ListDomainEventsAfter(ctx, replayedThrough, pageSize)
		if err != nil {
			return replayedThrough, err
		}
		for _, event := range stored {
			if err := writeWebSocketEvent(ctx, conn, event); err != nil {
				return replayedThrough, err
			}
			if event.ID > replayedThrough {
				replayedThrough = event.ID
			}
		}
		if len(stored) < pageSize {
			return replayedThrough, nil
		}
	}
}

func writeWebSocketEvent(ctx context.Context, conn *websocket.Conn, event events.Event) error {
	b, err := json.Marshal(event)
	if err != nil {
		return err
	}
	return conn.Write(ctx, websocket.MessageText, b)
}
