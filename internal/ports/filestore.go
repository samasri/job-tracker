package ports

import (
	"context"
	"io"
)

// FileStore defines operations for artifact storage on filesystem
type FileStore interface {
	// CreateCompanyFolder creates the company folder structure and company.md
	CreateCompanyFolder(ctx context.Context, slug string) (folderPath string, err error)

	// CreateRoleFolder creates the role folder structure
	CreateRoleFolder(ctx context.Context, companySlug, roleSlug string) (folderPath string, err error)

	// CreateMeetingNote creates a meeting note file (legacy, for old meetings table)
	CreateMeetingNote(ctx context.Context, companySlug, meetingID string, occurredAt, title string) (filePath string, err error)

	// CreateRoleMeetingNote creates a meeting note file for a role meeting (meetings_v2)
	// Path: data/companies/<company>/roles/<role>/meetings/<YYYY-MM-DD>_<title>_<id>.md
	CreateRoleMeetingNote(ctx context.Context, companySlug, roleSlug, occurredAt, title, meetingID string) (filePath string, err error)

	// CreateThreadMeetingNote creates a meeting note file for a thread-only meeting (meetings_v2)
	// Path: data/threads/<thread-slug>/<YYYY-MM-DD>_<title>_<id>.md (flattened, no /meetings subfolder)
	CreateThreadMeetingNote(ctx context.Context, threadSlug, occurredAt, title, meetingID string) (filePath string, err error)

	// SaveJobDescriptionHTML saves the HTML job description
	SaveJobDescriptionHTML(ctx context.Context, companySlug, roleSlug string, content string) (filePath string, err error)

	// SaveJobDescriptionPDF saves the PDF job description
	SaveJobDescriptionPDF(ctx context.Context, companySlug, roleSlug string, content io.Reader) (filePath string, err error)

	// ReadFile reads the content of a file at the given relative path
	ReadFile(ctx context.Context, path string) (string, error)

	// SaveRoleResumeJSON saves the resume JSON data for a role
	SaveRoleResumeJSON(ctx context.Context, companySlug, roleSlug string, content string) (filePath string, err error)

	// SaveRoleResumePDF saves the resume PDF for a role
	SaveRoleResumePDF(ctx context.Context, companySlug, roleSlug string, content io.Reader) (filePath string, err error)

	// SaveRoleArtifact saves an artifact file for a role.
	// fileExtension is used when artifactType is "file" to preserve the original extension.
	SaveRoleArtifact(ctx context.Context, companySlug, roleSlug, artifactName, artifactType, fileExtension string, content io.Reader) (filePath string, err error)

	// ReadFileBytes reads the raw bytes of a file at the given relative path
	ReadFileBytes(ctx context.Context, path string) ([]byte, error)

	// DeleteFile deletes a file at the given relative path
	DeleteFile(ctx context.Context, path string) error
}
