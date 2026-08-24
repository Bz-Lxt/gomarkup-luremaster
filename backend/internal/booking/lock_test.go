package booking

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestStateMachineRejectsIllegal(t *testing.T) {
	if err := CanSlot(SlotOpen, SlotConfirmed, "CONFIRM"); err == nil {
		t.Fatal("skip lock")
	}
	if err := CanSlot(SlotOpen, SlotLocked, "CLAIM"); err != nil {
		t.Fatal(err)
	}
	if !CanActivity(ActDraft, ActOpen) || CanActivity(ActClosed, ActOpen) {
		t.Fatal("activity machine")
	}
}

func TestThousandConcurrentClaimsExactlyOne(t *testing.T) {
	store := NewMemoryStore(SlotRecord{ID: "s1", Status: SlotOpen})
	locker := &Locker{Store: store, Redis: NewMemoryRedis(), TTL: time.Minute}
	var okN, failN atomic.Int64
	var wg sync.WaitGroup
	n := 1000
	wg.Add(n)
	for i := 0; i < n; i++ {
		i := i
		go func() {
			defer wg.Done()
			_, err := locker.Claim(context.Background(), "s1", "u"+itoa(i))
			if err == nil {
				okN.Add(1)
			} else {
				failN.Add(1)
			}
		}()
	}
	wg.Wait()
	if okN.Load() != 1 {
		t.Fatalf("winners=%d", okN.Load())
	}
	if failN.Load() != int64(n-1) {
		t.Fatalf("losers=%d", failN.Load())
	}
	got, _ := store.GetForUpdate(context.Background(), "s1")
	if got.Status != SlotLocked || got.HolderID == "" {
		t.Fatalf("slot %+v", got)
	}
}

func TestRedisDownStillCorrect(t *testing.T) {
	store := NewMemoryStore(SlotRecord{ID: "s2", Status: SlotOpen})
	r := NewMemoryRedis()
	r.Fail = true
	locker := &Locker{Store: store, Redis: r, TTL: time.Minute}
	var okN atomic.Int64
	var wg sync.WaitGroup
	wg.Add(200)
	for i := 0; i < 200; i++ {
		i := i
		go func() {
			defer wg.Done()
			if _, err := locker.Claim(context.Background(), "s2", "u"+itoa(i)); err == nil {
				okN.Add(1)
			}
		}()
	}
	wg.Wait()
	if okN.Load() != 1 {
		t.Fatalf("redis-down winners=%d", okN.Load())
	}
}

func TestExpireReleases(t *testing.T) {
	now := time.Date(2026, 8, 24, 6, 0, 0, 0, time.UTC)
	store := NewMemoryStore(SlotRecord{ID: "s3", Status: SlotLocked, HolderID: "u1", LockExpiresAt: now.Add(-time.Second)})
	n, err := store.ExpireDue(context.Background(), now)
	if err != nil || n != 1 {
		t.Fatalf("expire %d %v", n, err)
	}
	got, _ := store.GetForUpdate(context.Background(), "s3")
	if got.Status != SlotOpen || got.HolderID != "" {
		t.Fatalf("%+v", got)
	}
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	s := ""
	for i > 0 {
		s = string(rune('0'+i%10)) + s
		i /= 10
	}
	return s
}
