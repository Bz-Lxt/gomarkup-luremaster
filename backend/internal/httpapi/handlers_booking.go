package httpapi

import (
	"context"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"luremaster/internal/booking"
	"luremaster/internal/model"
	"luremaster/internal/spot"
	"luremaster/internal/timeutil"
)

type createActivityReq struct {
	Title      string   `json:"title"`
	Kind       string   `json:"kind"`
	SpotID     string   `json:"spot_id"`
	StartsAt   string   `json:"starts_at"`
	EndsAt     string   `json:"ends_at"`
	MeetLat    float64  `json:"meet_lat"`
	MeetLon    float64  `json:"meet_lon"`
	MeetRadius float64  `json:"meet_radius_m"`
	FeeAmount  float64  `json:"fee_amount"`
	FeeNote    string   `json:"fee_note"`
	Slots      []string `json:"slots"`
}

type checkinReq struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

type SlotDTO struct {
	ID            string `json:"id"`
	ActivityID    string `json:"activity_id"`
	Label         string `json:"label"`
	Status        string `json:"status"`
	HolderID      string `json:"holder_id"`
	LockExpiresAt string `json:"lock_expires_at"`
	Version       int    `json:"version"`
}

type ActivityDTO struct {
	ID         string    `json:"id"`
	HostID     string    `json:"host_id"`
	ClubID     *string   `json:"club_id"`
	SpotID     string    `json:"spot_id"`
	Title      string    `json:"title"`
	Kind       string    `json:"kind"`
	Status     string    `json:"status"`
	StartsAt   string    `json:"starts_at"`
	EndsAt     string    `json:"ends_at"`
	MeetLat    float64   `json:"meet_lat"`
	MeetLon    float64   `json:"meet_lon"`
	MeetRadius float64   `json:"meet_radius_m"`
	FeeAmount  float64   `json:"fee_amount"`
	FeeNote    string    `json:"fee_note"`
	Slots      []SlotDTO `json:"slots"`
	CreatedAt  string    `json:"created_at"`
}

var activityKinds = []string{"CLUB_CHARTER", "WILD_MEETUP", "HOT_SLOT"}

