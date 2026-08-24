package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"luremaster/internal/auth"
	"luremaster/internal/booking"
	"luremaster/internal/config"
	"luremaster/internal/db"
	"luremaster/internal/httpapi"
	"luremaster/internal/hydro"
	"luremaster/internal/logger"
	"luremaster/internal/redisx"
	"luremaster/internal/seed"
	"luremaster/internal/storage"
	"luremaster/internal/timeutil"
)

func main() {
	cfg := config.Load()
	logger.Init(cfg.LogLevel, cfg.Env)
	log := logger.From()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	var database *db.DB
	var err error
	for i := 0; i < 20; i++ {
		database, err = db.Open(ctx, cfg.DatabaseURL)
		if err == nil {
			break
		}
		log.Warn("db open retry", "attempt", i+1, "err", err)
		select {
		case <-ctx.Done():
			os.Exit(1)
		case <-time.After(2 * time.Second):
		}
	}
	if err != nil {
		log.Error("db open", "err", err)
		os.Exit(1)
	}
	defer database.Close()

	if err := database.Migrate(ctx, db.ResolveMigrationsDir("")); err != nil {
		log.Error("migrate", "err", err)
		os.Exit(1)
	}

	rdb := redisx.Open(cfg.RedisAddr)
	defer func() { _ = rdb.Close() }()

	store, err := storage.New(cfg)
	if err != nil {
		log.Error("storage", "err", err)
		os.Exit(1)
	}

	var weather hydro.WeatherProvider = hydro.MockWeather{}
	if cfg.HydroProvider == "openmeteo" {
		weather = hydro.OpenMeteo{}
	}
	engine := hydro.NewEngine(weather, hydro.HarmonicTide{})
	worker := hydro.NewWorker(database.Pool, engine)
	if err := worker.Recovery(ctx); err != nil {
		log.Error("hydro recovery", "err", err)
	}

	if err := seed.Run(ctx, database.Pool); err != nil {
		log.Error("seed", "err", err)
		os.Exit(1)
	}

	slots := booking.NewPostgresStore(database.Pool)
	locker := &booking.Locker{
		Store: slots,
		Redis: rdb,
		TTL:   15 * time.Minute,
		Now:   timeutil.NowUTC,
	}

	go background(ctx, worker, slots)

	srv := &httpapi.Server{
		Cfg:    cfg,
		DB:     database.Pool,
		Redis:  rdb,
		Store:  store,
		Auth:   auth.New(cfg.JWTSecret),
		Locker: locker,
	}
	engineHTTP := httpapi.NewRouter(srv)
	httpSrv := &http.Server{Addr: cfg.HTTPAddr, Handler: engineHTTP}

	go func() {
		log.Info("listen", "addr", cfg.HTTPAddr, "hydro", cfg.HydroProvider)
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("http", "err", err)
			stop()
		}
	}()

	<-ctx.Done()
	shut, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	_ = httpSrv.Shutdown(shut)
}

func background(ctx context.Context, worker *hydro.Worker, slots *booking.PostgresStore) {
	tick := time.NewTicker(400 * time.Millisecond)
	expire := time.NewTicker(30 * time.Second)
	defer tick.Stop()
	defer expire.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
			if err := worker.Tick(ctx); err != nil {
				logger.From().Error("hydro tick", "err", err)
			}
		case <-expire.C:
			if _, err := slots.ExpireDue(ctx, timeutil.NowUTC()); err != nil {
				logger.From().Error("expire slots", "err", err)
			}
			if err := worker.Recovery(ctx); err != nil {
				logger.From().Error("hydro recovery", "err", err)
			}
		}
	}
}
