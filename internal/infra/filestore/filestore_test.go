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

	// Create thread meeting note
	threadID := "thread-uuid-123"
	path, err := fs.CreateThreadMeetingNote(ctx, threadID, "2024-07-20T14:30:00Z", "Coffee Chat", "XYZ98765")
	if err != nil {
		t.Fatalf("CreateThreadMeetingNote failed: %v", err)
	}

	// Verify path format
	expectedPath := "data/threads/thread-uuid-123/meetings/2024-07-20_Coffee-Chat_XYZ98765.md"
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

	// Create thread meeting
	threadPath, err := fs.CreateThreadMeetingNote(ctx, "recruiter-thread", "2024-08-01T11:00:00Z", "Networking Call", "THRD5678")
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
}
