package filestore

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// FileStore implements ports.FileStore using the local filesystem
type FileStore struct {
	repoRoot string
}

// New creates a new FileStore
func New(repoRoot string) *FileStore {
	return &FileStore{repoRoot: repoRoot}
}

// CreateCompanyFolder creates the company folder and company.md
func (fs *FileStore) CreateCompanyFolder(ctx context.Context, slug string) (string, error) {
	folderPath := filepath.Join("data", "companies", slug)
	absPath := filepath.Join(fs.repoRoot, folderPath)

	// Create company folder
	if err := os.MkdirAll(absPath, 0755); err != nil {
		return "", fmt.Errorf("creating company folder: %w", err)
	}

	// Create roles subfolder
	if err := os.MkdirAll(filepath.Join(absPath, "roles"), 0755); err != nil {
		return "", fmt.Errorf("creating roles folder: %w", err)
	}

	// Create meetings subfolder
	if err := os.MkdirAll(filepath.Join(absPath, "meetings"), 0755); err != nil {
		return "", fmt.Errorf("creating meetings folder: %w", err)
	}

	// Create resumes subfolder
	if err := os.MkdirAll(filepath.Join(absPath, "resumes"), 0755); err != nil {
		return "", fmt.Errorf("creating resumes folder: %w", err)
	}

	// Create company.md for notes (status is now computed from roles, not stored in file)
	companyMDPath := filepath.Join(absPath, "company.md")
	if _, err := os.Stat(companyMDPath); os.IsNotExist(err) {
		content := `# Company Notes

`
		if err := os.WriteFile(companyMDPath, []byte(content), 0644); err != nil {
			return "", fmt.Errorf("creating company.md: %w", err)
		}
	}

	return folderPath, nil
}

// CreateRoleFolder creates the role folder structure
func (fs *FileStore) CreateRoleFolder(ctx context.Context, companySlug, roleSlug string) (string, error) {
	folderPath := filepath.Join("data", "companies", companySlug, "roles", roleSlug)
	absPath := filepath.Join(fs.repoRoot, folderPath)

	if err := os.MkdirAll(absPath, 0755); err != nil {
		return "", fmt.Errorf("creating role folder: %w", err)
	}

	return folderPath, nil
}

// CreateMeetingNote creates a meeting note file
func (fs *FileStore) CreateMeetingNote(ctx context.Context, companySlug, meetingID string, occurredAt, title string) (string, error) {
	// Format: YYYY-MM-DD_<title>_<id>.md
	safeTitle := strings.ReplaceAll(title, " ", "-")
	safeTitle = strings.ReplaceAll(safeTitle, "/", "-")
	filename := fmt.Sprintf("%s_%s_%s.md", occurredAt[:10], safeTitle, meetingID)

	filePath := filepath.Join("data", "companies", companySlug, "meetings", filename)
	absPath := filepath.Join(fs.repoRoot, filePath)

	content := fmt.Sprintf(`# %s

meeting_id: %s
occurred_at: %s

## Notes

`, title, meetingID, occurredAt)

	if err := os.WriteFile(absPath, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("creating meeting note: %w", err)
	}

	return filePath, nil
}

// CreateRoleMeetingNote creates a meeting note file for a role meeting
// Path: data/companies/<company>/roles/<role>/meetings/<YYYY-MM-DD>_<title>_<id>.md
func (fs *FileStore) CreateRoleMeetingNote(ctx context.Context, companySlug, roleSlug, occurredAt, title, meetingID string) (string, error) {
	// Format: YYYY-MM-DD_<title>_<id>.md
	safeTitle := strings.ReplaceAll(title, " ", "-")
	safeTitle = strings.ReplaceAll(safeTitle, "/", "-")
	filename := fmt.Sprintf("%s_%s_%s.md", occurredAt[:10], safeTitle, meetingID)

	// Create meetings folder under role if it doesn't exist
	meetingsDir := filepath.Join("data", "companies", companySlug, "roles", roleSlug, "meetings")
	absMeetingsDir := filepath.Join(fs.repoRoot, meetingsDir)
	if err := os.MkdirAll(absMeetingsDir, 0755); err != nil {
		return "", fmt.Errorf("creating role meetings folder: %w", err)
	}

	filePath := filepath.Join(meetingsDir, filename)
	absPath := filepath.Join(fs.repoRoot, filePath)

	content := fmt.Sprintf(`# %s

meeting_id: %s
occurred_at: %s

## Notes

`, title, meetingID, occurredAt)

	if err := os.WriteFile(absPath, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("creating role meeting note: %w", err)
	}

	return filePath, nil
}

// CreateThreadMeetingNote creates a meeting note file for a thread-only meeting
// Path: data/threads/<thread-id>/meetings/<YYYY-MM-DD>_<title>_<id>.md
func (fs *FileStore) CreateThreadMeetingNote(ctx context.Context, threadID, occurredAt, title, meetingID string) (string, error) {
	// Format: YYYY-MM-DD_<title>_<id>.md
	safeTitle := strings.ReplaceAll(title, " ", "-")
	safeTitle = strings.ReplaceAll(safeTitle, "/", "-")
	filename := fmt.Sprintf("%s_%s_%s.md", occurredAt[:10], safeTitle, meetingID)

	// Create meetings folder under thread if it doesn't exist
	meetingsDir := filepath.Join("data", "threads", threadID, "meetings")
	absMeetingsDir := filepath.Join(fs.repoRoot, meetingsDir)
	if err := os.MkdirAll(absMeetingsDir, 0755); err != nil {
		return "", fmt.Errorf("creating thread meetings folder: %w", err)
	}

	filePath := filepath.Join(meetingsDir, filename)
	absPath := filepath.Join(fs.repoRoot, filePath)

	content := fmt.Sprintf(`# %s

meeting_id: %s
occurred_at: %s

## Notes

`, title, meetingID, occurredAt)

	if err := os.WriteFile(absPath, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("creating thread meeting note: %w", err)
	}

	return filePath, nil
}

// SaveJobDescriptionHTML saves the HTML job description
func (fs *FileStore) SaveJobDescriptionHTML(ctx context.Context, companySlug, roleSlug string, content string) (string, error) {
	filePath := filepath.Join("data", "companies", companySlug, "roles", roleSlug, "job.html")
	absPath := filepath.Join(fs.repoRoot, filePath)

	if err := os.WriteFile(absPath, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("saving job.html: %w", err)
	}

	return filePath, nil
}

// SaveJobDescriptionPDF saves the PDF job description
func (fs *FileStore) SaveJobDescriptionPDF(ctx context.Context, companySlug, roleSlug string, content io.Reader) (string, error) {
	filePath := filepath.Join("data", "companies", companySlug, "roles", roleSlug, "job.pdf")
	absPath := filepath.Join(fs.repoRoot, filePath)

	f, err := os.Create(absPath)
	if err != nil {
		return "", fmt.Errorf("creating job.pdf: %w", err)
	}
	defer f.Close()

	if _, err := io.Copy(f, content); err != nil {
		return "", fmt.Errorf("writing job.pdf: %w", err)
	}

	return filePath, nil
}
