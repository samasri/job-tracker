package testharness

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"jobtracker/internal/app"
	httpserver "jobtracker/internal/http"
	"jobtracker/internal/infra/filestore"
	"jobtracker/internal/infra/sqlite"

	// Register migrations
	_ "jobtracker/internal/infra/sqlite/migrations"
)

// TestEnv holds the test environment
type TestEnv struct {
	T        *testing.T
	Server   *httptest.Server
	DB       *sqlite.DB
	RepoRoot string
	DBPath   string
	Client   *http.Client

	// Services (for tests that need direct access)
	CompanyService   *app.CompanyService
	ContactService   *app.ContactService
	MeetingService *app.MeetingService
	JDService        *app.JDService
	ResumeService    *app.ResumeService
	ArtifactService  *app.ArtifactService
	ExportService    *app.ExportService
}

// NewTestEnv creates a new test environment with temp dirs and server
func NewTestEnv(t *testing.T) *TestEnv {
	t.Helper()

	// Create temp directory for repo root
	repoRoot, err := os.MkdirTemp("", "jobtracker-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp repo root: %v", err)
	}

	// Create data directories
	if err := os.MkdirAll(filepath.Join(repoRoot, "data", "companies"), 0755); err != nil {
		t.Fatalf("Failed to create data dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(repoRoot, "db"), 0755); err != nil {
		t.Fatalf("Failed to create db dir: %v", err)
	}

	// Create temp SQLite database
	dbPath := filepath.Join(repoRoot, "db", "index.sqlite")

	// Initialize database
	db, err := sqlite.New(dbPath)
	if err != nil {
		os.RemoveAll(repoRoot)
		t.Fatalf("Failed to create database: %v", err)
	}

	// Run migrations
	if err := db.RunMigrations(); err != nil {
		db.Close()
		os.RemoveAll(repoRoot)
		t.Fatalf("Failed to run migrations: %v", err)
	}

	// Create repositories
	companyRepo := sqlite.NewCompanyRepo(db)
	roleRepo := sqlite.NewRoleRepo(db)
	contactRepo := sqlite.NewContactRepo(db)
	contactRoleRepo := sqlite.NewContactRoleRepo(db)
	meetingRepo := sqlite.NewMeetingRepo(db)
	jdRepo := sqlite.NewJobDescriptionRepo(db)
	resumeRepo := sqlite.NewResumeRepo(db)
	artifactRepo := sqlite.NewRoleArtifactRepo(db)

	// Create filestore
	fs := filestore.New(repoRoot)

	// Create services
	companyService := app.NewCompanyService(companyRepo, roleRepo, fs)
	contactService := app.NewContactService(contactRepo, contactRoleRepo, companyRepo, roleRepo, fs)
	meetingService := app.NewMeetingService(meetingRepo, companyRepo, roleRepo, contactRepo, fs)
	jdService := app.NewJDService(jdRepo, companyRepo, roleRepo, fs)
	resumeService := app.NewResumeService(resumeRepo, companyRepo, roleRepo, fs)
	artifactService := app.NewArtifactService(artifactRepo, companyRepo, roleRepo, fs)
	exportService := app.NewExportService(sqlite.NewExportQuerier(db), repoRoot)

	// Create handlers
	handlers := httpserver.NewHandlers(companyService, contactService, meetingService, jdService, resumeService, artifactService, exportService)

	// Create HTTP server
	server := httpserver.NewServer(handlers)
	ts := httptest.NewServer(server)

	env := &TestEnv{
		T:                t,
		Server:           ts,
		DB:               db,
		RepoRoot:         repoRoot,
		DBPath:           dbPath,
		Client:           ts.Client(),
		CompanyService:   companyService,
		ContactService:   contactService,
		MeetingService: meetingService,
		JDService:        jdService,
		ResumeService:    resumeService,
		ArtifactService:  artifactService,
		ExportService:    exportService,
	}

	// Register cleanup
	t.Cleanup(func() {
		ts.Close()
		db.Close()
		os.RemoveAll(repoRoot)
	})

	return env
}

// URL returns the full URL for an endpoint
func (e *TestEnv) URL(path string) string {
	return e.Server.URL + path
}

// Get makes a GET request
func (e *TestEnv) Get(path string) *http.Response {
	e.T.Helper()
	resp, err := e.Client.Get(e.URL(path))
	if err != nil {
		e.T.Fatalf("GET %s failed: %v", path, err)
	}
	return resp
}

