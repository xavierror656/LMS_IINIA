package main

import (
	"aulaquest/internal/config"
	"aulaquest/internal/database"
	"aulaquest/internal/handlers"
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	cfg, e := config.Load()
	if e != nil {
		slog.Error("invalid configuration", "error", e)
		os.Exit(1)
	}
	db, e := database.Open(cfg.DatabaseURL)
	if e != nil {
		slog.Error("database unavailable")
		os.Exit(1)
	}
	pool, _ := db.DB()
	defer pool.Close()
	app, hub := handlers.New(db, cfg)
	done := make(chan error, 1)
	go func() { done <- app.Listen(cfg.Address) }()
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	select {
	case <-ctx.Done():
		hub.Close()
		if e = app.ShutdownWithTimeout(5 * time.Second); e != nil {
			slog.Error("shutdown failed")
		}
	case e = <-done:
		hub.Close()
		if e != nil {
			slog.Error("listen failed", "error", e)
			os.Exit(1)
		}
	}
}
