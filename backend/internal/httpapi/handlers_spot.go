package httpapi

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"luremaster/internal/logger"
	"luremaster/internal/model"
	"luremaster/internal/timeutil"
)

type createSpotReq struct {
	Name         string     `json:"name"`
	WaterType    string     `json:"water_type"`
	Structure    string     `json:"structure"`
	Visibility   string     `json:"visibility"`
	Lat          float64    `json:"lat"`
	Lon          float64    `json:"lon"`
	ShoreBearing float64    `json:"shore_bearing"`
	Tidal        bool       `json:"tidal"`
	ClubID       string     `json:"club_id"`
	Note         string     `json:"note"`
	Depths       []DepthDTO `json:"depths"`
}

func (s *Server) ListSpots(c *gin.Context) {
	ctx := c.Request.Context()
	viewer := currentUserID(c)
	structure := model.NormalizeEnum(c.Query("structure"))
	visibility := model.NormalizeEnum(c.Query("visibility"))
	q := strings.TrimSpace(c.Query("q"))
	sql, args := s.spotsWhere(viewer, structure, visibility, q, 0, 0, 0)
	sql += " ORDER BY created_at DESC LIMIT 200"
	rows, err := s.DB.Query(ctx, sql, args...)
	if err != nil {
		Abort(c, err)
		return
	}
	defer rows.Close()
	out := []SpotDTO{}
	for rows.Next() {
		var row spotRow
		if err := scanSpot(rows, &row); err != nil {
			Abort(c, err)
			return
		}
		out = append(out, s.toSpotDTO(ctx, viewer, row, s.loadDepths(ctx, row.ID)))
	}
	OK(c, out)
}

func (s *Server) NearbySpots(c *gin.Context) {
	ctx := c.Request.Context()
	viewer := currentUserID(c)
	lat, err1 := strconv.ParseFloat(c.Query("lat"), 64)
	lon, err2 := strconv.ParseFloat(c.Query("lon"), 64)
	if err1 != nil || err2 != nil {
		Abort(c, Validation("lat and lon required"))
		return
	}
	km := 10.0
	if raw := c.Query("km"); raw != "" {
		v, err := strconv.ParseFloat(raw, 64)
		if err != nil || v <= 0 || v > 200 {
			Abort(c, Validation("invalid km"))
			return
		}
		km = v
	}
	if !s.allowScan(c, viewer) {
		Abort(c, ErrRateLimited)
		return
	}
	sql, args := s.spotsWhere(viewer, model.NormalizeEnum(c.Query("structure")), model.NormalizeEnum(c.Query("visibility")), strings.TrimSpace(c.Query("q")), lon, lat, km*1000)
	sql += " ORDER BY created_at DESC LIMIT 200"
	rows, err := s.DB.Query(ctx, sql, args...)
	if err != nil {
		Abort(c, err)
		return
	}
	defer rows.Close()
	out := []SpotDTO{}
	for rows.Next() {
		var row spotRow
		if err := scanSpot(rows, &row); err != nil {
			Abort(c, err)
			return
		}
		out = append(out, s.toSpotDTO(ctx, viewer, row, s.loadDepths(ctx, row.ID)))
	}
	s.audit(ctx, viewer, "spot.nearby", fmt.Sprintf("%.4f,%.4f,%.1f", lat, lon, km), c.ClientIP())
	OK(c, out)
}

func (s *Server) GetSpot(c *gin.Context) {
	ctx := c.Request.Context()
	viewer := currentUserID(c)
	id := c.Param("id")
	if !parseUUID(id) {
		Abort(c, ErrNotFound)
		return
	}
	row, err := s.loadSpot(ctx, id)
	if err != nil {
		Abort(c, err)
		return
	}
	clubID := ""
	if row.ClubID != nil {
		clubID = *row.ClubID
	}
	sameClub := s.isClubMember(ctx, clubID, viewer)
	isFriend := s.areFriends(ctx, viewer, row.OwnerID)
	depths := s.loadDepths(ctx, row.ID)
	if !canListSpot(row.Visibility, row.OwnerID, viewer, sameClub, isFriend) {
		depths = []DepthDTO{}
	}
	dto := s.toSpotDTO(ctx, viewer, row, depths)
	if dto.Fuzzed {
		dto.Note = ""
		dto.Depths = []DepthDTO{}
		s.audit(ctx, viewer, "spot.view_fuzzed", "spot:"+row.ID, c.ClientIP())
	} else {
		s.audit(ctx, viewer, "spot.view_exact", "spot:"+row.ID, c.ClientIP())
	}
	OK(c, dto)
}

