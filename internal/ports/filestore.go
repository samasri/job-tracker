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

	// CreateMeetingNote creates a meeting note file
	CreateMeetingNote(ctx context.Context, companySlug, meetingID string, occurredAt, title string) (filePath string, err error)

	// SaveJobDescriptionHTML saves the HTML job description
	SaveJobDescriptionHTML(ctx context.Context, companySlug, roleSlug string, content string) (filePath string, err error)

	// SaveJobDescriptionPDF saves the PDF job description
	SaveJobDescriptionPDF(ctx context.Context, companySlug, roleSlug string, content io.Reader) (filePath string, err error)
}
