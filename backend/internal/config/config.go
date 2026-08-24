package config

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Env            string
	HTTPAddr       string
	DatabaseURL    string
	RedisAddr      string
	JWTSecret      string
	HydroProvider  string
	TideProvider   string
	StorageDriver  string
	S3Endpoint     string
	S3AccessKey    string
	S3SecretKey    string
	S3Bucket       string
	S3UsePathStyle bool
	CORSOrigins    []string
	LogLevel       string
	UploadDir      string
}

func Load() Config {
	return Config{
		Env:            env("APP_ENV", "local"),
		HTTPAddr:       env("HTTP_ADDR", ":8080"),
		DatabaseURL:    env("DATABASE_URL", "postgres://lure:lure_dev_2026@127.0.0.1:29683/luremaster?sslmode=disable"),
		RedisAddr:      env("REDIS_ADDR", "127.0.0.1:29684"),
		JWTSecret:      env("JWT_SECRET", "luremaster-dev-jwt-secret-change-me"),
		HydroProvider:  strings.ToLower(env("HYDRO_PROVIDER", "mock")),
		TideProvider:   strings.ToLower(env("TIDE_PROVIDER", "harmonic")),
		StorageDriver:  strings.ToLower(env("STORAGE_DRIVER", "local")),
		S3Endpoint:     env("S3_ENDPOINT", "http://127.0.0.1:29685"),
		S3AccessKey:    env("S3_ACCESS_KEY", "lureminio"),
		S3SecretKey:    env("S3_SECRET_KEY", "lureminio_dev_2026"),
		S3Bucket:       env("S3_BUCKET", "lure-catches"),
		S3UsePathStyle: envBool("S3_USE_PATH_STYLE", true),
		CORSOrigins:    split(env("CORS_ORIGINS", "http://localhost:29681")),
		LogLevel:       env("LOG_LEVEL", "info"),
		UploadDir:      env("UPLOAD_DIR", "/tmp/lure-uploads"),
	}
}

func env(k, def string) string {
	if v := strings.TrimSpace(os.Getenv(k)); v != "" {
		return v
	}
	return def
}

func envBool(k string, def bool) bool {
	v := strings.TrimSpace(os.Getenv(k))
	if v == "" {
		return def
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return def
	}
	return b
}

func split(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