func (s *Server) CreateSpot(c *gin.Context) {
	var req createSpotReq
	if err := c.ShouldBindJSON(&req); err != nil {
		Abort(c, Validation("invalid json"))
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		Abort(c, Validation("name required"))
		return
	}
	wt := model.NormalizeEnum(req.WaterType)
	st := model.NormalizeEnum(req.Structure)
	vis := model.NormalizeEnum(req.Visibility)
	if !model.InSet(wt, model.WaterTypes) {
		Abort(c, Validation("invalid water_type"))
		return
	}
	if !model.InSet(st, model.Structures) {
		Abort(c, Validation("invalid structure"))
		return
	}
	if !model.InSet(vis, model.Visibilities) {
		Abort(c, Validation("invalid visibility"))
		return
	}
	if req.Lat < -90 || req.Lat > 90 || req.Lon < -180 || req.Lon > 180 {
		Abort(c, Validation("invalid coordinates"))
		return
	}
	uid := currentUserID(c)
	var club any
	if strings.TrimSpace(req.ClubID) != "" {
		if !parseUUID(req.ClubID) {
			Abort(c, Validation("invalid club_id"))
			return
		}
		if vis == "CLUB" && !s.isClubMember(c.Request.Context(), req.ClubID, uid) {
			Abort(c, ErrForbidden)
			return
		}
		club = req.ClubID
	} else if vis == "CLUB" {
		Abort(c, Validation("club_id required for CLUB visibility"))
		return
	}
	now := timeutil.NowUTC()
	id := uuid.NewString()
	ctx := c.Request.Context()
	_, err := s.DB.Exec(ctx, `
		INSERT INTO spots (
			id, owner_id, club_id, name, water_type, structure, visibility,
			location, lat, lon, shore_bearing, tidal, note, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7,
			ST_SetSRID(ST_MakePoint($8, $9), 4326)::geography, $10, $11, $12, $13, $14, $15, $16
		)`,
		id, uid, club, name, wt, st, vis,
		req.Lon, req.Lat, req.Lat, req.Lon, req.ShoreBearing, req.Tidal, strings.TrimSpace(req.Note), now, now)
	if err != nil {
		Abort(c, err)
		return
	}
	for _, d := range req.Depths {
		_, err := s.DB.Exec(ctx, `
			INSERT INTO spot_depths (id, spot_id, offset_m, depth_m, sampled_at)
			VALUES ($1, $2, $3, $4, $5)`, uuid.NewString(), id, d.OffsetM, d.DepthM, now)
		if err != nil {
			Abort(c, err)
			return
		}
	}
	row, err := s.loadSpot(ctx, id)
	if err != nil {
		Abort(c, err)
		return
	}
	Created(c, s.toSpotDTO(ctx, uid, row, s.loadDepths(ctx, id)))
}

func (s *Server) spotsWhere(viewer, structure, visibility, q string, lon, lat, meters float64) (string, []any) {
	args := []any{}
	var b strings.Builder
	b.WriteString(`SELECT id, owner_id, club_id, name, water_type, structure, visibility, lat, lon, shore_bearing, tidal, note, created_at FROM spots WHERE `)
	if viewer == "" {
		b.WriteString(`visibility = 'PUBLIC'`)
	} else {
		args = append(args, viewer)
		b.WriteString(`(
			visibility = 'PUBLIC'
			OR owner_id = $1
			OR (visibility = 'CLUB' AND club_id IN (SELECT club_id FROM club_members WHERE user_id = $1 AND status = 'APPROVED'))
			OR (visibility = 'FRIENDS' AND EXISTS (
				SELECT 1 FROM friendships f
				WHERE f.status = 'ACCEPTED'
				  AND ((f.user_id = $1 AND f.friend_id = spots.owner_id) OR (f.friend_id = $1 AND f.user_id = spots.owner_id))
			))
		)`)
	}
	if structure != "" {
		args = append(args, structure)
		b.WriteString(fmt.Sprintf(` AND structure = $%d`, len(args)))
	}
	if visibility != "" {
		args = append(args, visibility)
		b.WriteString(fmt.Sprintf(` AND visibility = $%d`, len(args)))
	}
	if q != "" {
		args = append(args, "%"+q+"%")
		n := len(args)
		b.WriteString(fmt.Sprintf(` AND (name ILIKE $%d OR note ILIKE $%d)`, n, n))
	}
	if meters > 0 {
		args = append(args, lon, lat, meters)
		n := len(args)
		b.WriteString(fmt.Sprintf(` AND ST_DWithin(location, ST_SetSRID(ST_MakePoint($%d, $%d), 4326)::geography, $%d)`, n-2, n-1, n))
	}
	return b.String(), args
}

func scanSpot(rows interface {
	Scan(dest ...any) error
}, row *spotRow) error {
	return rows.Scan(&row.ID, &row.OwnerID, &row.ClubID, &row.Name, &row.WaterType, &row.Structure, &row.Visibility,
		&row.Lat, &row.Lon, &row.ShoreBearing, &row.Tidal, &row.Note, &row.CreatedAt)
}

func (s *Server) allowScan(c *gin.Context, viewer string) bool {
	if s.Redis == nil {
		return true
	}
	key := "scan:" + viewer
	if key == "scan:" {
		key = "scan:ip:" + c.ClientIP()
	}
	n, err := s.Redis.IncrWindow(c.Request.Context(), key, time.Minute)
	if err != nil {
		logger.From().Warn("rate limit redis", "err", err)
		return true
	}
	return n <= 60
}
