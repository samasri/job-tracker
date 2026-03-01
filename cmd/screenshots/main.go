package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/chromedp/chromedp"

	"jobtracker/internal/app"
	httpserver "jobtracker/internal/http"
	"jobtracker/internal/infra/filestore"
	"jobtracker/internal/infra/sqlite"

	// Register migrations
	_ "jobtracker/internal/infra/sqlite/migrations"
)

func main() {
	chromiumPath := os.Getenv("CHROMIUM_PATH")
	if chromiumPath == "" {
		chromiumPath = "chromium"
	}

	outDir := "docs/screenshots"
	if err := os.MkdirAll(outDir, 0755); err != nil {
		log.Fatalf("creating output dir: %v", err)
	}

	tmpDir, err := os.MkdirTemp("", "screenshots-*")
	if err != nil {
		log.Fatalf("creating temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	baseURL, cleanup, err := startServer(tmpDir)
	if err != nil {
		log.Fatalf("starting server: %v", err)
	}
	defer cleanup()

	aliceID, err := seedData(tmpDir)
	if err != nil {
		log.Fatalf("seeding data: %v", err)
	}

	pages := []struct {
		filename string
		path     string
	}{
		{"companies.png", "/companies"},
		{"company-stripe.png", "/companies/stripe"},
		{"role-stripe-swe.png", "/companies/stripe/roles/senior-software-engineer"},
		{"contacts.png", "/contacts"},
		{"contact-alice.png", "/contacts/" + aliceID},
	}

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.ExecPath(chromiumPath),
		chromedp.WindowSize(1280, 900),
	)

	allocCtx, cancelAlloc := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancelAlloc()

	browserCtx, cancelBrowser := chromedp.NewContext(allocCtx)
	defer cancelBrowser()

	for _, p := range pages {
		outPath := filepath.Join(outDir, p.filename)
		if err := screenshot(browserCtx, baseURL+p.path, outPath); err != nil {
			log.Fatalf("screenshot %s: %v", p.filename, err)
		}
		log.Printf("saved %s", outPath)
	}
}

func startServer(tmpDir string) (baseURL string, cleanup func(), err error) {
	dbPath := filepath.Join(tmpDir, "index.sqlite")
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return "", nil, fmt.Errorf("creating db dir: %w", err)
	}
	if err := os.MkdirAll(filepath.Join(tmpDir, "data", "companies"), 0755); err != nil {
		return "", nil, fmt.Errorf("creating data dir: %w", err)
	}

	db, err := sqlite.New(dbPath)
	if err != nil {
		return "", nil, fmt.Errorf("opening db: %w", err)
	}

	if err := db.RunMigrations(); err != nil {
		db.Close()
		return "", nil, fmt.Errorf("running migrations: %w", err)
	}

	companyRepo := sqlite.NewCompanyRepo(db)
	roleRepo := sqlite.NewRoleRepo(db)
	contactRepo := sqlite.NewContactRepo(db)
	contactRoleRepo := sqlite.NewContactRoleRepo(db)
	meetingRepo := sqlite.NewMeetingRepo(db)
	jdRepo := sqlite.NewJobDescriptionRepo(db)
	resumeRepo := sqlite.NewResumeRepo(db)
	artifactRepo := sqlite.NewRoleArtifactRepo(db)

	fs := filestore.New(tmpDir)

	companyService := app.NewCompanyService(companyRepo, roleRepo, fs)
	contactService := app.NewContactService(contactRepo, contactRoleRepo, companyRepo, roleRepo, fs)
	meetingService := app.NewMeetingService(meetingRepo, companyRepo, roleRepo, contactRepo, fs)
	jdService := app.NewJDService(jdRepo, companyRepo, roleRepo, fs)
	resumeService := app.NewResumeService(resumeRepo, companyRepo, roleRepo, fs)
	artifactService := app.NewArtifactService(artifactRepo, companyRepo, roleRepo, fs)
	exportService := app.NewExportService(sqlite.NewExportQuerier(db), tmpDir)

	handlers := httpserver.NewHandlers(companyService, contactService, meetingService, jdService, resumeService, artifactService, exportService)
	server := httpserver.NewServer(handlers)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		db.Close()
		return "", nil, fmt.Errorf("listening: %w", err)
	}

	go http.Serve(ln, server) //nolint:errcheck

	baseURL = "http://" + ln.Addr().String()
	cleanup = func() {
		ln.Close()
		db.Close()
	}
	return baseURL, cleanup, nil
}

