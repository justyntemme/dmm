package server

import (
	"context"
	"sync"
)

type downloadSlotGate struct {
	mu     sync.Mutex
	active int
	max    int
	wake   chan struct{}
}

func newDownloadSlotGate(max int) *downloadSlotGate {
	if max < 1 {
		max = 1
	}
	return &downloadSlotGate{
		max:  max,
		wake: make(chan struct{}),
	}
}

func (g *downloadSlotGate) acquire(ctx context.Context) bool {
	for {
		g.mu.Lock()
		if g.active < g.max {
			g.active++
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

func (g *downloadSlotGate) release() {
	g.mu.Lock()
	if g.active > 0 {
		g.active--
	}
	g.broadcastLocked()
	g.mu.Unlock()
}

func (g *downloadSlotGate) setMax(max int) {
	if max < 1 {
		max = 1
	}
	g.mu.Lock()
	if g.max != max {
		g.max = max
		g.broadcastLocked()
	}
	g.mu.Unlock()
}

func (g *downloadSlotGate) status() (active int, max int) {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.active, g.max
}

func (g *downloadSlotGate) broadcastLocked() {
	close(g.wake)
	g.wake = make(chan struct{})
}