func (s *Server) CreateActivity(c *gin.Context) {
	var req createActivityReq
	if err := c.ShouldBindJSON(&req); err != nil {
		Abort(c, Validation("invalid json"))
		return
	}
	title := strings.TrimSpace(req.Title)
	kind := model.NormalizeEnum(req.Kind)
	if title == "" || !parseUUID(req.SpotID) {
		Abort(c, Validation("title and spot_id required"))
		return
	}
	if !model.InSet(kind, activityKinds) {
		Abort(c, Validation("invalid kind"))
		return
	}
	starts, err := time.Parse(time.RFC3339, strings.TrimSpace(req.StartsAt))
	if err != nil {
		Abort(c, Validation("invalid starts_at"))
		return
	}
	ends, err := time.Parse(time.RFC3339, strings.TrimSpace(req.EndsAt))
	if err != nil {
		Abort(c, Validation("invalid ends_at"))
		return
	}
	if !ends.After(starts) {
		Abort(c, Validation("ends_at must be after starts_at"))
		return
	}
	ctx := c.Request.Context()
	row, err := s.loadSpot(ctx, req.SpotID)
	if err != nil {
		Abort(c, err)
		return
	}
	if row.Visibility != "PUBLIC" && row.Visibility != "CLUB" {
		Abort(c, ErrForbidden)
		return
	}
	uid := currentUserID(c)
	if row.Visibility == "CLUB" {
		clubID := ""
		if row.ClubID != nil {
			clubID = *row.ClubID
		}
		if !s.isClubMember(ctx, clubID, uid) && uid != row.OwnerID {
			Abort(c, ErrForbidden)
			return
		}
	}
	radius := req.MeetRadius
	if radius <= 0 {
		radius = 300
	}
	now := timeutil.NowUTC()
	id := uuid.NewString()
	labels := req.Slots
	if labels == nil {
		labels = []string{}
	}
	err = s.withTx(ctx, func(txctx context.Context, tx pgx.Tx) error {
		_, err := tx.Exec(txctx, `
			INSERT INTO activities (
				id, host_id, club_id, spot_id, title, kind, status, starts_at, ends_at,
				meet_lat, meet_lon, meet_radius_m, fee_amount, fee_note, created_at
			) VALUES (
				$1,$2,$3,$4,$5,$6,$7,$8,$9,
				$10,$11,$12,$13,$14,$15
			)`,
			id, uid, row.ClubID, req.SpotID, title, kind, booking.ActDraft, starts.UTC(), ends.UTC(),
			req.MeetLat, req.MeetLon, radius, req.FeeAmount, strings.TrimSpace(req.FeeNote), now)
		if err != nil {
			return err
		}
		for _, label := range labels {
			label = strings.TrimSpace(label)
			if label == "" {
				continue
			}
			if _, err := tx.Exec(txctx, `
				INSERT INTO slots (id, activity_id, label, status, version)
				VALUES ($1, $2, $3, $4, 0)`, uuid.NewString(), id, label, booking.SlotOpen); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		Abort(c, err)
		return
	}
	dto, err := s.loadActivity(ctx, id)
	if err != nil {
		Abort(c, err)
		return
	}
	Created(c, dto)
}

func (s *Server) OpenActivity(c *gin.Context) {
	ctx := c.Request.Context()
	id := c.Param("id")
	if !parseUUID(id) {
		Abort(c, ErrNotFound)
		return
	}
	dto, err := s.loadActivity(ctx, id)
	if err != nil {
		Abort(c, err)
		return
	}
	if dto.HostID != currentUserID(c) {
		Abort(c, ErrForbidden)
		return
	}
	if !booking.CanActivity(dto.Status, booking.ActOpen) {
		Abort(c, ErrActivityState)
		return
	}
	_, err = s.DB.Exec(ctx, `UPDATE activities SET status = $1 WHERE id = $2`, booking.ActOpen, id)
	if err != nil {
		Abort(c, err)
		return
	}
	dto, err = s.loadActivity(ctx, id)
	if err != nil {
		Abort(c, err)
		return
	}
	OK(c, dto)
}

func (s *Server) ListActivities(c *gin.Context) {
	ctx := c.Request.Context()
	rows, err := s.DB.Query(ctx, `
		SELECT id FROM activities
		WHERE status IN ($1, $2)
		ORDER BY starts_at ASC LIMIT 200`, booking.ActOpen, booking.ActDraft)
	if err != nil {
		Abort(c, err)
		return
	}
	defer rows.Close()
	ids := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			Abort(c, err)
			return
		}
		ids = append(ids, id)
	}
	out := []ActivityDTO{}
	for _, id := range ids {
		dto, err := s.loadActivity(ctx, id)
		if err != nil {
			continue
		}
		if dto.Status == booking.ActDraft && dto.HostID != currentUserID(c) {
			continue
		}
		out = append(out, dto)
	}
	OK(c, out)
}

func (s *Server) ClaimSlot(c *gin.Context) {
	s.slotOp(c, "claim")
}

func (s *Server) ConfirmSlot(c *gin.Context) {
	s.slotOp(c, "confirm")
}

func (s *Server) ReleaseSlot(c *gin.Context) {
	s.slotOp(c, "release")
}

func (s *Server) slotOp(c *gin.Context, op string) {
	actID := c.Param("id")
	sid := c.Param("sid")
	if !parseUUID(actID) || !parseUUID(sid) {
		Abort(c, ErrNotFound)
		return
	}
	ctx := c.Request.Context()
	act, err := s.loadActivity(ctx, actID)
	if err != nil {
		Abort(c, err)
		return
	}
	if op == "claim" && act.Status != booking.ActOpen {
		Abort(c, ErrActivityState)
		return
	}
	uid := currentUserID(c)
	var rec booking.SlotRecord
	err = s.withTx(ctx, func(txctx context.Context, _ pgx.Tx) error {
		var e error
		switch op {
		case "claim":
			rec, e = s.Locker.Claim(txctx, sid, uid)
		case "confirm":
			rec, e = s.Locker.Confirm(txctx, sid, uid)
		case "release":
			e = s.Locker.Release(txctx, sid, uid)
			if e == nil {
				rec, e = s.Locker.Store.GetForUpdate(txctx, sid)
			}
		default:
			e = ErrNotFound
		}
		return e
	})
	if err != nil {
		Abort(c, mapBooking(err))
		return
	}
	OK(c, slotDTO(rec))
}

