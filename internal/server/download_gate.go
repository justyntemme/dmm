package server

import (
	"context"
	"sync"
)

type downloadSlotGate struct {
	mu          sync.Mutex
	active      int
	max         int
	maxPerKey   int
	activeByKey map[string]int
	wake        chan struct{}
}

func newDownloadSlotGate(max, maxPerKey int) *downloadSlotGate {
	if max < 1 {
		max = 1
	}
	if maxPerKey < 1 {
		maxPerKey = 1
	}
	if maxPerKey > max {
		maxPerKey = max
	}
	return &downloadSlotGate{
		max:         max,
		maxPerKey:   maxPerKey,
		activeByKey: map[string]int{},
		wake:        make(chan struct{}),
	}
}

func (g *downloadSlotGate) acquire(ctx context.Context, key string) bool {
	key = normalizeDownloadGateKey(key)
	for {
		g.mu.Lock()
		if g.active < g.max && g.activeByKey[key] < g.maxPerKey {
			g.active++
			g.activeByKey[key]++
			g.mu.Unlock()
			return true
		}
		wake := g.wake
		g.mu.Unlock()

		select {
		case <-ctx.Done():
			return false
		case <-wake:
		}
	}
}

func (g *downloadSlotGate) release(key string) {
	key = normalizeDownloadGateKey(key)
	g.mu.Lock()
	if g.active > 0 {
		g.active--
	}
	if g.activeByKey[key] > 1 {
		g.activeByKey[key]--
	} else {
		delete(g.activeByKey, key)
	}
	g.broadcastLocked()
	g.mu.Unlock()
}

func (g *downloadSlotGate) setLimits(max, maxPerKey int) {
	if max < 1 {
		max = 1
	}
	if maxPerKey < 1 {
		maxPerKey = 1
	}
	if maxPerKey > max {
		maxPerKey = max
	}
	g.mu.Lock()
	if g.max != max || g.maxPerKey != maxPerKey {
		g.max = max
		g.maxPerKey = maxPerKey
		g.broadcastLocked()
	}
	g.mu.Unlock()
}

func (g *downloadSlotGate) status() downloadSlotGateStatus {
	g.mu.Lock()
	defer g.mu.Unlock()
	activeByKey := make(map[string]int, len(g.activeByKey))
	for key, active := range g.activeByKey {
		activeByKey[key] = active
	}
	return downloadSlotGateStatus{
		Active:      g.active,
		Max:         g.max,
		MaxPerKey:   g.maxPerKey,
		ActiveByKey: activeByKey,
	}
}

func (g *downloadSlotGate) broadcastLocked() {
	close(g.wake)
	g.wake = make(chan struct{})
}

type downloadSlotGateStatus struct {
	Active      int
	Max         int
	MaxPerKey   int
	ActiveByKey map[string]int
}

func normalizeDownloadGateKey(key string) string {
	if key == "" {
		return "default"
	}
	return key
}
