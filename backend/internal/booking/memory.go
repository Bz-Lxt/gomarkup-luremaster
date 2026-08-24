package booking

import (
	"context"
	"sync"
	"time"
)

// MemoryStore is the authoritative in-process stand-in used by concurrency tests.
// Production uses PostgresStore (SELECT FOR UPDATE + slot_holds unique).
type MemoryStore struct {
	mu    sync.Mutex
	slots map[string]*SlotRecord
}

func NewMemoryStore(slots ...SlotRecord) *MemoryStore {
	m := &MemoryStore{slots: map[string]*SlotRecord{}}
	for i := range slots {
		cp := slots[i]
		m.slots[cp.ID] = &cp
	}
	return m
}

func (m *MemoryStore) GetForUpdate(_ context.Context, slotID string) (SlotRecord, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.slots[slotID]
	if !ok {
		return SlotRecord{}, ErrBadState
	}
	return *s, nil
}

func (m *MemoryStore) SaveClaim(_ context.Context, rec SlotRecord, userID, status string, expires time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.slots[rec.ID]
	if !ok {
		return ErrBadState
	}
	if s.Status != SlotOpen && status == SlotLocked {
		return ErrTaken
	}
	if status == SlotLocked && s.Status != SlotOpen {
		return ErrTaken
	}
	s.Status = status
	s.HolderID = userID
	s.LockExpiresAt = expires
	s.Version++
	return nil
}

func (m *MemoryStore) ReleaseHold(_ context.Context, slotID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.slots[slotID]
	if !ok {
		return ErrBadState
	}
	s.Status = SlotOpen
	s.HolderID = ""
	s.LockExpiresAt = time.Time{}
	s.Version++
	return nil
}

func (m *MemoryStore) ExpireDue(_ context.Context, now time.Time) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	n := 0
	for _, s := range m.slots {
		if s.Status == SlotLocked && !s.LockExpiresAt.IsZero() && !now.Before(s.LockExpiresAt) {
			s.Status = SlotOpen
			s.HolderID = ""
			s.LockExpiresAt = time.Time{}
			n++
		}
	}
	return n, nil
}

type MemoryRedis struct {
	mu   sync.Mutex
	keys map[string]struct{}
	Fail bool
}

func NewMemoryRedis() *MemoryRedis { return &MemoryRedis{keys: map[string]struct{}{}} }

func (r *MemoryRedis) TryLock(_ context.Context, key string, _ time.Duration) (bool, func(), error) {
	if r.Fail {
		return false, nil, errorsRedisDown
	}
	r.mu.Lock()
	if _, ok := r.keys[key]; ok {
		r.mu.Unlock()
		return false, nil, nil
	}
	r.keys[key] = struct{}{}
	r.mu.Unlock()
	return true, func() {
		r.mu.Lock()
		delete(r.keys, key)
		r.mu.Unlock()
	}, nil
}

var errorsRedisDown = errRedisDown{}

type errRedisDown struct{}

func (errRedisDown) Error() string { return "redis down" }
