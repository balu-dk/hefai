package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/balu-dk/hefai/backend/internal/auth"
	"github.com/balu-dk/hefai/backend/internal/config"
	"github.com/balu-dk/hefai/backend/internal/database"
	"github.com/balu-dk/hefai/backend/internal/httpapi"
	"github.com/balu-dk/hefai/backend/internal/repository"
	"github.com/balu-dk/hefai/backend/internal/service"
)

func main() {
	if err := run(); err != nil {
		slog.Error("fatal", "err", err)
		os.Exit(1)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := config.FromEnv()
	if err != nil {
		return err
	}

	pool, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	if err := database.Migrate(ctx, pool, cfg.MigrationsDir); err != nil {
		return err
	}
	slog.Info("migrations applied", "dir", cfg.MigrationsDir)

	tokens := auth.NewTokenIssuer(cfg.JWTSecret, cfg.TokenTTL)

	users := repository.NewUsers(pool)
	projects := repository.NewProjects(pool)
	phases := repository.NewPhases(pool)
	tasks := repository.NewTasks(pool)

	svc := httpapi.Services{
		Auth:     service.NewAuth(users, tokens),
		Projects: service.NewProjects(projects, users),
		Phases:   service.NewPhases(phases, projects),
		Tasks:    service.NewTasks(tasks, phases, projects),
	}

	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           httpapi.New(svc, tokens, cfg.CORSOrigin),
		ReadHeaderTimeout: 10 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		slog.Info("listening", "addr", cfg.HTTPAddr)
		if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		slog.Info("shutting down")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return server.Shutdown(shutdownCtx)
	}
}
