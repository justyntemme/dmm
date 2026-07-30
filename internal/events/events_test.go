package events

import "testing"

func TestSubscribeReplaysEventsAfterID(t *testing.T) {
	bus := NewBus(3)
	first := bus.Publish(Event{Type: TypeGameChanged})
	second := bus.Publish(Event{Type: TypeJobUpdated})

	sub := bus.Subscribe(first.ID)
	defer sub.Close()

	got := <-sub.C
	if got.ID != second.ID || got.Type != TypeJobUpdated {
		t.Fatalf("replayed event = %+v, want second event %+v", got, second)
	}
}

func TestPublishDropsSlowSubscriberInsteadOfBlocking(t *testing.T) {
	bus := NewBus(2)
	sub := bus.Subscribe(0)
	defer sub.Close()

	for i := 0; i < 128; i++ {
		bus.Publish(Event{Type: TypeJobUpdated})
	}
}

func TestHistoryIsBounded(t *testing.T) {
	bus := NewBus(2)
	first := bus.Publish(Event{Type: TypeGameChanged})
	bus.Publish(Event{Type: TypeJobUpdated})
	third := bus.Publish(Event{Type: TypeDeploymentChanged})

	sub := bus.Subscribe(first.ID - 1)
	defer sub.Close()

	got := <-sub.C
	if got.ID == first.ID {
		t.Fatalf("oldest event was replayed despite bounded history: %+v", got)
	}
	if got.ID >= third.ID {
		t.Fatalf("first replayed event = %+v, want event before third", got)
	}
}

func TestSubscribeReplaysFullBoundedHistoryWithoutBlocking(t *testing.T) {
	bus := NewBus(96)
	for i := 0; i < 96; i++ {
		bus.Publish(Event{Type: TypeJobUpdated})
	}

	sub := bus.Subscribe(0)
	defer sub.Close()

	for i := 0; i < 96; i++ {
		select {
		case <-sub.C:
		default:
			t.Fatalf("replayed %d events, want full bounded history", i)
		}
	}
}

func TestNewBusWithHistoryPreservesEventIDs(t *testing.T) {
	bus := NewBusWithHistory(4, []Event{
		{ID: 7, Type: TypeGameChanged},
		{ID: 8, Type: TypeJobUpdated},
	})
	next := bus.Publish(Event{Type: TypeDeploymentChanged})
	if next.ID != 9 {
		t.Fatalf("next event id = %d, want 9", next.ID)
	}

	sub := bus.Subscribe(7)
	defer sub.Close()
	got := <-sub.C
	if got.ID != 8 {
		t.Fatalf("replayed event id = %d, want 8", got.ID)
	}
}

func TestLastIDReportsHighWaterMark(t *testing.T) {
	bus := NewBusWithHistory(4, []Event{
		{ID: 7, Type: TypeGameChanged},
		{ID: 9, Type: TypeJobUpdated},
	})
	if got := bus.LastID(); got != 9 {
		t.Fatalf("last id = %d, want 9", got)
	}
	next := bus.Publish(Event{Type: TypeDeploymentChanged})
	if got := bus.LastID(); got != next.ID {
		t.Fatalf("last id after publish = %d, want %d", got, next.ID)
	}
}
