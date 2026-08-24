package httpapi

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"luremaster/internal/auth"
	"luremaster/internal/booking"
	"luremaster/internal/config"
	"luremaster/internal/logger"
	"luremaster/internal/redisx"
	"luremaster/internal/spot"
	"luremaster/internal/storage"
	"luremaster/internal/timeutil"
)

type Server struct {
	Cfg    config.Config
	DB     *pgxpool.Pool
	Redis  *redisx.Client
	Store  storage.Storage
	Auth   *auth.Service
	Locker *booking.Locker
}

func NewRouter(s *Server) *gin.Engine {
	if strings.EqualFold(s.Cfg.Env, "production") {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.New()
	r.MaxMultipartMemory = 8 << 20
	r.Use(RequestID(), AccessLog(), CORS(s.Cfg.CORSOrigins), gin.Recovery())

	r.GET("/healthz", s.Healthz)
	r.GET("/readyz", s.Readyz)

	v1 := r.Group("/api/v1")
	v1.POST("/auth/register", s.Register)
	v1.POST("/auth/login", s.Login)
	v1.POST("/auth/refresh", s.Refresh)
	v1.GET("/files/*key", s.GetFile)

	pub := v1.Group("")
	pub.Use(s.AuthOptional())
	pub.GET("/spots", s.ListSpots)
	pub.GET("/spots/nearby", s.NearbySpots)
	pub.GET("/spots/:id", s.GetSpot)

	authed := v1.Group("")
	authed.Use(s.AuthRequired())
	authed.GET("/me", s.GetMe)
	authed.PATCH("/me", s.PatchMe)
	authed.GET("/me/stats", s.MyStats)
	authed.POST("/spots", s.CreateSpot)
	authed.GET("/catches", s.ListCatches)
	authed.POST("/catches", s.CreateCatch)
	authed.GET("/catches/:id", s.GetCatch)
	authed.POST("/catches/:id/photo", s.UploadCatchPhoto)
	authed.POST("/lures/recommend", s.RecommendLures)
	authed.GET("/clubs", s.ListClubs)
	authed.POST("/clubs", s.CreateClub)
	authed.POST("/clubs/:id/join", s.JoinClub)
	authed.GET("/activities", s.ListActivities)
	authed.POST("/activities", s.CreateActivity)
	authed.POST("/activities/:id/open", s.OpenActivity)
	authed.POST("/activities/:id/slots/:sid/claim", s.ClaimSlot)
	authed.POST("/activities/:id/slots/:sid/confirm", s.ConfirmSlot)
	authed.POST("/activities/:id/slots/:sid/release", s.ReleaseSlot)
	authed.POST("/activities/:id/checkin", s.Checkin)

	return r
}

type UserDTO struct {
	ID          string `json:"id"`
	Email       string `json:"email"`
	Username    string `json:"username"`
	Nickname    string `json:"nickname"`
	AvatarURL   string `json:"avatar_url"`
	HomeWater   string `json:"home_water"`
	LurePref    string `json:"lure_pref"`
	CreditScore int    `json:"credit_score"`
	CreatedAt   string `json:"created_at"`
}

type DepthDTO struct {
	OffsetM float64 `json:"offset_m"`
	DepthM  float64 `json:"depth_m"`
}

type SpotDTO struct {
	ID           string     `json:"id"`
	OwnerID      string     `json:"owner_id"`
	ClubID       *string    `json:"club_id"`
	Name         string     `json:"name"`
	WaterType    string     `json:"water_type"`
	Structure    string     `json:"structure"`
	Visibility   string     `json:"visibility"`
	Lat          float64    `json:"lat"`
	Lon          float64    `json:"lon"`
	ShoreBearing float64    `json:"shore_bearing"`
	Tidal        bool       `json:"tidal"`
	Note         string     `json:"note"`
	Fuzzed       bool       `json:"fuzzed"`
	Depths       []DepthDTO `json:"depths"`
	CreatedAt    string     `json:"created_at"`
}

type spotRow struct {
	ID           string
	OwnerID      string
	ClubID       *string
	Name         string
	WaterType    string
	Structure    string
	Visibility   string
	Lat          float64
	Lon          float64
	ShoreBearing float64
	Tidal        bool
	Note         string
	CreatedAt    time.Time
}

func (s *Server) loadUser(ctx context.Context, id string) (UserDTO, error) {
	var u UserDTO
	var created time.Time
	err := s.DB.QueryRow(ctx, `
		SELECT id, email, username, nickname, avatar_url, home_water, lure_pref, credit_score, created_at
		FROM users WHERE id = $1`, id).Scan(
		&u.ID, &u.Email, &u.Username, &u.Nickname, &u.AvatarURL, &u.HomeWater, &u.LurePref, &u.CreditScore, &created)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return UserDTO{}, ErrNotFound
		}
		return UserDTO{}, err
	}
	u.CreatedAt = timeutil.FormatRFC3339(created)
	return u, nil
}

func (s *Server) isClubMember(ctx context.Context, clubID, userID string) bool {
	if clubID == "" || userID == "" {
		return false
	}
	var n int
	_ = s.DB.QueryRow(ctx, `
		SELECT COUNT(*) FROM club_members
		WHERE club_id = $1 AND user_id = $2 AND status = 'APPROVED'`, clubID, userID).Scan(&n)
	return n > 0
}

