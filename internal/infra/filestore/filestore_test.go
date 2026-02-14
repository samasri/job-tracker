package filestore_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"jobtracker/internal/infra/filestore"
)

func TestCreateRoleMeetingNote(t *testing.T) {
	// Create temp directory
	tempDir, err := os.MkdirTemp("", "filestore-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	fs := filestore.New(tempDir)
	ctx := context.Background()

	// Create role meeting note
	path, err := fs.CreateRoleMeetingNote(ctx, "acme-corp", "senior-engineer", "2024-06-15T10:00:00Z", "Technical Interview", "ABC12345")
	if err != nil {
		t.Fatalf("CreateRoleMeetingNote failed: %v", err)
	}

	// Verify path format
	expectedPath := "data/companies/acme-corp/roles/senior-engineer/meetings/2024-06-15_Technical-Interview_ABC12345.md"
	if path != expectedPath {
		t.Errorf("Expected path %q, got %q", expectedPath, path)
	}

	// Verify file exists
	absPath := filepath.Join(tempDir, path)
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		t.Errorf("File should exist at %s", absPath)
	}

	// Verify content
	content, err := os.ReadFile(absPath)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, "# Technical Interview") {
		t.Error("Content should contain title")
	}
	if !strings.Contains(contentStr, "meeting_id: ABC12345") {
		t.Error("Content should contain meeting_id")
	}
	if !strings.Contains(contentStr, "occurred_at: 2024-06-15T10:00:00Z") {
		t.Error("Content should contain occurred_at")
	}
}

func TestCreateThreadMeetingNote(t *testing.T) {
	// Create temp directory
	tempDir, err := os.MkdirTemp("", "filestore-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	fs := filestore.New(tempDir)
	ctx := context.Background()

	// Create thread meeting note using thread slug (flattened path, no /meetings subfolder)
	threadSlug := "john-doe-ABC12345"
	path, err := fs.CreateThreadMeetingNote(ctx, threadSlug, "2024-07-20T14:30:00Z", "Coffee Chat", "XYZ98765")
	if err != nil {
		t.Fatalf("CreateThreadMeetingNote failed: %v", err)
	}

	// Verify path format (flattened - no /meetings subfolder)
	expectedPath := "data/threads/john-doe-ABC12345/2024-07-20_Coffee-Chat_XYZ98765.md"
	if path != expectedPath {
		t.Errorf("Expected path %q, got %q", expectedPath, path)
	}

	// Verify file exists
	absPath := filepath.Join(tempDir, path)
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		t.Errorf("File should exist at %s", absPath)
	}

	// Verify content
	content, err := os.ReadFile(absPath)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, "# Coffee Chat") {
		t.Error("Content should contain title")
	}
	if !strings.Contains(contentStr, "meeting_id: XYZ98765") {
		t.Error("Content should contain meeting_id")
	}
	if !strings.Contains(contentStr, "occurred_at: 2024-07-20T14:30:00Z") {
		t.Error("Content should contain occurred_at")
	}
}

