package filestore

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var (
	// slugRegexp matches non-alphanumeric characters
	slugRegexp = regexp.MustCompile(`[^a-z0-9]+`)
	// multiDashRegexp matches multiple consecutive dashes
	multiDashRegexp = regexp.MustCompile(`-+`)
)

// fileStore implements ports.FileStore using the local filesystem
type fileStore struct {
	repoRoot string
}

// New creates a new fileStore
func New(repoRoot string) *fileStore {
	return &fileStore{repoRoot: repoRoot}
}

// CreateCompanyFolder creates the company folder and company.md
func (fs *fileStore) CreateCompanyFolder(ctx context.Context, slug string) (string, error) {
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
func (fs *fileStore) CreateRoleFolder(ctx context.Context, companySlug, roleSlug string) (string, error) {
	folderPath := filepath.Join("data", "companies", companySlug, "roles", roleSlug)
	absPath := filepath.Join(fs.repoRoot, folderPath)

	if err := os.MkdirAll(absPath, 0755); err != nil {
		return "", fmt.Errorf("creating role folder: %w", err)
	}

	return folderPath, nil
}

// CreateRoleMeetingNote creates a meeting note file for a role meeting
// Path: data/companies/<company>/roles/<role>/meetings/<YYYY-MM-DD>_<title>_<id>.md
func (fs *fileStore) CreateRoleMeetingNote(ctx context.Context, companySlug, roleSlug, occurredAt, title, meetingID string) (string, error) {
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

// CreateContactFolder creates the contact folder
func (fs *fileStore) CreateContactFolder(ctx context.Context, slug string) (string, error) {
	folderPath := filepath.Join("data", "contacts", slug)
	absPath := filepath.Join(fs.repoRoot, folderPath)

	if err := os.MkdirAll(absPath, 0755); err != nil {
		return "", fmt.Errorf("creating contact folder: %w", err)
	}

	return folderPath, nil
}

// CreateContactMeetingNote creates a meeting note file for a contact meeting
// Path: data/contacts/<slug>/<YYYY-MM-DD>_<title>_<id>.md
func (fs *fileStore) CreateContactMeetingNote(ctx context.Context, contactSlug string, occurredAt time.Time, title, meetingID string) (string, error) {
	safeTitle := strings.ReplaceAll(title, " ", "-")
	safeTitle = strings.ReplaceAll(safeTitle, "/", "-")
	filename := fmt.Sprintf("%s_%s_%s.md", occurredAt.Format("2006-01-02"), safeTitle, meetingID)

	contactDir := filepath.Join("data", "contacts", contactSlug)
	absContactDir := filepath.Join(fs.repoRoot, contactDir)
	if err := os.MkdirAll(absContactDir, 0755); err != nil {
		return "", fmt.Errorf("creating contact folder: %w", err)
	}

	filePath := filepath.Join(contactDir, filename)
	absPath := filepath.Join(fs.repoRoot, filePath)

	content := fmt.Sprintf(`# %s

meeting_id: %s
occurred_at: %s

## Notes

`, title, meetingID, occurredAt.Format("2006-01-02T15:04:05Z07:00"))

	if err := os.WriteFile(absPath, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("creating contact meeting note: %w", err)
	}

	return filePath, nil
}

// MoveFile moves a file from oldPath to newPath (both relative to repoRoot)
func (fs *fileStore) MoveFile(ctx context.Context, oldPath, newPath string) error {
	absOld := filepath.Join(fs.repoRoot, oldPath)
	absNew := filepath.Join(fs.repoRoot, newPath)

	if err := os.MkdirAll(filepath.Dir(absNew), 0755); err != nil {
		return fmt.Errorf("creating parent directory for move: %w", err)
	}

	if err := os.Rename(absOld, absNew); err != nil {
		return fmt.Errorf("moving file from %s to %s: %w", oldPath, newPath, err)
	}

	return nil
}

// SaveJobDescriptionHTML saves the HTML job description
func (fs *fileStore) SaveJobDescriptionHTML(ctx context.Context, companySlug, roleSlug string, content string) (string, error) {
	filePath := filepath.Join("data", "companies", companySlug, "roles", roleSlug, "job.html")
	absPath := filepath.Join(fs.repoRoot, filePath)

	if err := os.WriteFile(absPath, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("saving job.html: %w", err)
	}

	return filePath, nil
}

// SaveJobDescriptionPDF saves the PDF job description
func (fs *fileStore) SaveJobDescriptionPDF(ctx context.Context, companySlug, roleSlug string, content io.Reader) (string, error) {
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

// ReadFile reads the content of a file at the given relative path
func (fs *fileStore) ReadFile(ctx context.Context, path string) (string, error) {
	absPath := filepath.Join(fs.repoRoot, path)

	content, err := os.ReadFile(absPath)
	if err != nil {
		return "", fmt.Errorf("reading file: %w", err)
	}

	return string(content), nil
}

// SaveRoleResumeJSON saves the resume JSON data for a role
func (fs *fileStore) SaveRoleResumeJSON(ctx context.Context, companySlug, roleSlug string, content string) (string, error) {
	// Create resume folder if it doesn't exist
	resumeDir := filepath.Join("data", "companies", companySlug, "roles", roleSlug, "resume")
	absResumeDir := filepath.Join(fs.repoRoot, resumeDir)
	if err := os.MkdirAll(absResumeDir, 0755); err != nil {
		return "", fmt.Errorf("creating resume folder: %w", err)
	}

	filePath := filepath.Join(resumeDir, "resume.jsonc")
	absPath := filepath.Join(fs.repoRoot, filePath)

	if err := os.WriteFile(absPath, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("writing resume.jsonc: %w", err)
	}

	return filePath, nil
}

// SaveRoleResumePDF saves the resume PDF for a role
func (fs *fileStore) SaveRoleResumePDF(ctx context.Context, companySlug, roleSlug string, content io.Reader) (string, error) {
	// Create resume folder if it doesn't exist
	resumeDir := filepath.Join("data", "companies", companySlug, "roles", roleSlug, "resume")
	absResumeDir := filepath.Join(fs.repoRoot, resumeDir)
	if err := os.MkdirAll(absResumeDir, 0755); err != nil {
		return "", fmt.Errorf("creating resume folder: %w", err)
	}

	filePath := filepath.Join(resumeDir, "resume.pdf")
	absPath := filepath.Join(fs.repoRoot, filePath)

	f, err := os.Create(absPath)
	if err != nil {
		return "", fmt.Errorf("creating resume.pdf: %w", err)
	}
	defer f.Close()

	if _, err := io.Copy(f, content); err != nil {
		return "", fmt.Errorf("writing resume.pdf: %w", err)
	}

	return filePath, nil
}

// slugify converts a string to a filesystem-safe slug
func slugify(s string) string {
	result := strings.ToLower(s)
	result = slugRegexp.ReplaceAllString(result, "-")
	result = multiDashRegexp.ReplaceAllString(result, "-")
	result = strings.Trim(result, "-")
	return result
}

// extensionForType returns the file extension for an artifact type
func extensionForType(artifactType string) string {
	switch artifactType {
	case "pdf":
		return ".pdf"
	case "jsonc":
		return ".jsonc"
	case "text":
		return ".txt"
	case "html":
		return ".html"
	case "markdown":
		return ".md"
	case "png":
		return ".png"
	case "file":
		return ""
	default:
		return ".txt"
	}
}

// SaveRoleArtifact saves an artifact file for a role
func (fs *fileStore) SaveRoleArtifact(ctx context.Context, companySlug, roleSlug, artifactName, artifactType, fileExtension string, content io.Reader) (string, error) {
	artifactsDir := filepath.Join("data", "companies", companySlug, "roles", roleSlug, "artifacts")
	absArtifactsDir := filepath.Join(fs.repoRoot, artifactsDir)
	if err := os.MkdirAll(absArtifactsDir, 0755); err != nil {
		return "", fmt.Errorf("creating artifacts folder: %w", err)
	}

	ext := extensionForType(artifactType)
	if artifactType == "file" && fileExtension != "" {
		ext = fileExtension
	}

	filename := slugify(artifactName) + ext
	filePath := filepath.Join(artifactsDir, filename)
	absPath := filepath.Join(fs.repoRoot, filePath)

	f, err := os.Create(absPath)
	if err != nil {
		return "", fmt.Errorf("creating artifact file: %w", err)
	}
	defer f.Close()

	if _, err := io.Copy(f, content); err != nil {
		return "", fmt.Errorf("writing artifact file: %w", err)
	}

	return filePath, nil
}

// ReadFileBytes reads the raw bytes of a file at the given relative path
func (fs *fileStore) ReadFileBytes(ctx context.Context, path string) ([]byte, error) {
	absPath := filepath.Join(fs.repoRoot, path)
	content, err := os.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("reading file: %w", err)
	}
	return content, nil
}

// DeleteFile deletes a file at the given relative path
func (fs *fileStore) DeleteFile(ctx context.Context, path string) error {
	absPath := filepath.Join(fs.repoRoot, path)
	err := os.Remove(absPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("deleting file: %w", err)
	}
	return nil
}
