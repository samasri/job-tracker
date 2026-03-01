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

func TestSaveRoleArtifact_Text(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "filestore-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	fs := filestore.New(tempDir)
	ctx := context.Background()

	// Save text artifact
	textContent := strings.NewReader("This is a job description in plain text.")
	path, err := fs.SaveRoleArtifact(ctx, "acme-corp", "senior-engineer", "job-description", "text", "", textContent)
	if err != nil {
		t.Fatalf("SaveRoleArtifact failed: %v", err)
	}

	// Verify path format
	expectedPath := "data/companies/acme-corp/roles/senior-engineer/artifacts/job-description.txt"
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
	if string(content) != "This is a job description in plain text." {
		t.Errorf("Content mismatch: got %q", string(content))
	}
}

func TestSaveRoleArtifact_JSONC(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "filestore-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	fs := filestore.New(tempDir)
	ctx := context.Background()

	// Save JSONC artifact with comments
	jsoncContent := `{
  // This is a comment
  "name": "John Doe",
  "skills": ["Go", "TypeScript"], // inline comment
}`
	path, err := fs.SaveRoleArtifact(ctx, "acme-corp", "senior-engineer", "resume", "jsonc", "", strings.NewReader(jsoncContent))
	if err != nil {
		t.Fatalf("SaveRoleArtifact failed: %v", err)
	}

	// Verify path format
	expectedPath := "data/companies/acme-corp/roles/senior-engineer/artifacts/resume.jsonc"
	if path != expectedPath {
		t.Errorf("Expected path %q, got %q", expectedPath, path)
	}

	// Verify file exists and content is preserved exactly (including comments)
	absPath := filepath.Join(tempDir, path)
	content, err := os.ReadFile(absPath)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}
	if string(content) != jsoncContent {
		t.Errorf("Content should be preserved exactly, got %q", string(content))
	}
}

func TestSaveRoleArtifact_PDF(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "filestore-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	fs := filestore.New(tempDir)
	ctx := context.Background()

	// Save PDF artifact
	pdfContent := strings.NewReader("%PDF-1.4 fake pdf content for resume")
	path, err := fs.SaveRoleArtifact(ctx, "acme-corp", "senior-engineer", "Resume PDF", "pdf", "", pdfContent)
	if err != nil {
		t.Fatalf("SaveRoleArtifact failed: %v", err)
	}

	// Verify path format (name should be slugified)
	expectedPath := "data/companies/acme-corp/roles/senior-engineer/artifacts/resume-pdf.pdf"
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

func TestSaveRoleArtifact_PNG(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "filestore-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	fs := filestore.New(tempDir)
	ctx := context.Background()

	// Save PNG artifact (simulated PNG content - PNG magic bytes)
	pngContent := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	path, err := fs.SaveRoleArtifact(ctx, "acme-corp", "senior-engineer", "Screenshot", "png", "", strings.NewReader(string(pngContent)))
	if err != nil {
		t.Fatalf("SaveRoleArtifact failed: %v", err)
	}

	// Verify path format (name should be slugified)
	expectedPath := "data/companies/acme-corp/roles/senior-engineer/artifacts/screenshot.png"
	if path != expectedPath {
		t.Errorf("Expected path %q, got %q", expectedPath, path)
	}

	// Verify file exists
	absPath := filepath.Join(tempDir, path)
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		t.Errorf("File should exist at %s", absPath)
	}

	// Verify content is preserved (binary content)
	content, err := os.ReadFile(absPath)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}
	if len(content) != len(pngContent) {
		t.Errorf("Content length mismatch: expected %d, got %d", len(pngContent), len(content))
	}
	for i, b := range pngContent {
		if content[i] != b {
			t.Errorf("Content mismatch at byte %d: expected %02x, got %02x", i, b, content[i])
			break
		}
	}
}

func TestSaveRoleArtifact_Overwrite(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "filestore-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	fs := filestore.New(tempDir)
	ctx := context.Background()

	// Save initial artifact
	path1, err := fs.SaveRoleArtifact(ctx, "acme-corp", "senior-engineer", "notes", "text", "", strings.NewReader("version 1"))
	if err != nil {
		t.Fatalf("First SaveRoleArtifact failed: %v", err)
	}

	// Save updated artifact (should overwrite)
	path2, err := fs.SaveRoleArtifact(ctx, "acme-corp", "senior-engineer", "notes", "text", "", strings.NewReader("version 2"))
	if err != nil {
		t.Fatalf("Second SaveRoleArtifact failed: %v", err)
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
	if string(content) != "version 2" {
		t.Errorf("Content should be updated to 'version 2', got %q", string(content))
	}
}

func TestReadFileBytes(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "filestore-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	fs := filestore.New(tempDir)
	ctx := context.Background()

	// Save an artifact first
	originalContent := "test content for reading"
	path, err := fs.SaveRoleArtifact(ctx, "acme-corp", "dev", "test-file", "text", "", strings.NewReader(originalContent))
	if err != nil {
		t.Fatalf("SaveRoleArtifact failed: %v", err)
	}

	// Read it back
	readContent, err := fs.ReadFileBytes(ctx, path)
	if err != nil {
		t.Fatalf("ReadFileBytes failed: %v", err)
	}

	if string(readContent) != originalContent {
		t.Errorf("Content mismatch: expected %q, got %q", originalContent, string(readContent))
	}
}

func TestDeleteFile(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "filestore-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	fs := filestore.New(tempDir)
	ctx := context.Background()

	// Save an artifact
	path, err := fs.SaveRoleArtifact(ctx, "acme-corp", "dev", "to-delete", "text", "", strings.NewReader("delete me"))
	if err != nil {
		t.Fatalf("SaveRoleArtifact failed: %v", err)
	}

	// Verify file exists
	absPath := filepath.Join(tempDir, path)
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		t.Fatal("File should exist before deletion")
	}

	// Delete it
	if err := fs.DeleteFile(ctx, path); err != nil {
		t.Fatalf("DeleteFile failed: %v", err)
	}

	// Verify file no longer exists
	if _, err := os.Stat(absPath); !os.IsNotExist(err) {
		t.Error("File should not exist after deletion")
	}

	// Deleting non-existent file should not error
	if err := fs.DeleteFile(ctx, path); err != nil {
		t.Errorf("DeleteFile on non-existent file should not error: %v", err)
	}
}
