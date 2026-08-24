package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"luremaster/internal/hydro"
	"luremaster/internal/model"
	"luremaster/internal/timeutil"
)

type createCatchReq struct {
	SpotID      string   `json:"spot_id"`
	LocalTime   string   `json:"local_time"`
	Timezone    string   `json:"timezone"`
	Species     string   `json:"species"`
	LengthCM    float64  `json:"length_cm"`
	WeightKG    *float64 `json:"weight_kg"`
	LureType    string   `json:"lure_type"`
	LureWeightG *float64 `json:"lure_weight_g"`
	LureColor   string   `json:"lure_color"`
	Retrieve    string   `json:"retrieve"`
	Layer       string   `json:"layer"`
	WaterDepthM *float64 `json:"water_depth_m"`
	WaterColor  string   `json:"water_color"`
	Turbidity   string   `json:"turbidity"`
	WaterTempC  *float64 `json:"water_temp_c"`
	Current     string   `json:"current"`
	Released    *bool    `json:"released"`
	Note        string   `json:"note"`
}

type CatchDTO struct {
	ID          string         `json:"id"`
	UserID      string         `json:"user_id"`
	SpotID      string         `json:"spot_id"`
	CaughtAt    string         `json:"caught_at"`
	Timezone    string         `json:"timezone"`
	LocalTime   string         `json:"local_time"`
	Species     string         `json:"species"`
	LengthCM    float64        `json:"length_cm"`
	WeightKG    *float64       `json:"weight_kg"`
	LureType    string         `json:"lure_type"`
	LureWeightG *float64       `json:"lure_weight_g"`
	LureColor   string         `json:"lure_color"`
	Retrieve    string         `json:"retrieve"`
	Layer       string         `json:"layer"`
	WaterDepthM *float64       `json:"water_depth_m"`
	WaterColor  string         `json:"water_color"`
	Turbidity   string         `json:"turbidity"`
	WaterTempC  *float64       `json:"water_temp_c"`
	Current     string         `json:"current"`
	Released    bool           `json:"released"`
	Note        string         `json:"note"`
	PhotoKey    string         `json:"photo_key"`
	PhotoURL    string         `json:"photo_url"`
	HydroStatus string         `json:"hydro_status"`
	Hydro       map[string]any `json:"hydro"`
	CreatedAt   string         `json:"created_at"`
}

func (s *Server) ListCatches(c *gin.Context) {
	ctx := c.Request.Context()
	uid := currentUserID(c)
	rows, err := s.DB.Query(ctx, `
		SELECT id, user_id, spot_id, caught_at, timezone, local_time, species, length_cm, weight_kg,
		       lure_type, lure_weight_g, lure_color, retrieve, layer, water_depth_m, water_color,
		       turbidity, water_temp_c, current, released, note, photo_key, hydro_status, created_at
		FROM catches WHERE user_id = $1 ORDER BY caught_at DESC LIMIT 200`, uid)
	if err != nil {
		Abort(c, err)
		return
	}
	defer rows.Close()
	out := []CatchDTO{}
	for rows.Next() {
		dto, err := scanCatch(rows)
		if err != nil {
			Abort(c, err)
			return
		}
		dto.PhotoURL = s.Store.URL(dto.PhotoKey)
		dto.Hydro = map[string]any{"status": dto.HydroStatus}
		out = append(out, dto)
	}
	OK(c, out)
}

