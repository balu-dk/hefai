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

	"github.com/balu-dk/hefai/backend/internal/ai"
	"github.com/balu-dk/hefai/backend/internal/auth"
	"github.com/balu-dk/hefai/backend/internal/config"
	"github.com/balu-dk/hefai/backend/internal/database"
	"github.com/balu-dk/hefai/backend/internal/filestore"
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
	rooms := repository.NewRooms(pool)
	suppliers := repository.NewSuppliers(pool)
	budget := repository.NewBudget(pool)
	materials := repository.NewMaterials(pool)
	documents := repository.NewDocuments(pool)
	caseFiles := repository.NewCaseFiles(pool)
	drawings := repository.NewDrawings(pool)
	compliance := repository.NewCompliance(pool)
	sources := repository.NewSources(pool)
	generated := repository.NewGenerated(pool)
	structural := repository.NewStructural(pool)
	packages := repository.NewPackages(pool)
	rules := repository.NewComplianceRules(pool)

	files, err := filestore.NewDisk(cfg.FileStoreDir)
	if err != nil {
		return err
	}

	// LLM-provider: sæt LLM_BASE_URL (+ LLM_API_KEY, LLM_MODEL) for at
	// aktivere assistent og AI-projektstart. Uden konfiguration degraderer
	// begge pænt (kildeuddrag hhv. skabelon-blueprint).
	var llm ai.Provider = ai.Unconfigured{}
	if cfg.LLMBaseURL != "" {
		llm = ai.NewOpenAICompatible(cfg.LLMBaseURL, cfg.LLMAPIKey, cfg.LLMModel)
		slog.Info("llm provider configured", "model", cfg.LLMModel, "base", cfg.LLMBaseURL)
	}

	svc := httpapi.Services{
		Auth:      service.NewAuth(users, tokens),
		Projects:  service.NewProjects(projects, users),
		Phases:    service.NewPhases(phases, projects),
		Tasks:     service.NewTasks(tasks, phases, projects),
		Rooms:     service.NewRooms(rooms, projects),
		Suppliers: service.NewSuppliers(suppliers, projects),
		Budget:    service.NewBudget(budget, phases, projects),
		Materials: service.NewMaterials(materials, suppliers, projects),
		Documents: service.NewDocuments(documents, files, projects),
		CaseFiles: service.NewCaseFiles(caseFiles, projects),
		Drawings:  service.NewDrawings(drawings, caseFiles, projects),
		Checklist: service.NewCompliance(compliance, caseFiles, sources, projects),
		Sources:   service.NewSources(sources, projects),
		Assistant: service.NewAssistant(sources, caseFiles, llm, projects),
		Generator: service.NewGenerator(generated, caseFiles, drawings, compliance,
			projects, documents, files, projects),
		Structural: service.NewStructural(structural, projects),
		Packages:   service.NewPackages(packages, structural, projects, documents, files, projects),
		Blueprints: service.NewBlueprints(llm, cfg.AIDocsDir, projects, phases, rooms, tasks,
			budget, materials, caseFiles, projects),
		Ortho:      service.NewOrtho(cfg.OrthoWMSURL, cfg.OrthoLayer, cfg.OrthoToken, projects),
		DrawAssist: service.NewDrawAssist(llm, cfg.AIDocsDir, drawings, projects),
		Rules:      service.NewRules(rules, llm, cfg.AIDocsDir, projects, drawings, sources, projects),
		Lookup:     service.NewLookup(cfg.DAWABaseURL, cfg.PlandataWFSURL),
		PlanImport: service.NewPlanImport(files, documents, sources, projects),
	}
	svc.Advisor = service.NewAdvisor(llm, cfg.AIDocsDir, phases, tasks, budget, materials,
		caseFiles, drawings, sources, svc.Rules, projects)

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
