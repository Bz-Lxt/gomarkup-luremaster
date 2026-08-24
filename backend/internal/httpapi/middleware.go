package httpapi

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"luremaster/internal/logger"
)

func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := strings.TrimSpace(c.GetHeader("X-Request-ID"))
		if id == "" {
			id = uuid.NewString()
		}
		c.Set("request_id", id)
		c.Header("X-Request-ID", id)
		c.Next()
	}
}

func CORS(origins []string) gin.HandlerFunc {
	allow := make(map[string]struct{}, len(origins))
	for _, o := range origins {
		o = strings.TrimSpace(o)
		if o != "" {
			allow[o] = struct{}{}
		}
	}
	return func(c *gin.Context) {
		origin := strings.TrimSpace(c.GetHeader("Origin"))
		if origin != "" {
			host := c.Request.Host
			same := origin == "http://"+host || origin == "https://"+host
			_, listed := allow[origin]
			if same || listed {
				c.Header("Access-Control-Allow-Origin", origin)
				c.Header("Access-Control-Allow-Credentials", "true")
				c.Header("Vary", "Origin")
				c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Request-ID")
				c.Header("Access-Control-Allow-Methods", "GET, POST, PATCH, PUT, DELETE, OPTIONS")
				c.Header("Access-Control-Max-Age", "86400")
			}
		}
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

func AccessLog() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		logger.From().Info("http",
			"method", c.Request.Method,
			"path", c.FullPath(),
			"uri", c.Request.URL.RequestURI(),
			"status", c.Writer.Status(),
			"latency_ms", time.Since(start).Milliseconds(),
			"request_id", c.GetString("request_id"),
			"ip", c.ClientIP(),
		)
	}
}

func (s *Server) AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		uid, err := s.bearerUser(c)
		if err != nil {
			Abort(c, ErrUnauthorized)
			return
		}
		c.Set("user_id", uid)
		c.Next()
	}
}

func (s *Server) AuthOptional() gin.HandlerFunc {
	return func(c *gin.Context) {
		if uid, err := s.bearerUser(c); err == nil {
			c.Set("user_id", uid)
		}
		c.Next()
	}
}

func (s *Server) bearerUser(c *gin.Context) (string, error) {
	h := c.GetHeader("Authorization")
	if !strings.HasPrefix(strings.ToLower(h), "bearer ") {
		return "", ErrUnauthorized
	}
	raw := strings.TrimSpace(h[7:])
	claims, err := s.Auth.ParseAccess(raw)
	if err != nil {
		return "", ErrUnauthorized
	}
	return claims.UserID, nil
}

func currentUserID(c *gin.Context) string {
	v, _ := c.Get("user_id")
	s, _ := v.(string)
	return s
}
