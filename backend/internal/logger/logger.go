package logger

import (
	"log/slog"
	"os"
	"strings"
)

var L *slog.Logger

func Init(level string, env string) {
	lv := slog.LevelInfo
	switch strings.ToLower(level) {
	case "debug":
		if strings.EqualFold(env, "production") {
			lv = slog.LevelInfo
		} else {
			lv = slog.LevelDebug
		}
	case "warn", "warning":
		lv = slog.LevelWarn
	case "error":
		lv = slog.LevelError
	}
	L = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: lv}))
	slog.SetDefault(L)
}

func From() *slog.Logger {
	if L == nil {
		Init("info", "docker")
	}
	return L
}
