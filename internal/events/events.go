package events

import (
	"encoding/json"
	"sync"
	"time"
)

type Type string

const (
	TypeJobsSnapshot       = "jobs.snapshot"
	TypeJobUpdated         = "job.updated"
	TypeGameChanged        = "game.changed"
	TypeProfileModsChanged = "profile_mods.changed"
	TypeDeploymentChanged  = "deployment.changed"
	TypeLaunchChanged      = "launch.changed"
	TypeWorkshopChanged    = "workshop.changed"
	TypeInstallChanged     = "install.changed"
	TypeUIChanged          = "ui.changed"
)

type Event struct {
	ID        int64           `json:"id"`
	Type      Type            `json:"type"`
	AppID     string          `json:"app_id,omitempty"`
	JobID     string          `json:"job_id,omitempty"`
	Payload   json.RawMessage `json:"payload,omitempty"`
	CreatedAt time.Time       `json:"created_at"`
}

type Bus struct {
	mu          sync.Mutex
	nextID      int64
	maxHistory  int
	history     []Event
	subscribers map[chan Event]struct{}
}

type Subscription struct {
	C     <-chan Event
	once  sync.Once
	close func()
}

func NewBus(maxHistory int) *Bus {
	if maxHistory <= 0 {
		maxHistory = 256
	}
	return &Bus{
		maxHistory:  maxHistory,
		subscribers: map[chan Event]struct{}{},
	}
}

func NewBusWithHistory(maxHistory int, history []Event) *Bus {
	bus := NewBus(maxHistory)
	for _, event := range history {
		bus.appendLocked(event)
	}
	return bus
}

func (b *Bus) Publish(event Event) Event {
	b.mu.Lock()
	if event.ID <= 0 {
		b.nextID++
		event.ID = b.nextID
	}
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now().UTC()
	}
	b.appendLocked(event)
	subscribers := make([]chan Event, 0, len(b.subscribers))
	for ch := range b.subscribers {
		subscribers = append(subscribers, ch)
	}
	b.mu.Unlock()

	for _, ch := range subscribers {
		select {
		case ch <- event:
		default:
		}
	}
	return event
}

func (b *Bus) appendLocked(event Event) {
	if event.ID > b.nextID {
		b.nextID = event.ID
	}
	b.history = append(b.history, event)
	if len(b.history) > b.maxHistory {
		b.history = b.history[len(b.history)-b.maxHistory:]
	}
}

func (b *Bus) Subscribe(afterID int64) *Subscription {
	ch := make(chan Event, b.maxHistory+64)
	b.mu.Lock()
	for _, event := range b.history {
		if event.ID > afterID {
			ch <- event
		}
	}
	b.subscribers[ch] = struct{}{}
	b.mu.Unlock()
	return &Subscription{
		C: ch,
		close: func() {
			b.mu.Lock()
			delete(b.subscribers, ch)
			b.mu.Unlock()
			close(ch)
		},
	}
}

func (b *Bus) LastID() int64 {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.nextID
}

func (s *Subscription) Close() {
	if s == nil || s.close == nil {
		return
	}
	s.once.Do(s.close)
}

func MustPayload(value any) json.RawMessage {
	b, err := json.Marshal(value)
	if err != nil {
		return nil
	}
	return b
}
