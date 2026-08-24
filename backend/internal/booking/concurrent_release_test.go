package booking_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"luremaster/internal/booking"
)

type firstReadGate struct {
	*booking.MemoryStore
	slotID  string
	entered chan struct{}
	resume  chan struct{}
	once    sync.Once
}

func (s *firstReadGate) GetForUpdate(ctx context.Context, slotID string) (booking.SlotRecord, error) {
	rec, err := s.MemoryStore.GetForUpdate(ctx, slotID)
	if err != nil || slotID != s.slotID {
		return rec, err
	}
	wait := false
	s.once.Do(func() {
		wait = true
		close(s.entered)
	})
	if wait {
		select {
		case <-s.resume:
		case <-ctx.Done():
			return booking.SlotRecord{}, ctx.Err()
		}
	}
	return rec, nil
}

func TestConcurrentClaimsReleaseTheirOwnDistributedLocks(t *testing.T) {
	store := &firstReadGate{
		MemoryStore: booking.NewMemoryStore(
			booking.SlotRecord{ID: "slot-a", Status: booking.SlotOpen},
			booking.SlotRecord{ID: "slot-b", Status: booking.SlotOpen},
		),
		slotID:  "slot-a",
		entered: make(chan struct{}),
		resume:  make(chan struct{}),
	}
	locker := &booking.Locker{
		Store: store,
		Redis: booking.NewMemoryRedis(),
		TTL:   time.Minute,
	}

	claimA := make(chan error, 1)
	go func() {
		_, err := locker.Claim(context.Background(), "slot-a", "user-a")
		claimA <- err
	}()

	select {
	case <-store.entered:
	case <-time.After(3 * time.Second):
		t.Fatal("first claim did not reach the store")
	}

	if _, err := locker.Claim(context.Background(), "slot-b", "user-b"); err != nil {
		close(store.resume)
		t.Fatalf("claiming an independent slot: %v", err)
	}
	close(store.resume)
	if err := <-claimA; err != nil {
		t.Fatalf("first claim: %v", err)
	}

	if err := locker.Release(context.Background(), "slot-a", "user-a"); err != nil {
		t.Fatalf("release first slot: %v", err)
	}
	if _, err := locker.Claim(context.Background(), "slot-a", "user-c"); err != nil {
		t.Fatalf("claim after release: %v", err)
	}
}