func (s *Server) CreateCatch(c *gin.Context) {
	var req createCatchReq
	if err := c.ShouldBindJSON(&req); err != nil {
		Abort(c, Validation("invalid json"))
		return
	}
	if !parseUUID(req.SpotID) {
		Abort(c, Validation("spot_id required"))
		return
	}
	species := model.NormalizeEnum(req.Species)
	lureType := model.NormalizeEnum(req.LureType)
	if !model.InSet(species, model.Species) {
		Abort(c, Validation("invalid species"))
		return
	}
	if !model.InSet(lureType, model.LureTypes) {
		Abort(c, Validation("invalid lure_type"))
		return
	}
	if req.LengthCM <= 0 {
		Abort(c, Validation("length_cm required"))
		return
	}
	tz := strings.TrimSpace(req.Timezone)
	if tz == "" {
		tz = timeutil.DefaultTimezone
	}
	caughtAt, _, err := timeutil.ParseCatchLocal(req.LocalTime, tz)
	if err != nil {
		Abort(c, Validation("invalid local_time"))
		return
	}
	ctx := c.Request.Context()
	uid := currentUserID(c)
	row, err := s.loadSpot(ctx, req.SpotID)
	if err != nil {
		Abort(c, err)
		return
	}
	clubID := ""
	if row.ClubID != nil {
		clubID = *row.ClubID
	}
	if !canListSpot(row.Visibility, row.OwnerID, uid, s.isClubMember(ctx, clubID, uid), s.areFriends(ctx, uid, row.OwnerID)) {
		Abort(c, ErrForbidden)
		return
	}
	released := true
	if req.Released != nil {
		released = *req.Released
	}
	now := timeutil.NowUTC()
	id := uuid.NewString()
	relInc := 0
	if released {
		relInc = 1
	}
	err = s.withTx(ctx, func(txctx context.Context, tx pgx.Tx) error {
		_, err := tx.Exec(txctx, `
			INSERT INTO catches (
				id, user_id, spot_id, caught_at, timezone, local_time, species, length_cm, weight_kg,
				lure_type, lure_weight_g, lure_color, retrieve, layer, water_depth_m, water_color,
				turbidity, water_temp_c, current, released, note, photo_key, hydro_status, created_at, updated_at
			) VALUES (
				$1,$2,$3,$4,$5,$6,$7,$8,$9,
				$10,$11,$12,$13,$14,$15,$16,
				$17,$18,$19,$20,$21,'','PENDING',$22,$22
			)`,
			id, uid, req.SpotID, caughtAt, tz, strings.TrimSpace(req.LocalTime), species, req.LengthCM, req.WeightKG,
			lureType, req.LureWeightG, strings.TrimSpace(req.LureColor), strings.TrimSpace(req.Retrieve),
			strings.TrimSpace(req.Layer), req.WaterDepthM, strings.TrimSpace(req.WaterColor),
			strings.TrimSpace(req.Turbidity), req.WaterTempC, strings.TrimSpace(req.Current),
			released, strings.TrimSpace(req.Note), now)
		if err != nil {
			return err
		}
		if err := hydro.EnqueueJob(txctx, tx, id); err != nil {
			return err
		}
		_, err = tx.Exec(txctx, `
			INSERT INTO user_stats (user_id, total_catches, released_count, max_length_cm, max_species, streak_days, updated_at)
			VALUES ($1, 1, $2, $3, $4, 1, $5)
			ON CONFLICT (user_id) DO UPDATE SET
				total_catches = user_stats.total_catches + 1,
				released_count = user_stats.released_count + $2,
				max_species = CASE WHEN $3 > user_stats.max_length_cm THEN $4 ELSE user_stats.max_species END,
				max_length_cm = GREATEST(user_stats.max_length_cm, $3),
				updated_at = $5`, uid, relInc, req.LengthCM, species, now)
		return err
	})
	if err != nil {
		Abort(c, err)
		return
	}
	dto, err := s.loadCatch(ctx, id, false)
	if err != nil {
		Abort(c, err)
		return
	}
	Created(c, dto)
}

func (s *Server) GetCatch(c *gin.Context) {
	id := c.Param("id")
	if !parseUUID(id) {
		Abort(c, ErrNotFound)
		return
	}
	dto, err := s.loadCatch(c.Request.Context(), id, true)
	if err != nil {
		Abort(c, err)
		return
	}
	if dto.UserID != currentUserID(c) {
		Abort(c, ErrForbidden)
		return
	}
	OK(c, dto)
}

func (s *Server) UploadCatchPhoto(c *gin.Context) {
	id := c.Param("id")
	if !parseUUID(id) {
		Abort(c, ErrNotFound)
		return
	}
	ctx := c.Request.Context()
	dto, err := s.loadCatch(ctx, id, false)
	if err != nil {
		Abort(c, err)
		return
	}
	if dto.UserID != currentUserID(c) {
		Abort(c, ErrForbidden)
		return
	}
	fh, err := c.FormFile("file")
	if err != nil {
		Abort(c, Validation("file required"))
		return
	}
	if fh.Size > 8<<20 {
		Abort(c, Validation("file too large"))
		return
	}
	ext := strings.ToLower(filepath.Ext(fh.Filename))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".webp", ".gif":
	default:
		Abort(c, Validation("unsupported file type"))
		return
	}
	src, err := fh.Open()
	if err != nil {
		Abort(c, err)
		return
	}
	defer src.Close()
	key := "catches/" + id + "/" + uuid.NewString() + ext
	ct := fh.Header.Get("Content-Type")
	if err := s.Store.Put(ctx, key, ct, src, fh.Size); err != nil {
		Abort(c, err)
		return
	}
	_, err = s.DB.Exec(ctx, `UPDATE catches SET photo_key = $1, updated_at = $2 WHERE id = $3`, key, timeutil.NowUTC(), id)
	if err != nil {
		Abort(c, err)
		return
	}
	dto.PhotoKey = key
	dto.PhotoURL = s.Store.URL(key)
	OK(c, dto)
}

