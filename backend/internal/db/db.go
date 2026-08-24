package db

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"luremaster/internal/logger"
)

const advisoryLockKey int64 = 916016

type DB struct {
	Pool *pgxpool.Pool
}

func Open(ctx context.Context, url string) (*DB, error) {
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, fmt.Errorf("parse database url: %w", err)
	}
	if cfg.MaxConns < 8 {
		cfg.MaxConns = 8
	}
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("open pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}
	return &DB{Pool: pool}, nil
}

func (d *DB) Close() {
	if d != nil && d.Pool != nil {
		d.Pool.Close()
	}
}

func (d *DB) Ping(ctx context.Context) error {
	return d.Pool.Ping(ctx)
}

// Migrate acquires pg_advisory_lock(916016) on the same connection that
// subsequently executes every migrations/*.sql file, then unlocks.
func (d *DB) Migrate(ctx context.Context, dir string) error {
	if dir == "" {
		return fmt.Errorf("migrations dir is empty")
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("read migrations %s: %w", dir, err)
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".sql") {
			continue
		}
		names = append(names, e.Name())
	}
	sort.Strings(names)

	conn, err := d.Pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("acquire conn: %w", err)
	}
	defer conn.Release()

	if _, err := conn.Exec(ctx, "SELECT pg_advisory_lock($1)", advisoryLockKey); err != nil {
		return fmt.Errorf("advisory lock: %w", err)
	}
	defer func() {
		if _, uerr := conn.Exec(context.Background(), "SELECT pg_advisory_unlock($1)", advisoryLockKey); uerr != nil {
			logger.From().Error("advisory unlock", "err", uerr)
		}
	}()

	for _, name := range names {
		body, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return fmt.Errorf("read %s: %w", name, err)
		}
		sql := strings.TrimSpace(string(body))
		if sql == "" {
			continue
		}
		if _, err := conn.Exec(ctx, sql); err != nil {
			return fmt.Errorf("migrate %s: %w", name, err)
		}
		logger.From().Info("migration applied", "file", name)
	}
	return nil
}

func ResolveMigrationsDir(explicit string) string {
	cands := []string{explicit, os.Getenv("MIGRATIONS_DIR"), "migrations", "/app/migrations"}
	for _, c := range cands {
		c = strings.TrimSpace(c)
		if c == "" {
			continue
		}
		if st, err := os.Stat(c); err == nil && st.IsDir() {
			return c
		}
	}
	return "migrations"
}
