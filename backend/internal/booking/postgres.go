package booking

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"luremaster/internal/timeutil"
)

type txCtxKey struct{}

func WithTx(ctx context.Context, tx pgx.Tx) context.Context {
	return context.WithValue(ctx, txCtxKey{}, tx)
}

type querier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

type PostgresStore struct {
	Pool *pgxpool.Pool
}

func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{Pool: pool}
}

func (s *PostgresStore) q(ctx context.Context) querier {
	if tx, ok := ctx.Value(txCtxKey{}).(pgx.Tx); ok && tx != nil {
		return tx
	}
	return s.Pool
}

func (s *PostgresStore) GetForUpdate(ctx context.Context, slotID string) (SlotRecord, error) {
	row := s.q(ctx).QueryRow(ctx, `
		SELECT id, activity_id, label, status, COALESCE(holder_id::text, ''), lock_expires_at, version
		FROM slots
		WHERE id = $1
		FOR UPDATE`, slotID)
	var rec SlotRecord
	var exp *time.Time
	if err := row.Scan(&rec.ID, &rec.ActivityID, &rec.Label, &rec.Status, &rec.HolderID, &exp, &rec.Version); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return SlotRecord{}, ErrBadState
		}
		return SlotRecord{}, err
	}
	if exp != nil {
		rec.LockExpiresAt = exp.UTC()
	}
	return rec, nil
}

func (s *PostgresStore) SaveClaim(ctx context.Context, rec SlotRecord, userID, status string, expires time.Time) error {
	ownTx := false
	q := s.q(ctx)
	if _, ok := ctx.Value(txCtxKey{}).(pgx.Tx); !ok {
		tx, err := s.Pool.Begin(ctx)
		if err != nil {
			return err
		}
		ownTx = true
		q = tx
		defer func() {
			if ownTx {
				_ = tx.Rollback(ctx)
			}
		}()
		ctx = WithTx(ctx, tx)
		_ = ctx
	}
	now := timeutil.NowUTC()
	var tag pgconn.CommandTag
	var err error
	switch status {
	case SlotLocked:
		tag, err = q.Exec(ctx, `
			UPDATE slots
			SET status = $1, holder_id = $2, locked_at = $3, lock_expires_at = $4, version = version + 1
			WHERE id = $5 AND status = $6`,
			SlotLocked, userID, now, expires.UTC(), rec.ID, SlotOpen)
	case SlotConfirmed:
		tag, err = q.Exec(ctx, `
			UPDATE slots
			SET status = $1, holder_id = $2, confirmed_at = $3, lock_expires_at = NULL, version = version + 1
			WHERE id = $4 AND holder_id = $5 AND status = $6`,
			SlotConfirmed, userID, now, rec.ID, userID, SlotLocked)
	case SlotCheckedIn:
		tag, err = q.Exec(ctx, `
			UPDATE slots
			SET status = $1, holder_id = $2, checked_in_at = $3, version = version + 1
			WHERE id = $4 AND holder_id = $5`,
			SlotCheckedIn, userID, now, rec.ID, userID)
	default:
		return ErrBadState
	}
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		if status == SlotLocked {
			return ErrTaken
		}
		return ErrBadState
	}

	if status == SlotLocked {
		_, err = q.Exec(ctx, `
			INSERT INTO slot_holds (slot_id, user_id, status, created_at)
			VALUES ($1, $2, $3, $4)`, rec.ID, userID, status, now)
		if err != nil {
			if isUnique(err) {
				return ErrTaken
			}
			return err
		}
	} else {
		tag, err = q.Exec(ctx, `
			INSERT INTO slot_holds (slot_id, user_id, status, created_at)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (slot_id) DO UPDATE
			SET status = EXCLUDED.status
			WHERE slot_holds.user_id = EXCLUDED.user_id`, rec.ID, userID, status, now)
		if err != nil {
			if isUnique(err) {
				return ErrTaken
			}
			return err
		}
		if tag.RowsAffected() == 0 {
			return ErrTaken
		}
	}

	if ownTx {
		if tx, ok := q.(pgx.Tx); ok {
			if err := tx.Commit(ctx); err != nil {
				return err
			}
			ownTx = false
		}
	}
	return nil
}

func (s *PostgresStore) ReleaseHold(ctx context.Context, slotID string) error {
	ownTx := false
	q := s.q(ctx)
	if _, ok := ctx.Value(txCtxKey{}).(pgx.Tx); !ok {
		tx, err := s.Pool.Begin(ctx)
		if err != nil {
			return err
		}
		ownTx = true
		q = tx
		defer func() {
			if ownTx {
				_ = tx.Rollback(ctx)
			}
		}()
	}
	if _, err := q.Exec(ctx, `
		UPDATE slots
		SET status = $1, holder_id = NULL, locked_at = NULL, lock_expires_at = NULL,
		    confirmed_at = NULL, checked_in_at = NULL, version = version + 1
		WHERE id = $2`, SlotOpen, slotID); err != nil {
		return err
	}
	if _, err := q.Exec(ctx, `DELETE FROM slot_holds WHERE slot_id = $1`, slotID); err != nil {
		return err
	}
	if ownTx {
		if tx, ok := q.(pgx.Tx); ok {
			if err := tx.Commit(ctx); err != nil {
				return err
			}
			ownTx = false
		}
	}
	return nil
}

func (s *PostgresStore) ExpireDue(ctx context.Context, now time.Time) (int, error) {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	rows, err := tx.Query(ctx, `
		UPDATE slots
		SET status = $1, holder_id = NULL, locked_at = NULL, lock_expires_at = NULL, version = version + 1
		WHERE status = $2 AND lock_expires_at IS NOT NULL AND lock_expires_at <= $3
		RETURNING id`, SlotOpen, SlotLocked, now.UTC())
	if err != nil {
		return 0, err
	}
	ids := make([]string, 0)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return 0, err
		}
		ids = append(ids, id)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, err
	}
	if len(ids) > 0 {
		if _, err := tx.Exec(ctx, `DELETE FROM slot_holds WHERE slot_id = ANY($1)`, ids); err != nil {
			return 0, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return len(ids), nil
}

func isUnique(err error) bool {
	var pe *pgconn.PgError
	return errors.As(err, &pe) && pe.Code == "23505"
}