// PostJSON makes a POST request with JSON body
func (e *TestEnv) PostJSON(path string, body interface{}) *http.Response {
	e.T.Helper()
	jsonBytes, err := json.Marshal(body)
	if err != nil {
		e.T.Fatalf("Failed to marshal JSON: %v", err)
	}

	resp, err := e.Client.Post(e.URL(path), "application/json", bytes.NewReader(jsonBytes))
	if err != nil {
		e.T.Fatalf("POST %s failed: %v", path, err)
	}
	return resp
}

// PatchJSON makes a PATCH request with JSON body
func (e *TestEnv) PatchJSON(path string, body interface{}) *http.Response {
	e.T.Helper()
	jsonBytes, err := json.Marshal(body)
	if err != nil {
		e.T.Fatalf("Failed to marshal JSON: %v", err)
	}

	req, err := http.NewRequest("PATCH", e.URL(path), bytes.NewReader(jsonBytes))
	if err != nil {
		e.T.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := e.Client.Do(req)
	if err != nil {
		e.T.Fatalf("PATCH %s failed: %v", path, err)
	}
	return resp
}

// PostMultipart makes a POST request with multipart form data
func (e *TestEnv) PostMultipart(path string, fields map[string]string, files map[string][]byte) *http.Response {
	e.T.Helper()

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Add fields
	for key, val := range fields {
		if err := writer.WriteField(key, val); err != nil {
			e.T.Fatalf("Failed to write field %s: %v", key, err)
		}
	}

	// Add files
	for key, content := range files {
		part, err := writer.CreateFormFile(key, key)
		if err != nil {
			e.T.Fatalf("Failed to create form file %s: %v", key, err)
		}
		if _, err := part.Write(content); err != nil {
			e.T.Fatalf("Failed to write file %s: %v", key, err)
		}
	}

	if err := writer.Close(); err != nil {
		e.T.Fatalf("Failed to close multipart writer: %v", err)
	}

	resp, err := e.Client.Post(e.URL(path), writer.FormDataContentType(), &buf)
	if err != nil {
		e.T.Fatalf("POST %s failed: %v", path, err)
	}
	return resp
}

// ReadJSON reads JSON response into target
func (e *TestEnv) ReadJSON(resp *http.Response, target interface{}) {
	e.T.Helper()
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
		e.T.Fatalf("Failed to decode JSON response: %v", err)
	}
}

// ReadBody reads the response body as string
func (e *TestEnv) ReadBody(resp *http.Response) string {
	e.T.Helper()
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		e.T.Fatalf("Failed to read response body: %v", err)
	}
	return string(body)
}

// FileExists checks if a file exists relative to repo root
func (e *TestEnv) FileExists(relPath string) bool {
	e.T.Helper()
	_, err := os.Stat(filepath.Join(e.RepoRoot, relPath))
	return err == nil
}

// ReadFile reads a file relative to repo root
func (e *TestEnv) ReadFile(relPath string) string {
	e.T.Helper()
	content, err := os.ReadFile(filepath.Join(e.RepoRoot, relPath))
	if err != nil {
		e.T.Fatalf("Failed to read file %s: %v", relPath, err)
	}
	return string(content)
}

// ReadFileBytes reads a file relative to repo root as bytes
func (e *TestEnv) ReadFileBytes(relPath string) []byte {
	e.T.Helper()
	content, err := os.ReadFile(filepath.Join(e.RepoRoot, relPath))
	if err != nil {
		e.T.Fatalf("Failed to read file %s: %v", relPath, err)
	}
	return content
}

// AssertStatus asserts the HTTP status code
func (e *TestEnv) AssertStatus(resp *http.Response, expected int) {
	e.T.Helper()
	if resp.StatusCode != expected {
		body, _ := io.ReadAll(resp.Body)
		e.T.Fatalf("Expected status %d, got %d. Body: %s", expected, resp.StatusCode, string(body))
	}
}

// PostForm makes a POST request with form-urlencoded data
func (e *TestEnv) PostForm(path string, data map[string]string) *http.Response {
	e.T.Helper()
	form := url.Values{}
	for key, val := range data {
		form.Set(key, val)
	}

	resp, err := e.Client.Post(e.URL(path), "application/x-www-form-urlencoded", strings.NewReader(form.Encode()))
	if err != nil {
		e.T.Fatalf("POST %s failed: %v", path, err)
	}
	return resp
}

// PostFormFollowRedirect makes a POST request with form data and follows redirects
func (e *TestEnv) PostFormFollowRedirect(path string, data map[string]string) *http.Response {
	e.T.Helper()
	form := url.Values{}
	for key, val := range data {
		form.Set(key, val)
	}

	req, err := http.NewRequest("POST", e.URL(path), strings.NewReader(form.Encode()))
	if err != nil {
		e.T.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := e.Client.Do(req)
	if err != nil {
		e.T.Fatalf("POST %s failed: %v", path, err)
	}
	return resp
}
