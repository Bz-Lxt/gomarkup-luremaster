package hydro

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"luremaster/internal/logger"
	"luremaster/internal/timeutil"
)

const (
	JobPending = "PENDING"
	JobRunning = "RUNNING"
	JobBound   = "BOUND"
	JobFailed  = "FAILED"

	MaxAttempts     = 3
	StuckRunningFor = 2 * time.Minute
)

type Worker struct {
	Pool   *pgxpool.Pool
	Engine *Engine
}

func NewWorker(pool *pgxpool.Pool, engine *Engine) *Worker {
	return &Worker{Pool: pool, Engine: engine}
}

// ShouldRetry is the narrow-retry gate: only TRANSIENT, and only while attempts < 3.
func ShouldRetry(class string, attempts int) bool {
	return Retryable(class) && attempts < MaxAttempts
}

func RetryBackoff(attempts int) time.Duration {
	if attempts < 1 {
		attempts = 1
	}
	d := time.Second << uint(attempts-1)
	if d > 16*time.Second {
		d = 16 * time.Second
	}
	return d
}

func (w *Worker) Recovery(ctx context.Context) error {
	now := timeutil.NowUTC()
	cutoff := now.Add(-StuckRunningFor)
	tag, err := w.Pool.Exec(ctx, `
		UPDATE hydro_jobs
		SET status = $1, updated_at = $2
		WHERE status = $3 AND started_at IS NOT NULL AND started_at <= $4`,
		JobPending, now, JobRunning, cutoff)
	if err != nil {
		return err
	}
	if tag.RowsAffected() > 0 {
		logger.From().Info("hydro recovery", "reset", tag.RowsAffected())
	}
	return nil
}

func (w *Worker) Tick(ctx context.Context) error {
	now := timeutil.NowUTC()
	rows, err := w.Pool.Query(ctx, `
		SELECT id
		FROM hydro_jobs
		WHERE status IN ($1, $2) AND next_run_at <= $3
		ORDER BY next_run_at ASC
		LIMIT 8`, JobPending, JobRunning, now)
	if err != nil {
		return err
	}
	ids := make([]string, 0)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		ids = append(ids, id)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	for _, id := range ids {
		if err := w.process(ctx, id); err != nil {
			logger.From().Error("hydro job", "id", id, "err", err)
		}
	}
	return nil
}

