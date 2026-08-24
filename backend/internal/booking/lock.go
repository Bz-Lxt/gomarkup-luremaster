package booking

import (
	"context"
	"errors"
	"time"
)

var (
	ErrTaken     = errors.New("slot taken")
	ErrNotHolder = errors.New("not holder")
	ErrExpired   = errors.New("lock expired")
	ErrBadState  = errors.New("bad slot state")
)

type DistLock interface {
	TryLock(ctx context.Context, key string, ttl time.Duration) (bool, func(), error)
}

type SlotRecord struct {
	ID            string
	ActivityID    string
	Label         string
	Status        string
	HolderID      string
	LockExpiresAt time.Time
	Version       int
}

type SlotStore interface {
	GetForUpdate(ctx context.Context, slotID string) (SlotRecord, error)
	SaveClaim(ctx context.Context, rec SlotRecord, userID, status string, expires time.Time) error
	ReleaseHold(ctx context.Context, slotID string) error
	ExpireDue(ctx context.Context, now time.Time) (int, error)
}

type Locker struct {
	Store SlotStore
	Redis DistLock
	TTL   time.Duration
	Now   func() time.Time
}

func (l *Locker) now() time.Time {
	if l.Now != nil {
		return l.Now()
	}
	return time.Now().UTC()
}

func (l *Locker) ttl() time.Duration {
	if l.TTL > 0 {
		return l.TTL
	}
	return 15 * time.Minute
}

func (l *Locker) Claim(ctx context.Context, slotID, userID string) (SlotRecord, error) {
	unlock := func() {}
	if l.Redis != nil {
		ok, rel, err := l.Redis.TryLock(ctx, "slot:"+slotID, 8*time.Second)
		if err == nil && !ok {
			return SlotRecord{}, ErrTaken
		}
		if err == nil && rel != nil {
			unlock = rel
		}
		// Redis error → degrade to DB pessimistic lock only
	}
	defer unlock()

	rec, err := l.Store.GetForUpdate(ctx, slotID)
	if err != nil {
		return SlotRecord{}, err
	}
	if rec.Status != SlotOpen {
		return SlotRecord{}, ErrTaken
	}
	if err := CanSlot(rec.Status, SlotLocked, "CLAIM"); err != nil {
		return SlotRecord{}, err
	}
	exp := l.now().Add(l.ttl())
	if err := l.Store.SaveClaim(ctx, rec, userID, SlotLocked, exp); err != nil {
		return SlotRecord{}, err
	}
	rec.Status = SlotLocked
	rec.HolderID = userID
	rec.LockExpiresAt = exp
	rec.Version++
	return rec, nil
}

func (l *Locker) Confirm(ctx context.Context, slotID, userID string) (SlotRecord, error) {
	rec, err := l.Store.GetForUpdate(ctx, slotID)
	if err != nil {
		return SlotRecord{}, err
	}
	if rec.HolderID != userID {
		return SlotRecord{}, ErrNotHolder
	}
	if rec.Status == SlotLocked && !rec.LockExpiresAt.IsZero() && !l.now().Before(rec.LockExpiresAt) {
		_ = l.Store.ReleaseHold(ctx, slotID)
		return SlotRecord{}, ErrExpired
	}
	if err := CanSlot(rec.Status, SlotConfirmed, "CONFIRM"); err != nil {
		return SlotRecord{}, err
	}
	if err := l.Store.SaveClaim(ctx, rec, userID, SlotConfirmed, time.Time{}); err != nil {
		return SlotRecord{}, err
	}
	rec.Status = SlotConfirmed
	return rec, nil
}

func (l *Locker) Release(ctx context.Context, slotID, userID string) error {
	rec, err := l.Store.GetForUpdate(ctx, slotID)
	if err != nil {
		return err
	}
	if rec.HolderID != userID {
		return ErrNotHolder
	}
	reason := "RELEASE"
	target := SlotOpen
	if err := CanSlot(rec.Status, target, reason); err != nil {
		return err
	}
	return l.Store.ReleaseHold(ctx, slotID)
}

func (l *Locker) CheckIn(ctx context.Context, slotID, userID string) (SlotRecord, error) {
	rec, err := l.Store.GetForUpdate(ctx, slotID)
	if err != nil {
		return SlotRecord{}, err
	}
	if rec.HolderID != userID {
		return SlotRecord{}, ErrNotHolder
	}
	if err := CanSlot(rec.Status, SlotCheckedIn, "CHECKIN"); err != nil {
		return SlotRecord{}, err
	}
	if err := l.Store.SaveClaim(ctx, rec, userID, SlotCheckedIn, time.Time{}); err != nil {
		return SlotRecord{}, err
	}
	rec.Status = SlotCheckedIn
	return rec, nil
}