func seedData(tmpDir string) (aliceID string, err error) {
	dbPath := filepath.Join(tmpDir, "index.sqlite")
	db, err := sqlite.New(dbPath)
	if err != nil {
		return "", fmt.Errorf("opening db for seeding: %w", err)
	}
	defer db.Close()

	companyRepo := sqlite.NewCompanyRepo(db)
	roleRepo := sqlite.NewRoleRepo(db)
	contactRepo := sqlite.NewContactRepo(db)
	contactRoleRepo := sqlite.NewContactRoleRepo(db)
	meetingRepo := sqlite.NewMeetingRepo(db)
	fs := filestore.New(tmpDir)

	companyService := app.NewCompanyService(companyRepo, roleRepo, fs)
	contactService := app.NewContactService(contactRepo, contactRoleRepo, companyRepo, roleRepo, fs)
	meetingService := app.NewMeetingService(meetingRepo, companyRepo, roleRepo, contactRepo, fs)

	ctx := context.Background()

	// Stripe roles
	if _, err := companyService.CreateCompany(ctx, app.CreateCompanyInput{Slug: "stripe", Name: "Stripe"}); err != nil {
		return "", fmt.Errorf("creating stripe: %w", err)
	}
	stripeSWE, err := companyService.CreateRole(ctx, app.CreateRoleInput{
		CompanySlug: "stripe", Slug: "senior-software-engineer", Title: "Senior Software Engineer",
	})
	if err != nil {
		return "", fmt.Errorf("creating stripe swe: %w", err)
	}
	if _, err := companyService.CreateRole(ctx, app.CreateRoleInput{
		CompanySlug: "stripe", Slug: "staff-engineer", Title: "Staff Engineer",
	}); err != nil {
		return "", fmt.Errorf("creating stripe staff: %w", err)
	}
	if err := companyService.UpdateRoleStatus(ctx, app.UpdateRoleStatusInput{
		CompanySlug: "stripe", RoleSlug: "senior-software-engineer", Status: "hr_interview",
	}); err != nil {
		return "", fmt.Errorf("updating stripe swe status: %w", err)
	}
	if err := companyService.UpdateRoleStatus(ctx, app.UpdateRoleStatusInput{
		CompanySlug: "stripe", RoleSlug: "staff-engineer", Status: "recruiter_reached_out",
	}); err != nil {
		return "", fmt.Errorf("updating stripe staff status: %w", err)
	}

	// Shopify roles
	if _, err := companyService.CreateCompany(ctx, app.CreateCompanyInput{Slug: "shopify", Name: "Shopify"}); err != nil {
		return "", fmt.Errorf("creating shopify: %w", err)
	}
	shopifyPlatform, err := companyService.CreateRole(ctx, app.CreateRoleInput{
		CompanySlug: "shopify", Slug: "platform-engineer", Title: "Platform Engineer",
	})
	if err != nil {
		return "", fmt.Errorf("creating shopify platform: %w", err)
	}
	if _, err := companyService.CreateRole(ctx, app.CreateRoleInput{
		CompanySlug: "shopify", Slug: "software-engineer", Title: "Software Engineer",
	}); err != nil {
		return "", fmt.Errorf("creating shopify swe: %w", err)
	}
	if err := companyService.UpdateRoleStatus(ctx, app.UpdateRoleStatusInput{
		CompanySlug: "shopify", RoleSlug: "platform-engineer", Status: "offer",
	}); err != nil {
		return "", fmt.Errorf("updating shopify platform status: %w", err)
	}
	if err := companyService.UpdateRoleStatus(ctx, app.UpdateRoleStatusInput{
		CompanySlug: "shopify", RoleSlug: "software-engineer", Status: "rejected",
	}); err != nil {
		return "", fmt.Errorf("updating shopify swe status: %w", err)
	}

	// Linear roles
	if _, err := companyService.CreateCompany(ctx, app.CreateCompanyInput{Slug: "linear", Name: "Linear"}); err != nil {
		return "", fmt.Errorf("creating linear: %w", err)
	}
	if _, err := companyService.CreateRole(ctx, app.CreateRoleInput{
		CompanySlug: "linear", Slug: "backend-engineer", Title: "Backend Engineer",
	}); err != nil {
		return "", fmt.Errorf("creating linear backend: %w", err)
	}
	if err := companyService.UpdateRoleStatus(ctx, app.UpdateRoleStatusInput{
		CompanySlug: "linear", RoleSlug: "backend-engineer", Status: "in_progress",
	}); err != nil {
		return "", fmt.Errorf("updating linear backend status: %w", err)
	}

	// Contacts
	alice, err := contactService.CreateContact(ctx, app.CreateContactInput{
		Name:        "Alice Chen",
		Email:       "alice.chen@stripe.com",
		Org:         "Stripe",
		LinkedInURL: "https://www.linkedin.com/in/fake-alice",
	})
	if err != nil {
		return "", fmt.Errorf("creating alice: %w", err)
	}

	bob, err := contactService.CreateContact(ctx, app.CreateContactInput{
		Name:  "Bob Martinez",
		Email: "bob.martinez@shopify.com",
		Org:   "Shopify",
	})
	if err != nil {
		return "", fmt.Errorf("creating bob: %w", err)
	}

	// Meetings
	if _, err := meetingService.CreateRoleMeeting(ctx, app.CreateRoleMeetingInput{
		CompanySlug: "stripe",
		RoleSlug:    "senior-software-engineer",
		OccurredAt:  "2025-11-10",
		Title:       "Phone Screen",
	}); err != nil {
		return "", fmt.Errorf("creating stripe swe meeting: %w", err)
	}

	if _, err := meetingService.CreateContactMeeting(ctx, app.CreateContactMeetingInput{
		ContactID:  alice.ID,
		OccurredAt: "2025-11-08",
		Title:      "Initial Outreach",
	}); err != nil {
		return "", fmt.Errorf("creating alice meeting: %w", err)
	}

	// Links
	if err := contactService.LinkRole(ctx, app.LinkContactRoleInput{
		ContactID:   alice.ID,
		CompanySlug: "stripe",
		RoleSlug:    "senior-software-engineer",
	}); err != nil {
		return "", fmt.Errorf("linking alice to stripe swe: %w", err)
	}
	if err := contactService.LinkRole(ctx, app.LinkContactRoleInput{
		ContactID:   bob.ID,
		CompanySlug: "shopify",
		RoleSlug:    "platform-engineer",
	}); err != nil {
		return "", fmt.Errorf("linking bob to shopify platform: %w", err)
	}

	_ = stripeSWE
	_ = shopifyPlatform

	return alice.ID, nil
}

func screenshot(ctx context.Context, url, outPath string) error {
	var buf []byte
	if err := chromedp.Run(ctx,
		chromedp.Navigate(url),
		chromedp.Sleep(500*time.Millisecond),
		chromedp.FullScreenshot(&buf, 90),
	); err != nil {
		return fmt.Errorf("chromedp run: %w", err)
	}
	return os.WriteFile(outPath, buf, 0644)
}
