package ports

import (
	"context"
	"time"
)

// ExportCompanyRow holds raw company data for export.
type ExportCompanyRow struct {
	ID, Slug, Name, FolderPath string
	CreatedAt, UpdatedAt       time.Time
}

// ExportRoleRow holds raw role data for export.
type ExportRoleRow struct {
	ID, CompanyID, Slug, Title, Status, FolderPath string
	CreatedAt, UpdatedAt                            time.Time
}

// ExportContactRow holds raw contact data for export.
type ExportContactRow struct {
	ID, Name, Org, LinkedInURL, Email string
	CreatedAt, UpdatedAt              time.Time
}

// ExportMeetingRow holds raw meetings data for export.
type ExportMeetingRow struct {
	ID, OccurredAt, Title, PathMD string
	RoleID, ContactID             *string
	CreatedAt, UpdatedAt          time.Time
}

// ExportJobDescriptionRow holds raw job description data for export.
type ExportJobDescriptionRow struct {
	RoleID           string
	PathHTML, PathPDF *string
}

// ExportResumeRow holds raw resume data for export.
type ExportResumeRow struct {
	RoleID            string
	PathJSON, PathPDF *string
}

// ExportRoleArtifactRow holds raw role artifact data for export.
type ExportRoleArtifactRow struct {
	ID, RoleID, Name, Type, Path string
	CreatedAt, UpdatedAt         time.Time
}

// ExportQuerier defines read-only queries needed by the export service.
type ExportQuerier interface {
	QueryCompanies(ctx context.Context) ([]ExportCompanyRow, error)
	QueryRoles(ctx context.Context) ([]ExportRoleRow, error)
	QueryContacts(ctx context.Context) ([]ExportContactRow, error)
	QueryMeetings(ctx context.Context) ([]ExportMeetingRow, error)
	QueryJobDescriptions(ctx context.Context) ([]ExportJobDescriptionRow, error)
	QueryResumes(ctx context.Context) ([]ExportResumeRow, error)
	QueryRoleArtifacts(ctx context.Context) ([]ExportRoleArtifactRow, error)
}
