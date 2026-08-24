package auth

import (
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"luremaster/internal/timeutil"
)

const (
	AccessTTL  = 2 * time.Hour
	RefreshTTL = 14 * 24 * time.Hour
	useAccess  = "access"
	useRefresh = "refresh"
)

type Claims struct {
	UserID string `json:"user_id"`
	Use    string `json:"use"`
	jwt.RegisteredClaims
}

type Service struct {
	secret []byte
}

func New(secret string) *Service {
	return &Service{secret: []byte(secret)}
}

func HashPassword(password string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func CheckPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

func (s *Service) IssueTokens(userID string) (access, refresh string, err error) {
	access, err = s.sign(userID, useAccess, AccessTTL)
	if err != nil {
		return "", "", err
	}
	refresh, err = s.sign(userID, useRefresh, RefreshTTL)
	if err != nil {
		return "", "", err
	}
	return access, refresh, nil
}

func (s *Service) ParseAccess(token string) (*Claims, error) {
	return s.parse(token, useAccess)
}

func (s *Service) ParseRefresh(token string) (*Claims, error) {
	return s.parse(token, useRefresh)
}

func (s *Service) sign(userID, use string, ttl time.Duration) (string, error) {
	now := timeutil.NowUTC()
	claims := Claims{
		UserID: userID,
		Use:    use,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
			NotBefore: jwt.NewNumericDate(now),
		},
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return t.SignedString(s.secret)
}

func (s *Service) parse(token, wantUse string) (*Claims, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, fmt.Errorf("empty token")
	}
	parsed, err := jwt.ParseWithClaims(token, &Claims{}, func(t *jwt.Token) (any, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected alg")
		}
		return s.secret, nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := parsed.Claims.(*Claims)
	if !ok || !parsed.Valid {
		return nil, fmt.Errorf("invalid claims")
	}
	if claims.UserID == "" || claims.Use != wantUse {
		return nil, fmt.Errorf("token use mismatch")
	}
	return claims, nil
}