func (s *Server) areFriends(ctx context.Context, a, b string) bool {
	if a == "" || b == "" || a == b {
		return false
	}
	var n int
	_ = s.DB.QueryRow(ctx, `
		SELECT COUNT(*) FROM friendships
		WHERE status = 'ACCEPTED'
		  AND ((user_id = $1 AND friend_id = $2) OR (user_id = $2 AND friend_id = $1))`, a, b).Scan(&n)
	return n > 0
}

func (s *Server) toSpotDTO(ctx context.Context, viewer string, row spotRow, depths []DepthDTO) SpotDTO {
	clubID := ""
	if row.ClubID != nil {
		clubID = *row.ClubID
	}
	sameClub := clubID != "" && s.isClubMember(ctx, clubID, viewer)
	isFriend := s.areFriends(ctx, viewer, row.OwnerID)
	exact := spot.CanSeeExact(row.Visibility, row.OwnerID, viewer, clubID, sameClub, isFriend)
	lat, lon := row.Lat, row.Lon
	fuzzed := false
	if !exact {
		lat, lon = spot.Fuzz(row.Lat, row.Lon, row.ID)
		fuzzed = true
	}
	if depths == nil {
		depths = []DepthDTO{}
	}
	return SpotDTO{
		ID: row.ID, OwnerID: row.OwnerID, ClubID: row.ClubID, Name: row.Name,
		WaterType: row.WaterType, Structure: row.Structure, Visibility: row.Visibility,
		Lat: lat, Lon: lon, ShoreBearing: row.ShoreBearing, Tidal: row.Tidal,
		Note: row.Note, Fuzzed: fuzzed, Depths: depths, CreatedAt: timeutil.FormatRFC3339(row.CreatedAt),
	}
}

func (s *Server) loadDepths(ctx context.Context, spotID string) []DepthDTO {
	rows, err := s.DB.Query(ctx, `
		SELECT offset_m, depth_m FROM spot_depths WHERE spot_id = $1 ORDER BY offset_m`, spotID)
	if err != nil {
		return []DepthDTO{}
	}
	defer rows.Close()
	out := []DepthDTO{}
	for rows.Next() {
		var d DepthDTO
		if err := rows.Scan(&d.OffsetM, &d.DepthM); err != nil {
			continue
		}
		out = append(out, d)
	}
	return out
}

func (s *Server) loadSpot(ctx context.Context, id string) (spotRow, error) {
	var row spotRow
	err := s.DB.QueryRow(ctx, `
		SELECT id, owner_id, club_id, name, water_type, structure, visibility, lat, lon, shore_bearing, tidal, note, created_at
		FROM spots WHERE id = $1`, id).Scan(
		&row.ID, &row.OwnerID, &row.ClubID, &row.Name, &row.WaterType, &row.Structure, &row.Visibility,
		&row.Lat, &row.Lon, &row.ShoreBearing, &row.Tidal, &row.Note, &row.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return spotRow{}, ErrNotFound
		}
		return spotRow{}, err
	}
	return row, nil
}

func canListSpot(visibility, ownerID, viewer string, sameClub, isFriend bool) bool {
	if viewer != "" && viewer == ownerID {
		return true
	}
	switch visibility {
	case "PUBLIC":
		return true
	case "CLUB":
		return sameClub
	case "FRIENDS":
		return isFriend
	default:
		return false
	}
}

func (s *Server) audit(ctx context.Context, userID, action, target, ip string) {
	_, err := s.DB.Exec(ctx, `
		INSERT INTO audit_logs (id, user_id, action, target, detail, ip, created_at)
		VALUES ($1, $2, $3, $4, '{}', $5, $6)`,
		uuid.NewString(), nilIfEmpty(userID), action, target, ip, timeutil.NowUTC())
	if err != nil {
		logger.From().Warn("audit", "err", err)
	}
}

func nilIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func isUnique(err error) bool {
	var pe *pgconn.PgError
	return errors.As(err, &pe) && pe.Code == "23505"
}

func mapBooking(err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, booking.ErrTaken):
		return ErrSlotTaken
	case errors.Is(err, booking.ErrNotHolder):
		return ErrForbidden
	case errors.Is(err, booking.ErrExpired):
		return ErrSlotState
	case errors.Is(err, booking.ErrBadState):
		return ErrNotFound
	}
	var te booking.TransitionError
	if errors.As(err, &te) {
		if te.Code == "SLOT_STATE" {
			return ErrSlotState
		}
		return ErrActivityState
	}
	return err
}

func (s *Server) withTx(ctx context.Context, fn func(context.Context, pgx.Tx) error) error {
	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := fn(booking.WithTx(ctx, tx), tx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Server) GetFile(c *gin.Context) {
	key := strings.TrimPrefix(c.Param("key"), "/")
	if key == "" {
		Abort(c, ErrNotFound)
		return
	}
	rc, ct, err := s.Store.Get(c.Request.Context(), key)
	if err != nil {
		Abort(c, ErrNotFound)
		return
	}
	defer rc.Close()
	c.Header("Content-Type", ct)
	c.Status(http.StatusOK)
	_, _ = io.Copy(c.Writer, rc)
}

func parseUUID(s string) bool {
	_, err := uuid.Parse(strings.TrimSpace(s))
	return err == nil
}