func (s *Server) Checkin(c *gin.Context) {
	var req checkinReq
	if err := c.ShouldBindJSON(&req); err != nil {
		Abort(c, Validation("invalid json"))
		return
	}
	id := c.Param("id")
	if !parseUUID(id) {
		Abort(c, ErrNotFound)
		return
	}
	ctx := c.Request.Context()
	act, err := s.loadActivity(ctx, id)
	if err != nil {
		Abort(c, err)
		return
	}
	if act.Status != booking.ActOpen {
		Abort(c, ErrActivityState)
		return
	}
	distKm := spot.HaversineKm(req.Lat, req.Lon, act.MeetLat, act.MeetLon)
	distM := distKm * 1000
	if distM > act.MeetRadius {
		Abort(c, ErrCheckinFar)
		return
	}
	uid := currentUserID(c)
	now := timeutil.NowUTC()
	_, err = s.DB.Exec(ctx, `
		INSERT INTO checkins (id, activity_id, user_id, lat, lon, distance_m, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (activity_id, user_id) DO NOTHING`,
		uuid.NewString(), id, uid, req.Lat, req.Lon, distM, now)
	if err != nil {
		Abort(c, err)
		return
	}
	var slotID string
	_ = s.DB.QueryRow(ctx, `
		SELECT id FROM slots
		WHERE activity_id = $1 AND holder_id = $2 AND status IN ($3, $4)
		LIMIT 1`, id, uid, booking.SlotConfirmed, booking.SlotCheckedIn).Scan(&slotID)
	if slotID != "" {
		err = s.withTx(ctx, func(txctx context.Context, _ pgx.Tx) error {
			_, e := s.Locker.CheckIn(txctx, slotID, uid)
			return e
		})
		if err != nil {
			Abort(c, mapBooking(err))
			return
		}
	}
	OK(c, gin.H{
		"activity_id": id,
		"user_id":     uid,
		"lat":         req.Lat,
		"lon":         req.Lon,
		"distance_m":  distM,
		"created_at":  timeutil.FormatRFC3339(now),
	})
}

func (s *Server) loadActivity(ctx context.Context, id string) (ActivityDTO, error) {
	var (
		dto          ActivityDTO
		starts, ends time.Time
		created      time.Time
	)
	err := s.DB.QueryRow(ctx, `
		SELECT id, host_id, club_id, spot_id, title, kind, status, starts_at, ends_at,
		       meet_lat, meet_lon, meet_radius_m, fee_amount, fee_note, created_at
		FROM activities WHERE id = $1`, id).Scan(
		&dto.ID, &dto.HostID, &dto.ClubID, &dto.SpotID, &dto.Title, &dto.Kind, &dto.Status, &starts, &ends,
		&dto.MeetLat, &dto.MeetLon, &dto.MeetRadius, &dto.FeeAmount, &dto.FeeNote, &created)
	if err != nil {
		if err == pgx.ErrNoRows {
			return ActivityDTO{}, ErrNotFound
		}
		return ActivityDTO{}, err
	}
	dto.StartsAt = timeutil.FormatRFC3339(starts)
	dto.EndsAt = timeutil.FormatRFC3339(ends)
	dto.CreatedAt = timeutil.FormatRFC3339(created)
	dto.Slots = []SlotDTO{}
	rows, err := s.DB.Query(ctx, `
		SELECT id, activity_id, label, status, COALESCE(holder_id::text, ''), lock_expires_at, version
		FROM slots WHERE activity_id = $1 ORDER BY label`, id)
	if err != nil {
		return dto, nil
	}
	defer rows.Close()
	for rows.Next() {
		var rec booking.SlotRecord
		var exp *time.Time
		if err := rows.Scan(&rec.ID, &rec.ActivityID, &rec.Label, &rec.Status, &rec.HolderID, &exp, &rec.Version); err != nil {
			continue
		}
		if exp != nil {
			rec.LockExpiresAt = *exp
		}
		dto.Slots = append(dto.Slots, slotDTO(rec))
	}
	return dto, nil
}

func slotDTO(rec booking.SlotRecord) SlotDTO {
	exp := ""
	if !rec.LockExpiresAt.IsZero() {
		exp = timeutil.FormatRFC3339(rec.LockExpiresAt)
	}
	return SlotDTO{
		ID: rec.ID, ActivityID: rec.ActivityID, Label: rec.Label, Status: rec.Status,
		HolderID: rec.HolderID, LockExpiresAt: exp, Version: rec.Version,
	}
}