func (s *Server) loadCatch(ctx context.Context, id string, withHydro bool) (CatchDTO, error) {
	row, err := s.DB.Query(ctx, `
		SELECT id, user_id, spot_id, caught_at, timezone, local_time, species, length_cm, weight_kg,
		       lure_type, lure_weight_g, lure_color, retrieve, layer, water_depth_m, water_color,
		       turbidity, water_temp_c, current, released, note, photo_key, hydro_status, created_at
		FROM catches WHERE id = $1`, id)
	if err != nil {
		return CatchDTO{}, err
	}
	defer row.Close()
	if !row.Next() {
		return CatchDTO{}, ErrNotFound
	}
	dto, err := scanCatch(row)
	if err != nil {
		return CatchDTO{}, err
	}
	dto.PhotoURL = s.Store.URL(dto.PhotoKey)
	dto.Hydro = map[string]any{"status": dto.HydroStatus}
	if withHydro {
		dto.Hydro = s.loadHydro(ctx, id, dto.HydroStatus)
	}
	return dto, nil
}

func (s *Server) loadHydro(ctx context.Context, catchID, status string) map[string]any {
	out := map[string]any{"status": status}
	var (
		boundAt      time.Time
		bindErr      int
		pHPa, pD3    float64
		trend        string
		airT, wDir   float64
		wLabel       string
		wMS          float64
		beau         int
		aspect       string
		tideH, tideP float64
		tideW, moon  string
		illum, score float64
		frenzy       bool
		contrib, ser []byte
	)
	err := s.DB.QueryRow(ctx, `
		SELECT bound_at, bind_error_sec, pressure_hpa, pressure_delta_3h, pressure_trend,
		       air_temp_c, wind_dir_deg, wind_dir_label, wind_speed_ms, beaufort, shore_aspect,
		       tide_height_m, tide_phase_pct, tide_window, moon_phase, moon_illum_pct,
		       bite_score, frenzy, contributions, series
		FROM hydro_snapshots WHERE catch_id = $1`, catchID).Scan(
		&boundAt, &bindErr, &pHPa, &pD3, &trend,
		&airT, &wDir, &wLabel, &wMS, &beau, &aspect,
		&tideH, &tideP, &tideW, &moon, &illum,
		&score, &frenzy, &contrib, &ser)
	if err != nil {
		return out
	}
	var contributions any = []any{}
	if len(contrib) > 0 {
		_ = json.Unmarshal(contrib, &contributions)
	}
	var series any = map[string]any{}
	if len(ser) > 0 {
		_ = json.Unmarshal(ser, &series)
	}
	out["bound_at"] = timeutil.FormatRFC3339(boundAt)
	out["bind_error_sec"] = bindErr
	out["pressure_hpa"] = pHPa
	out["pressure_delta_3h"] = pD3
	out["pressure_trend"] = trend
	out["air_temp_c"] = airT
	out["wind_dir_deg"] = wDir
	out["wind_dir_label"] = wLabel
	out["wind_speed_ms"] = wMS
	out["beaufort"] = beau
	out["shore_aspect"] = aspect
	out["tide_height_m"] = tideH
	out["tide_phase_pct"] = tideP
	out["tide_window"] = tideW
	out["moon_phase"] = moon
	out["moon_illum_pct"] = illum
	out["bite_score"] = score
	out["frenzy"] = frenzy
	out["contributions"] = contributions
	out["series"] = series
	return out
}

func scanCatch(rows interface {
	Scan(dest ...any) error
}) (CatchDTO, error) {
	var (
		dto     CatchDTO
		caught  time.Time
		created time.Time
	)
	err := rows.Scan(
		&dto.ID, &dto.UserID, &dto.SpotID, &caught, &dto.Timezone, &dto.LocalTime, &dto.Species, &dto.LengthCM, &dto.WeightKG,
		&dto.LureType, &dto.LureWeightG, &dto.LureColor, &dto.Retrieve, &dto.Layer, &dto.WaterDepthM, &dto.WaterColor,
		&dto.Turbidity, &dto.WaterTempC, &dto.Current, &dto.Released, &dto.Note, &dto.PhotoKey, &dto.HydroStatus, &created)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return CatchDTO{}, ErrNotFound
		}
		return CatchDTO{}, err
	}
	dto.CaughtAt = timeutil.FormatRFC3339(caught)
	dto.CreatedAt = timeutil.FormatRFC3339(created)
	if dto.Hydro == nil {
		dto.Hydro = map[string]any{"status": dto.HydroStatus}
	}
	return dto, nil
}
