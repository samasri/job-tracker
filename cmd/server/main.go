package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"jobtracker/internal/app"
	"jobtracker/internal/config"
	httpserver "jobtracker/internal/http"
	"jobtracker/internal/infra/filestore"
	"jobtracker/internal/infra/sqlite"

	// Register migrations
	_ "jobtracker/internal/infra/sqlite/migrations"
)

func main() {
	cfg := config.LoadConfig()

	log.Printf("Starting jobtracker server...")
	log.Printf("Repo root: %s", cfg.RepoRoot)
	log.Printf("DB path: %s", cfg.DBPath)
	log.Printf("Bind address: %s", cfg.Addr)

	// Ensure db directory exists
	if err := os.MkdirAll(filepath.Dir(cfg.DBPath), 0755); err != nil {
		log.Fatalf("Failed to create db directory: %v", err)
	}

	// Ensure data directory exists
	if err := os.MkdirAll(filepath.Join(cfg.RepoRoot, "data", "companies"), 0755); err != nil {
		log.Fatalf("Failed to create data directory: %v", err)
	}

	// Initialize database
	db, err := sqlite.New(cfg.DBPath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Run migrations
	if err := db.RunMigrations(); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}
	log.Printf("Database migrations complete")

	// Create repositories
	companyRepo := sqlite.NewCompanyRepo(db)
	roleRepo := sqlite.NewRoleRepo(db)
	contactRepo := sqlite.NewContactRepo(db)
	threadRepo := sqlite.NewThreadRepo(db)
	meetingRepo := sqlite.NewMeetingRepo(db)
	meetingV2Repo := sqlite.NewMeetingV2Repo(db)
	jdRepo := sqlite.NewJobDescriptionRepo(db)
	resumeRepo := sqlite.NewResumeRepo(db)
	artifactRepo := sqlite.NewRoleArtifactRepo(db)

	// Create filestore
	fs := filestore.New(cfg.RepoRoot)

	// Create services
	companyService := app.NewCompanyService(companyRepo, roleRepo, meetingRepo, fs)
	contactService := app.NewContactService(contactRepo)
	threadService := app.NewThreadService(threadRepo, meetingRepo, companyRepo, roleRepo, contactRepo)

	// Backfill thread codes for existing threads
	if err := threadService.BackfillThreadCodes(context.Background()); err != nil {
		log.Printf("Warning: failed to backfill thread codes: %v", err)
	}

	meetingService := app.NewMeetingService(meetingRepo, companyRepo, fs)
	meetingV2Service := app.NewMeetingV2Service(meetingV2Repo, companyRepo, roleRepo, threadRepo, fs)
	jdService := app.NewJDService(jdRepo, companyRepo, roleRepo, fs)
	resumeService := app.NewResumeService(resumeRepo, companyRepo, roleRepo, fs)
	artifactService := app.NewArtifactService(artifactRepo, companyRepo, roleRepo, fs)
	exportService := app.NewExportService(db, cfg.RepoRoot)

	// Create handlers
	handlers := httpserver.NewHandlers(companyService, contactService, threadService, meetingService, meetingV2Service, jdService, resumeService, artifactService, exportService)

	// Create HTTP server
	server := httpserver.NewServer(handlers)

	log.Printf("Server listening on http://%s", cfg.Addr)
	if err := http.ListenAndServe(cfg.Addr, server); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
