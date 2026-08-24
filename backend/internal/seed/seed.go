package seed

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"luremaster/internal/auth"
	"luremaster/internal/booking"
	"luremaster/internal/logger"
	"luremaster/internal/timeutil"
)

const seedPassword = "LureHunt@2026"

func Run(ctx context.Context, pool *pgxpool.Pool) error {
	var n int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM users WHERE email = $1`, "hunter@lure.local").Scan(&n); err != nil {
		return err
	}
	if n > 0 {
		logger.From().Info("seed skipped", "reason", "hunter exists")
		return nil
	}

	hash, err := auth.HashPassword(seedPassword)
	if err != nil {
		return err
	}
	now := timeutil.NowUTC()

	hunter := uuid.NewString()
	mate := uuid.NewString()
	skipper := uuid.NewString()
	users := []struct {
		id, email, username, nickname string
	}{
		{hunter, "hunter@lure.local", "hunter", "路亚猎人阿凯"},
		{mate, "mate@lure.local", "mate", "搭子阿海"},
		{skipper, "skipper@lure.local", "skipper", "船长老周"},
	}
	for _, u := range users {
		if _, err := pool.Exec(ctx, `
			INSERT INTO users (id, email, username, password_hash, nickname, avatar_url, home_water, lure_pref, credit_score, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, '', '', '', 100, $6, $6)`,
			u.id, u.email, u.username, hash, u.nickname, now); err != nil {
			return fmt.Errorf("seed user %s: %w", u.email, err)
		}
	}

	clubID := uuid.NewString()
	if _, err := pool.Exec(ctx, `
		INSERT INTO clubs (id, name, owner_id, description, created_at)
		VALUES ($1, $2, $3, $4, $5)`,
		clubID, "钱塘夜猎", skipper, "钱塘江口夜钓俱乐部", now); err != nil {
		return err
	}
	members := []struct {
		uid, role string
	}{
		{skipper, "OWNER"},
		{mate, "MEMBER"},
		{hunter, "MEMBER"},
	}
	for _, m := range members {
		if _, err := pool.Exec(ctx, `
			INSERT INTO club_members (club_id, user_id, role, status, joined_at)
			VALUES ($1, $2, $3, 'APPROVED', $4)`, clubID, m.uid, m.role, now); err != nil {
			return err
		}
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO friendships (user_id, friend_id, status, created_at)
		VALUES ($1, $2, 'ACCEPTED', $3)`, hunter, mate, now); err != nil {
		return err
	}

	privateQiandao := uuid.NewString()
	publicQiandao := uuid.NewString()
	publicQiantang := uuid.NewString()
	clubQiantang := uuid.NewString()

	type sp struct {
		id, owner, name, water, structure, vis string
		lat, lon, bearing                      float64
		tidal                                  bool
		club                                   any
		note                                   string
	}
	spots := []sp{
		{privateQiandao, hunter, "千岛湖暗桩湾", "RESERVOIR", "SNAG", "PRIVATE", 29.6050, 119.0050, 75, false, nil, "枯树密布，仅自己知道"},
		{publicQiandao, hunter, "千岛湖深浅交界", "RESERVOIR", "DROPOFF", "PUBLIC", 29.6120, 119.0210, 110, false, nil, "公开标点，适合教学"},
		{publicQiantang, hunter, "钱塘江口洄流带", "RIVER", "EDDY", "PUBLIC", 30.3780, 120.6920, 90, true, nil, "感潮河段，夜钓翘嘴"},
		{clubQiantang, skipper, "钱塘夜猎俱乐部入水口", "RIVER", "INLET", "CLUB", 30.3850, 120.7050, 80, true, clubID, "俱乐部夜钓集合点"},
	}
	for _, p := range spots {
		if _, err := pool.Exec(ctx, `
			INSERT INTO spots (
				id, owner_id, club_id, name, water_type, structure, visibility,
				location, lat, lon, shore_bearing, tidal, note, created_at, updated_at
			) VALUES (
				$1,$2,$3,$4,$5,$6,$7,
				ST_SetSRID(ST_MakePoint($8,$9),4326)::geography,$10,$11,$12,$13,$14,$15,$16
			)`,
			p.id, p.owner, p.club, p.name, p.water, p.structure, p.vis,
			p.lon, p.lat, p.lat, p.lon, p.bearing, p.tidal, p.note, now, now); err != nil {
			return fmt.Errorf("seed spot %s: %w", p.name, err)
		}
		if _, err := pool.Exec(ctx, `
			INSERT INTO spot_depths (id, spot_id, offset_m, depth_m, sampled_at)
			VALUES ($1, $2, 0, 3.5, $3), ($4, $2, 12, 6.2, $3)`,
			uuid.NewString(), p.id, now, uuid.NewString()); err != nil {
			return err
		}
	}

	type ct struct {
		spot, local, species, lure string
		length                     float64
		released                   bool
	}
	catches := []ct{
		{privateQiandao, "2026-08-20 19:15", "MANDARIN", "SOFT", 42, true},
		{publicQiantang, "2026-08-21 20:40", "YELLOWCHECK", "MINNOW", 68, true},
		{publicQiandao, "2026-08-22 18:05", "BASS", "VIB", 36, false},
		{publicQiantang, "2026-08-23 21:10", "SNAKEHEAD", "PENCIL", 55, true},
	}
	for _, rec := range catches {
		caughtAt, _, err := timeutil.ParseCatchLocal(rec.local, timeutil.DefaultTimezone)
		if err != nil {
			return err
		}
		cid := uuid.NewString()
		relInc := 0
		if rec.released {
			relInc = 1
		}
		if _, err := pool.Exec(ctx, `
			INSERT INTO catches (
				id, user_id, spot_id, caught_at, timezone, local_time, species, length_cm, weight_kg,
				lure_type, lure_weight_g, lure_color, retrieve, layer, water_depth_m, water_color,
				turbidity, water_temp_c, current, released, note, photo_key, hydro_status, created_at, updated_at
			) VALUES (
				$1,$2,$3,$4,$5,$6,$7,$8,NULL,
				$9,12,'银白','TWITCH','MID',2.4,'清','中',22,'缓',$10,'seed','','PENDING',$11,$11
			)`,
			cid, hunter, rec.spot, caughtAt, timeutil.DefaultTimezone, rec.local, rec.species, rec.length,
			rec.lure, rec.released, now); err != nil {
			return fmt.Errorf("seed catch: %w", err)
		}
		if _, err := pool.Exec(ctx, `
			INSERT INTO hydro_jobs (id, catch_id, status, attempts, last_error, error_class, next_run_at, created_at, updated_at)
			VALUES ($1, $2, 'PENDING', 0, '', '', $3, $3, $3)
			ON CONFLICT (catch_id) DO NOTHING`, uuid.NewString(), cid, now); err != nil {
			return err
		}
		if _, err := pool.Exec(ctx, `
			INSERT INTO user_stats (user_id, total_catches, released_count, max_length_cm, max_species, streak_days, updated_at)
			VALUES ($1, 1, $2, $3, $4, 1, $5)
			ON CONFLICT (user_id) DO UPDATE SET
				total_catches = user_stats.total_catches + 1,
				released_count = user_stats.released_count + $2,
				max_species = CASE WHEN $3 > user_stats.max_length_cm THEN $4 ELSE user_stats.max_species END,
				max_length_cm = GREATEST(user_stats.max_length_cm, $3),
				updated_at = $5`, hunter, relInc, rec.length, rec.species, now); err != nil {
			return err
		}
	}

	starts, _, err := timeutil.ParseCatchLocal("2026-08-25 18:00", timeutil.DefaultTimezone)
	if err != nil {
		return err
	}
	ends, _, err := timeutil.ParseCatchLocal("2026-08-25 23:00", timeutil.DefaultTimezone)
	if err != nil {
		return err
	}
	actID := uuid.NewString()
	if _, err := pool.Exec(ctx, `
		INSERT INTO activities (
			id, host_id, club_id, spot_id, title, kind, status, starts_at, ends_at,
			meet_lat, meet_lon, meet_radius_m, fee_amount, fee_note, created_at
		) VALUES (
			$1,$2,$3,$4,$5,$6,$7,$8,$9,
			$10,$11,300,0,'免费放位',$12
		)`,
		actID, skipper, clubID, publicQiantang, "钱塘江口热门放位", "HOT_SLOT", booking.ActOpen,
		starts, ends, 30.3780, 120.6920, now); err != nil {
		return err
	}
	for _, label := range []string{"A1", "A2", "A3"} {
		if _, err := pool.Exec(ctx, `
			INSERT INTO slots (id, activity_id, label, status, version)
			VALUES ($1, $2, $3, $4, 0)`, uuid.NewString(), actID, label, booking.SlotOpen); err != nil {
			return err
		}
	}

	logger.From().Info("seed applied", "club", "钱塘夜猎", "users", 3)
	return nil
}
