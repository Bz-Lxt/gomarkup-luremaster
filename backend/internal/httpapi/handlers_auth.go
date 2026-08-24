package httpapi

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"luremaster/internal/auth"
	"luremaster/internal/timeutil"
)

type registerReq struct {
	Email    string `json:"email"`
	Username string `json:"username"`
	Password string `json:"password"`
	Nickname string `json:"nickname"`
}

type loginReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type refreshReq struct {
	RefreshToken string `json:"refresh_token"`
}

type tokenUserData struct {
	AccessToken  string  `json:"access_token"`
	RefreshToken string  `json:"refresh_token"`
	User         UserDTO `json:"user"`
}

type patchMeReq struct {
	Nickname  *string `json:"nickname"`
	AvatarURL *string `json:"avatar_url"`
	HomeWater *string `json:"home_water"`
	LurePref  *string `json:"lure_pref"`
}

func (s *Server) Register(c *gin.Context) {
	var req registerReq
	if err := c.ShouldBindJSON(&req); err != nil {
		Abort(c, Validation("invalid json"))
		return
	}
	email := strings.ToLower(strings.TrimSpace(req.Email))
	username := strings.TrimSpace(req.Username)
	nickname := strings.TrimSpace(req.Nickname)
	if email == "" || !strings.Contains(email, "@") {
		Abort(c, Validation("email required"))
		return
	}
	if len(username) < 2 {
		Abort(c, Validation("username required"))
		return
	}
	if len(req.Password) < 8 {
		Abort(c, Validation("password too short"))
		return
	}
	if nickname == "" {
		nickname = username
	}
	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		Abort(c, ErrInternal)
		return
	}
	now := timeutil.NowUTC()
	id := uuid.NewString()
	_, err = s.DB.Exec(c.Request.Context(), `
		INSERT INTO users (id, email, username, password_hash, nickname, avatar_url, home_water, lure_pref, credit_score, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, '', '', '', 100, $6, $6)`,
		id, email, username, hash, nickname, now)
	if err != nil {
		if isUnique(err) {
			Abort(c, ErrConflictEmail)
			return
		}
		Abort(c, err)
		return
	}
	u, err := s.loadUser(c.Request.Context(), id)
	if err != nil {
		Abort(c, err)
		return
	}
	access, refresh, err := s.Auth.IssueTokens(id)
	if err != nil {
		Abort(c, ErrInternal)
		return
	}
	Created(c, tokenUserData{AccessToken: access, RefreshToken: refresh, User: u})
}

func (s *Server) Login(c *gin.Context) {
	var req loginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		Abort(c, Validation("invalid json"))
		return
	}
	email := strings.ToLower(strings.TrimSpace(req.Email))
	var id, hash string
	err := s.DB.QueryRow(c.Request.Context(), `SELECT id, password_hash FROM users WHERE email = $1`, email).Scan(&id, &hash)
	if err != nil || !auth.CheckPassword(hash, req.Password) {
		Abort(c, ErrInvalidCreds)
		return
	}
	u, err := s.loadUser(c.Request.Context(), id)
	if err != nil {
		Abort(c, err)
		return
	}
	access, refresh, err := s.Auth.IssueTokens(id)
	if err != nil {
		Abort(c, ErrInternal)
		return
	}
	OK(c, tokenUserData{AccessToken: access, RefreshToken: refresh, User: u})
}

func (s *Server) Refresh(c *gin.Context) {
	var req refreshReq
	if err := c.ShouldBindJSON(&req); err != nil {
		Abort(c, Validation("invalid json"))
		return
	}
	claims, err := s.Auth.ParseRefresh(req.RefreshToken)
	if err != nil {
		Abort(c, ErrUnauthorized)
		return
	}
	u, err := s.loadUser(c.Request.Context(), claims.UserID)
	if err != nil {
		Abort(c, ErrUnauthorized)
		return
	}
	access, refresh, err := s.Auth.IssueTokens(claims.UserID)
	if err != nil {
		Abort(c, ErrInternal)
		return
	}
	OK(c, tokenUserData{AccessToken: access, RefreshToken: refresh, User: u})
}

func (s *Server) GetMe(c *gin.Context) {
	u, err := s.loadUser(c.Request.Context(), currentUserID(c))
	if err != nil {
		Abort(c, err)
		return
	}
	OK(c, u)
}

func (s *Server) PatchMe(c *gin.Context) {
	var req patchMeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		Abort(c, Validation("invalid json"))
		return
	}
	uid := currentUserID(c)
	ctx := c.Request.Context()
	u, err := s.loadUser(ctx, uid)
	if err != nil {
		Abort(c, err)
		return
	}
	if req.Nickname != nil {
		u.Nickname = strings.TrimSpace(*req.Nickname)
	}
	if req.AvatarURL != nil {
		u.AvatarURL = strings.TrimSpace(*req.AvatarURL)
	}
	if req.HomeWater != nil {
		u.HomeWater = strings.TrimSpace(*req.HomeWater)
	}
	if req.LurePref != nil {
		u.LurePref = strings.TrimSpace(*req.LurePref)
	}
	_, err = s.DB.Exec(ctx, `
		UPDATE users SET nickname = $1, avatar_url = $2, home_water = $3, lure_pref = $4, updated_at = $5
		WHERE id = $6`, u.Nickname, u.AvatarURL, u.HomeWater, u.LurePref, timeutil.NowUTC(), uid)
	if err != nil {
		Abort(c, err)
		return
	}
	u, err = s.loadUser(ctx, uid)
	if err != nil {
		Abort(c, err)
		return
	}
	OK(c, u)
}