func TestCreateBothMeetingTypes(t *testing.T) {
	// Test that both role and thread meeting types can coexist in the same repo
	tempDir, err := os.MkdirTemp("", "filestore-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	fs := filestore.New(tempDir)
	ctx := context.Background()

	// Create role meeting
	rolePath, err := fs.CreateRoleMeetingNote(ctx, "tech-co", "backend-dev", "2024-08-01T09:00:00Z", "System Design", "ROLE1234")
	if err != nil {
		t.Fatalf("CreateRoleMeetingNote failed: %v", err)
	}

	// Create thread meeting (using thread slug, flattened path)
	threadPath, err := fs.CreateThreadMeetingNote(ctx, "recruiter-jane-XYZ99999", "2024-08-01T11:00:00Z", "Networking Call", "THRD5678")
	if err != nil {
		t.Fatalf("CreateThreadMeetingNote failed: %v", err)
	}

	// Verify both files exist
	if _, err := os.Stat(filepath.Join(tempDir, rolePath)); os.IsNotExist(err) {
		t.Error("Role meeting file should exist")
	}
	if _, err := os.Stat(filepath.Join(tempDir, threadPath)); os.IsNotExist(err) {
		t.Error("Thread meeting file should exist")
	}

	// Verify they're in different directories
	if strings.Contains(rolePath, "threads") {
		t.Error("Role meeting path should not contain 'threads'")
	}
	if strings.Contains(threadPath, "companies") {
		t.Error("Thread meeting path should not contain 'companies'")
	}

	// Verify thread path is flattened (no /meetings subfolder)
	if strings.Contains(threadPath, "/meetings/") {
		t.Error("Thread meeting path should be flattened (no /meetings subfolder)")
	}
}

func TestSaveRoleResumeJSON(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "filestore-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	fs := filestore.New(tempDir)
	ctx := context.Background()

	// Save resume JSON
	jsonContent := `{"name": "Test Resume", "skills": ["Go", "TypeScript"]}`
	path, err := fs.SaveRoleResumeJSON(ctx, "acme-corp", "senior-engineer", jsonContent)
	if err != nil {
		t.Fatalf("SaveRoleResumeJSON failed: %v", err)
	}

	// Verify path format
	expectedPath := "data/companies/acme-corp/roles/senior-engineer/resume/resume.jsonc"
	if path != expectedPath {
		t.Errorf("Expected path %q, got %q", expectedPath, path)
	}

	// Verify file exists
	absPath := filepath.Join(tempDir, path)
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		t.Errorf("File should exist at %s", absPath)
	}

	// Verify content
	content, err := os.ReadFile(absPath)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}
	if !strings.Contains(string(content), "Test Resume") {
		t.Error("Content should contain resume data")
	}
}

func TestSaveRoleResumePDF(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "filestore-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	fs := filestore.New(tempDir)
	ctx := context.Background()

	// Save resume PDF (simulated PDF content)
	pdfContent := strings.NewReader("%PDF-1.4 fake pdf content")
	path, err := fs.SaveRoleResumePDF(ctx, "acme-corp", "senior-engineer", pdfContent)
	if err != nil {
		t.Fatalf("SaveRoleResumePDF failed: %v", err)
	}

	// Verify path format
	expectedPath := "data/companies/acme-corp/roles/senior-engineer/resume/resume.pdf"
	if path != expectedPath {
		t.Errorf("Expected path %q, got %q", expectedPath, path)
	}

	// Verify file exists
	absPath := filepath.Join(tempDir, path)
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		t.Errorf("File should exist at %s", absPath)
	}

	// Verify content
	content, err := os.ReadFile(absPath)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}
	if !strings.Contains(string(content), "%PDF") {
		t.Error("Content should contain PDF data")
	}
}

func TestSaveRoleResumeOverwrite(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "filestore-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	fs := filestore.New(tempDir)
	ctx := context.Background()

	// Save initial resume JSON
	jsonContent1 := `{"version": 1}`
	path1, err := fs.SaveRoleResumeJSON(ctx, "acme-corp", "senior-engineer", jsonContent1)
	if err != nil {
		t.Fatalf("First SaveRoleResumeJSON failed: %v", err)
	}

	// Save updated resume JSON (should overwrite)
	jsonContent2 := `{"version": 2}`
	path2, err := fs.SaveRoleResumeJSON(ctx, "acme-corp", "senior-engineer", jsonContent2)
	if err != nil {
		t.Fatalf("Second SaveRoleResumeJSON failed: %v", err)
	}

	// Verify same path
	if path1 != path2 {
		t.Errorf("Paths should be the same: %q vs %q", path1, path2)
	}

	// Verify content is updated
	absPath := filepath.Join(tempDir, path2)
	content, err := os.ReadFile(absPath)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}
	if !strings.Contains(string(content), `"version": 2`) {
		t.Error("Content should be updated to version 2")
	}
	if strings.Contains(string(content), `"version": 1`) {
		t.Error("Old content should be overwritten")
	}
}
