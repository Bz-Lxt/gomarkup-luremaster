package httpapi

import (
	"context"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"luremaster/internal/lure"
	"luremaster/internal/model"
	"luremaster/internal/timeutil"
)

type createClubReq struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type ClubDTO struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	OwnerID     string `json:"owner_id"`
	Description string `json:"description"`
	Members     int    `json:"members"`
	CreatedAt   string `json:"created_at"`
}

type recommendReq struct {
	Species       string   `json:"species"`
	SpotID        string   `json:"spot_id"`
	PressureTrend string   `json:"pressure_trend"`
	TideWindow    string   `json:"tide_window"`
	WaterTempC    *float64 `json:"water_temp_c"`
}

type statsDTO struct {
	TotalCatches  int          `json:"total_catches"`
	ReleasedCount int          `json:"released_count"`
	ReleaseRate   float64      `json:"release_rate"`
	MaxLengthCM   float64      `json:"max_length_cm"`
	MaxSpecies    string       `json:"max_species"`
	StreakDays    int          `json:"streak_days"`
	TopLures      []topLureDTO `json:"top_lures"`
	TopSpots      []topSpotDTO `json:"top_spots"`
}

type topLureDTO struct {
	LureType string `json:"lure_type"`
	Count    int    `json:"count"`
}

type topSpotDTO struct {
	SpotID string `json:"spot_id"`
	Name   string `json:"name"`
	Count  int    `json:"count"`
}

func (s *Server) Healthz(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (s *Server) Readyz(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()
	if err := s.DB.Ping(ctx); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"ok": false, "error": gin.H{"code": "NOT_READY", "message": "db"}})
		return
	}
	if s.Redis != nil {
		if err := s.Redis.Ping(ctx); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"ok": false, "error": gin.H{"code": "NOT_READY", "message": "redis"}})
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (s *Server) CreateClub(c *gin.Context) {
	var req createClubReq
	if err := c.ShouldBindJSON(&req); err != nil {
		Abort(c, Validation("invalid json"))
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		Abort(c, Validation("name required"))
		return
	}
	uid := currentUserID(c)
	now := timeutil.NowUTC()
	id := uuid.NewString()
	ctx := c.Request.Context()
	_, err := s.DB.Exec(ctx, `
		INSERT INTO clubs (id, name, owner_id, description, created_at)
		VALUES ($1, $2, $3, $4, $5)`, id, name, uid, strings.TrimSpace(req.Description), now)
	if err != nil {
		Abort(c, err)
		return
	}
	_, err = s.DB.Exec(ctx, `
		INSERT INTO club_members (club_id, user_id, role, status, joined_at)
		VALUES ($1, $2, 'OWNER', 'APPROVED', $3)`, id, uid, now)
	if err != nil {
		Abort(c, err)
		return
	}
	Created(c, ClubDTO{
		ID: id, Name: name, OwnerID: uid, Description: strings.TrimSpace(req.Description),
		Members: 1, CreatedAt: timeutil.FormatRFC3339(now),
	})
}

func (s *Server) JoinClub(c *gin.Context) {
	id := c.Param("id")
	if !parseUUID(id) {
		Abort(c, ErrNotFound)
		return
	}
	ctx := c.Request.Context()
	var exists int
	if err := s.DB.QueryRow(ctx, `SELECT COUNT(*) FROM clubs WHERE id = $1`, id).Scan(&exists); err != nil || exists == 0 {
		Abort(c, ErrNotFound)
		return
	}
	uid := currentUserID(c)
	now := timeutil.NowUTC()
	_, err := s.DB.Exec(ctx, `
		INSERT INTO club_members (club_id, user_id, role, status, joined_at)
		VALUES ($1, $2, 'MEMBER', 'APPROVED', $3)
		ON CONFLICT (club_id, user_id) DO NOTHING`, id, uid, now)
	if err != nil {
		Abort(c, err)
		return
	}
	dto, err := s.loadClub(ctx, id)
	if err != nil {
		Abort(c, err)
		return
	}
	OK(c, dto)
}

func (s *Server) ListClubs(c *gin.Context) {
	ctx := c.Request.Context()
	rows, err := s.DB.Query(ctx, `
		SELECT c.id, c.name, c.owner_id, c.description, c.created_at,
		       (SELECT COUNT(*) FROM club_members m WHERE m.club_id = c.id AND m.status = 'APPROVED')
		FROM clubs c
		ORDER BY c.created_at DESC`)
	if err != nil {
		Abort(c, err)
		return
	}
	defer rows.Close()
	out := []ClubDTO{}
	for rows.Next() {
		var d ClubDTO
		var created time.Time
		if err := rows.Scan(&d.ID, &d.Name, &d.OwnerID, &d.Description, &created, &d.Members); err != nil {
			Abort(c, err)
			return
		}
		d.CreatedAt = timeutil.FormatRFC3339(created)
		out = append(out, d)
	}
	OK(c, out)
}

