package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jbutlerdev/claude-code-server/internal/api"
	"github.com/jbutlerdev/claude-code-server/internal/config"
	"github.com/jbutlerdev/claude-code-server/internal/database"
	"github.com/jbutlerdev/claude-code-server/internal/project"
	"github.com/jbutlerdev/claude-code-server/internal/session"
	"github.com/jbutlerdev/claude-code-server/internal/settings"
	_ "github.com/lib/pq"
)

func main() {
	cfg := config.Load()

	if err := run(cfg); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func run(cfg *config.Config) error {
	db, err := sql.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	db.SetMaxOpenConns(cfg.DatabaseMaxConnections)
	db.SetMaxIdleConns(cfg.DatabaseMaxIdleConnections)

	if err := db.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	if err := database.Migrate(cfg.DatabaseURL); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	// Initialize services
	repo := database.NewRepository(db)
	settingsService := settings.NewService(cfg, repo)
	projectService := project.NewService(repo, settingsService)

	log.Println("Syncing existing projects...")
	if err := projectService.SyncAllProjects(context.Background()); err != nil {
		log.Printf("Warning: failed to sync projects: %v", err)
	}

	sessionManager := session.NewManager(cfg, db, settingsService)
	if err := sessionManager.Start(); err != nil {
		return fmt.Errorf("failed to start session manager: %w", err)
	}
	defer sessionManager.Stop()

	router := api.NewRouter(cfg, db, sessionManager)

	srv := &http.Server{
		Addr:           fmt.Sprintf("%s:%d", cfg.ServerHost, cfg.ServerPort),
		Handler:        router,
		ReadTimeout:    cfg.ServerReadTimeout,
		WriteTimeout:   cfg.ServerWriteTimeout,
		MaxHeaderBytes: 1 << 20,
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Printf("Server starting on %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	<-done
	log.Println("Server shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		return fmt.Errorf("server shutdown failed: %w", err)
	}

	log.Println("Server stopped gracefully")
	return nil
}
