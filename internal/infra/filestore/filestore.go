package filestore

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	// slugRegexp matches non-alphanumeric characters
	slugRegexp = regexp.MustCompile(`[^a-z0-9]+`)
	// multiDashRegexp matches multiple consecutive dashes
	multiDashRegexp = regexp.MustCompile(`-+`)
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
// Path: data/threads/<thread-slug>/<YYYY-MM-DD>_<title>_<id>.md (flattened, no /meetings subfolder)
func (fs *FileStore) CreateThreadMeetingNote(ctx context.Context, threadSlug, occurredAt, title, meetingID string) (string, error) {
	// Format: YYYY-MM-DD_<title>_<id>.md
	safeTitle := strings.ReplaceAll(title, " ", "-")
	safeTitle = strings.ReplaceAll(safeTitle, "/", "-")
	filename := fmt.Sprintf("%s_%s_%s.md", occurredAt[:10], safeTitle, meetingID)

	// Create thread folder if it doesn't exist (flattened - no meetings subfolder)
	threadDir := filepath.Join("data", "threads", threadSlug)
	absThreadDir := filepath.Join(fs.repoRoot, threadDir)
	if err := os.MkdirAll(absThreadDir, 0755); err != nil {
		return "", fmt.Errorf("creating thread folder: %w", err)
	}

	filePath := filepath.Join(threadDir, filename)
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

// ReadFile reads the content of a file at the given relative path
func (fs *FileStore) ReadFile(ctx context.Context, path string) (string, error) {
	absPath := filepath.Join(fs.repoRoot, path)

	content, err := os.ReadFile(absPath)
	if err != nil {
		return "", fmt.Errorf("reading file: %w", err)
	}

	return string(content), nil
}

// SaveRoleResumeJSON saves the resume JSON data for a role
func (fs *FileStore) SaveRoleResumeJSON(ctx context.Context, companySlug, roleSlug string, content string) (string, error) {
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
func (fs *FileStore) SaveRoleResumePDF(ctx context.Context, companySlug, roleSlug string, content io.Reader) (string, error) {
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
	default:
		return ".txt"
	}
}

// SaveRoleArtifact saves an artifact file for a role
func (fs *FileStore) SaveRoleArtifact(ctx context.Context, companySlug, roleSlug, artifactName, artifactType string, content io.Reader) (string, error) {
	// Create artifacts folder if it doesn't exist
	artifactsDir := filepath.Join("data", "companies", companySlug, "roles", roleSlug, "artifacts")
	absArtifactsDir := filepath.Join(fs.repoRoot, artifactsDir)
	if err := os.MkdirAll(absArtifactsDir, 0755); err != nil {
		return "", fmt.Errorf("creating artifacts folder: %w", err)
	}

	// Generate filename from slugified name + extension
	filename := slugify(artifactName) + extensionForType(artifactType)
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
func (fs *FileStore) ReadFileBytes(ctx context.Context, path string) ([]byte, error) {
	absPath := filepath.Join(fs.repoRoot, path)
	content, err := os.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("reading file: %w", err)
	}
	return content, nil
}

// DeleteFile deletes a file at the given relative path
func (fs *FileStore) DeleteFile(ctx context.Context, path string) error {
	absPath := filepath.Join(fs.repoRoot, path)
	err := os.Remove(absPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("deleting file: %w", err)
	}
	return nil
}