func (w *Worker) process(ctx context.Context, jobID string) error {
	tx, err := w.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var (
		catchID   string
		status    string
		attempts  int
		startedAt *time.Time
	)
	err = tx.QueryRow(ctx, `
		SELECT catch_id, status, attempts, started_at
		FROM hydro_jobs
		WHERE id = $1
		FOR UPDATE`, jobID).Scan(&catchID, &status, &attempts, &startedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return err
	}
	if status == JobBound || status == JobFailed {
		return nil
	}
	now := timeutil.NowUTC()
	if status == JobRunning && startedAt != nil && now.Sub(startedAt.UTC()) < StuckRunningFor {
		return nil
	}

	attempts++
	if _, err := tx.Exec(ctx, `
		UPDATE hydro_jobs
		SET status = $1, started_at = $2, attempts = $3, updated_at = $2
		WHERE id = $4`, JobRunning, now, attempts, jobID); err != nil {
		return err
	}

	var (
		caughtAt  time.Time
		lat, lon  float64
		bearing   float64
		tidal     bool
		waterTemp *float64
	)
	err = tx.QueryRow(ctx, `
		SELECT c.caught_at, c.water_temp_c, s.lat, s.lon, s.shore_bearing, s.tidal
		FROM catches c
		JOIN spots s ON s.id = c.spot_id
		WHERE c.id = $1`, catchID).Scan(&caughtAt, &waterTemp, &lat, &lon, &bearing, &tidal)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return w.failOrRetry(ctx, tx, jobID, catchID, attempts, ClassValidation, err, now)
		}
		return err
	}

	snap, err := w.Engine.Retro(Query{
		Lat:          lat,
		Lon:          lon,
		At:           caughtAt.UTC(),
		ShoreBearing: bearing,
		Tidal:        tidal,
		WaterTempC:   waterTemp,
	})
	if err != nil {
		class := ClassifyError(err)
		return w.failOrRetry(ctx, tx, jobID, catchID, attempts, class, err, now)
	}

	contrib, err := json.Marshal(snap.Contributions)
	if err != nil {
		contrib = []byte("[]")
	}
	series, err := json.Marshal(seriesPayload(snap))
	if err != nil {
		series = []byte("{}")
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO hydro_snapshots (
			catch_id, bound_at, bind_error_sec, pressure_hpa, pressure_delta_3h, pressure_trend,
			air_temp_c, wind_dir_deg, wind_dir_label, wind_speed_ms, beaufort, shore_aspect,
			tide_height_m, tide_phase_pct, tide_window, moon_phase, moon_illum_pct,
			bite_score, frenzy, contributions, series, created_at
		) VALUES (
			$1,$2,$3,$4,$5,$6,
			$7,$8,$9,$10,$11,$12,
			$13,$14,$15,$16,$17,
			$18,$19,$20,$21,$22
		)
		ON CONFLICT (catch_id) DO UPDATE SET
			bound_at = EXCLUDED.bound_at,
			bind_error_sec = EXCLUDED.bind_error_sec,
			pressure_hpa = EXCLUDED.pressure_hpa,
			pressure_delta_3h = EXCLUDED.pressure_delta_3h,
			pressure_trend = EXCLUDED.pressure_trend,
			air_temp_c = EXCLUDED.air_temp_c,
			wind_dir_deg = EXCLUDED.wind_dir_deg,
			wind_dir_label = EXCLUDED.wind_dir_label,
			wind_speed_ms = EXCLUDED.wind_speed_ms,
			beaufort = EXCLUDED.beaufort,
			shore_aspect = EXCLUDED.shore_aspect,
			tide_height_m = EXCLUDED.tide_height_m,
			tide_phase_pct = EXCLUDED.tide_phase_pct,
			tide_window = EXCLUDED.tide_window,
			moon_phase = EXCLUDED.moon_phase,
			moon_illum_pct = EXCLUDED.moon_illum_pct,
			bite_score = EXCLUDED.bite_score,
			frenzy = EXCLUDED.frenzy,
			contributions = EXCLUDED.contributions,
			series = EXCLUDED.series`,
		catchID, now, snap.BindErrorSec, snap.PressureHPa, snap.PressureDelta3h, snap.PressureTrend,
		snap.AirTempC, snap.WindDirDeg, snap.WindDirLabel, snap.WindSpeedMS, snap.Beaufort, snap.ShoreAspect,
		snap.TideHeightM, snap.TidePhasePct, snap.TideWindow, snap.MoonPhase, snap.MoonIllumPct,
		snap.BiteScore, snap.Frenzy, contrib, series, now); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE hydro_jobs
		SET status = $1, last_error = '', error_class = '', finished_at = $2, updated_at = $2
		WHERE id = $3`, JobBound, now, jobID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE catches SET hydro_status = $1, updated_at = $2 WHERE id = $3`, JobBound, now, catchID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (w *Worker) failOrRetry(ctx context.Context, tx pgx.Tx, jobID, catchID string, attempts int, class string, cause error, now time.Time) error {
	msg := ""
	if cause != nil {
		msg = cause.Error()
	}
	if !ShouldRetry(class, attempts) {
		if _, err := tx.Exec(ctx, `
			UPDATE hydro_jobs
			SET status = $1, last_error = $2, error_class = $3, finished_at = $4, updated_at = $4
			WHERE id = $5`, JobFailed, msg, class, now, jobID); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
			UPDATE catches SET hydro_status = $1, updated_at = $2 WHERE id = $3`, JobFailed, now, catchID); err != nil {
			return err
		}
		return tx.Commit(ctx)
	}
	next := now.Add(RetryBackoff(attempts))
	if _, err := tx.Exec(ctx, `
		UPDATE hydro_jobs
		SET status = $1, last_error = $2, error_class = $3, next_run_at = $4, updated_at = $5
		WHERE id = $6`, JobPending, msg, class, next, now, jobID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

type jobExecer interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
}

func EnqueueJob(ctx context.Context, q jobExecer, catchID string) error {
	now := timeutil.NowUTC()
	_, err := q.Exec(ctx, `
		INSERT INTO hydro_jobs (id, catch_id, status, attempts, last_error, error_class, next_run_at, created_at, updated_at)
		VALUES ($1, $2, $3, 0, '', '', $4, $4, $4)
		ON CONFLICT (catch_id) DO NOTHING`, uuid.NewString(), catchID, JobPending, now)
	return err
}

func seriesPayload(snap Snapshot) map[string]any {
	hourly := make([]map[string]any, 0, len(snap.Hourly))
	for _, s := range snap.Hourly {
		hourly = append(hourly, map[string]any{
			"at":           timeutil.FormatRFC3339(s.At),
			"pressure_hpa": s.PressureHPa,
			"temp_c":       s.TempC,
			"wind_dir_deg": s.WindDirDeg,
			"wind_ms":      s.WindMS,
		})
	}
	tides := make([]map[string]any, 0, len(snap.Tides))
	for _, t := range snap.Tides {
		tides = append(tides, map[string]any{
			"at":       timeutil.FormatRFC3339(t.At),
			"height_m": t.HeightM,
		})
	}
	return map[string]any{"hourly": hourly, "tides": tides}
}