func (s *Server) RecommendLures(c *gin.Context) {
	var req recommendReq
	if err := c.ShouldBindJSON(&req); err != nil {
		Abort(c, Validation("invalid json"))
		return
	}
	species := model.NormalizeEnum(req.Species)
	if species == "" {
		Abort(c, Validation("species required"))
		return
	}
	hint := lure.HydroHint{
		PressureTrend: model.NormalizeEnum(req.PressureTrend),
		TideWindow:    model.NormalizeEnum(req.TideWindow),
	}
	if req.WaterTempC != nil {
		hint.WaterTempC = *req.WaterTempC
	}
	ctx := c.Request.Context()
	if hint.PressureTrend == "" && parseUUID(req.SpotID) {
		var trend, window string
		var temp *float64
		_ = s.DB.QueryRow(ctx, `
			SELECT h.pressure_trend, h.tide_window, c.water_temp_c
			FROM catches c
			JOIN hydro_snapshots h ON h.catch_id = c.id
			WHERE c.spot_id = $1
			ORDER BY c.caught_at DESC
			LIMIT 1`, req.SpotID).Scan(&trend, &window, &temp)
		if hint.PressureTrend == "" {
			hint.PressureTrend = trend
		}
		if hint.TideWindow == "" {
			hint.TideWindow = window
		}
		if req.WaterTempC == nil && temp != nil {
			hint.WaterTempC = *temp
		}
	}
	rows, err := s.DB.Query(ctx, `
		SELECT lure_type, lure_color, layer, retrieve FROM catches WHERE user_id = $1`, currentUserID(c))
	hist := []lure.HistoryHit{}
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var h lure.HistoryHit
			if err := rows.Scan(&h.LureType, &h.Color, &h.Layer, &h.Retrieve); err != nil {
				continue
			}
			h.Caught = true
			hist = append(hist, h)
		}
	}
	out := lure.Recommend(species, hint, hist)
	if out == nil {
		out = []lure.Advice{}
	}
	OK(c, out)
}

func (s *Server) MyStats(c *gin.Context) {
	ctx := c.Request.Context()
	uid := currentUserID(c)
	rows, err := s.DB.Query(ctx, `
		SELECT species, length_cm, released, lure_type, spot_id, caught_at
		FROM catches WHERE user_id = $1`, uid)
	if err != nil {
		Abort(c, err)
		return
	}
	defer rows.Close()
	var (
		total, released int
		maxLen          float64
		maxSp           string
		lureN           = map[string]int{}
		spotN           = map[string]int{}
		times           []time.Time
	)
	for rows.Next() {
		var species, lureType, spotID string
		var length float64
		var rel bool
		var at time.Time
		if err := rows.Scan(&species, &length, &rel, &lureType, &spotID, &at); err != nil {
			Abort(c, err)
			return
		}
		total++
		if rel {
			released++
		}
		if length > maxLen {
			maxLen = length
			maxSp = species
		}
		lureN[lureType]++
		spotN[spotID]++
		times = append(times, at)
	}
	rate := 0.0
	if total > 0 {
		rate = float64(released) / float64(total)
	}
	topLures := []topLureDTO{}
	for k, n := range lureN {
		topLures = append(topLures, topLureDTO{LureType: k, Count: n})
	}
	sort.Slice(topLures, func(i, j int) bool { return topLures[i].Count > topLures[j].Count })
	if len(topLures) > 3 {
		topLures = topLures[:3]
	}
	topSpots := []topSpotDTO{}
	for id, n := range spotN {
		name := id
		_ = s.DB.QueryRow(ctx, `SELECT name FROM spots WHERE id = $1`, id).Scan(&name)
		topSpots = append(topSpots, topSpotDTO{SpotID: id, Name: name, Count: n})
	}
	sort.Slice(topSpots, func(i, j int) bool { return topSpots[i].Count > topSpots[j].Count })
	if len(topSpots) > 3 {
		topSpots = topSpots[:3]
	}
	OK(c, statsDTO{
		TotalCatches:  total,
		ReleasedCount: released,
		ReleaseRate:   rate,
		MaxLengthCM:   maxLen,
		MaxSpecies:    maxSp,
		StreakDays:    streakDays(times),
		TopLures:      topLures,
		TopSpots:      topSpots,
	})
}

func (s *Server) loadClub(ctx context.Context, id string) (ClubDTO, error) {
	var d ClubDTO
	var created time.Time
	err := s.DB.QueryRow(ctx, `
		SELECT c.id, c.name, c.owner_id, c.description, c.created_at,
		       (SELECT COUNT(*) FROM club_members m WHERE m.club_id = c.id AND m.status = 'APPROVED')
		FROM clubs c WHERE c.id = $1`, id).Scan(&d.ID, &d.Name, &d.OwnerID, &d.Description, &created, &d.Members)
	if err != nil {
		return ClubDTO{}, ErrNotFound
	}
	d.CreatedAt = timeutil.FormatRFC3339(created)
	return d, nil
}

func streakDays(times []time.Time) int {
	if len(times) == 0 {
		return 0
	}
	seen := map[string]time.Time{}
	for _, t := range times {
		y, m, d := timeutil.CivilDate(t, timeutil.Beijing)
		key := time.Date(y, m, d, 0, 0, 0, 0, timeutil.Beijing).Format("2006-01-02")
		seen[key] = time.Date(y, m, d, 0, 0, 0, 0, timeutil.Beijing)
	}
	dates := make([]time.Time, 0, len(seen))
	for _, v := range seen {
		dates = append(dates, v)
	}
	sort.Slice(dates, func(i, j int) bool { return dates[i].After(dates[j]) })
	n := 1
	for i := 1; i < len(dates); i++ {
		if dates[i-1].Sub(dates[i]) == 24*time.Hour {
			n++
			continue
		}
		break
	}
	return n
}
