package app

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"jobtracker/internal/domain"
	"jobtracker/internal/ports"
)

// ExportService handles database export to JSON
type ExportService struct {
	querier  ports.ExportQuerier
	repoRoot string
}

// NewExportService creates a new ExportService
func NewExportService(querier ports.ExportQuerier, repoRoot string) *ExportService {
	return &ExportService{
		querier:  querier,
		repoRoot: repoRoot,
	}
}

// ExportData represents the complete database export
type ExportData struct {
	ExportedAt      string                 `json:"exported_at"`
	Companies       []ExportCompany        `json:"companies"`
	CompanyViews    []ExportCompanyView    `json:"company_views"`
	Roles           []ExportRole           `json:"roles"`
	Contacts        []ExportContact        `json:"contacts"`
	Meetings        []ExportMeeting        `json:"meetings"`
	JobDescriptions []ExportJobDescription `json:"job_descriptions"`
	Resumes         []ExportResume         `json:"resumes"`
	RoleArtifacts   []ExportRoleArtifact   `json:"role_artifacts"`
}

// ExportCompanyView represents computed view data for a company
type ExportCompanyView struct {
	CompanyID      string `json:"company_id"`
	ComputedStatus string `json:"computed_status"`
}

// ExportCompany represents a company in the export
type ExportCompany struct {
	ID         string `json:"id"`
	Slug       string `json:"slug"`
	Name       string `json:"name"`
	FolderPath string `json:"folder_path"`
	CreatedAt  string `json:"created_at"`
	UpdatedAt  string `json:"updated_at"`
}

// ExportRole represents a role in the export
type ExportRole struct {
	ID         string `json:"id"`
	CompanyID  string `json:"company_id"`
	Slug       string `json:"slug"`
	Title      string `json:"title"`
	Status     string `json:"status"`
	FolderPath string `json:"folder_path"`
	CreatedAt  string `json:"created_at"`
	UpdatedAt  string `json:"updated_at"`
}

// ExportContact represents a contact in the export
type ExportContact struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Org         string `json:"org"`
	LinkedInURL string `json:"linkedin_url"`
	Email       string `json:"email"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

// ExportMeeting represents a meeting in the export
type ExportMeeting struct {
	ID         string `json:"id"`
	OccurredAt string `json:"occurred_at"`
	Title      string `json:"title"`
	RoleID     string `json:"role_id,omitempty"`
	ContactID  string `json:"contact_id,omitempty"`
	PathMD     string `json:"path_md"`
	CreatedAt  string `json:"created_at"`
	UpdatedAt  string `json:"updated_at"`
}

// ExportJobDescription represents a job description in the export
type ExportJobDescription struct {
	RoleID   string `json:"role_id"`
	PathHTML string `json:"path_html,omitempty"`
	PathPDF  string `json:"path_pdf,omitempty"`
}

// ExportResume represents a role resume in the export
type ExportResume struct {
	RoleID   string `json:"role_id"`
	PathJSON string `json:"path_json,omitempty"`
	PathPDF  string `json:"path_pdf,omitempty"`
}

// ExportRoleArtifact represents a role artifact in the export
type ExportRoleArtifact struct {
	ID        string `json:"id"`
	RoleID    string `json:"role_id"`
	Name      string `json:"name"`
	Type      string `json:"type"`
	Path      string `json:"path"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// Export exports the database to db/export.json
func (s *ExportService) Export(ctx context.Context) error {
	data := &ExportData{
		ExportedAt:      time.Now().UTC().Format(time.RFC3339),
		Companies:       []ExportCompany{},
		CompanyViews:    []ExportCompanyView{},
		Roles:           []ExportRole{},
		Contacts:        []ExportContact{},
		Meetings:        []ExportMeeting{},
		JobDescriptions: []ExportJobDescription{},
		Resumes:         []ExportResume{},
		RoleArtifacts:   []ExportRoleArtifact{},
	}

	companyRows, err := s.querier.QueryCompanies(ctx)
	if err != nil {
		return fmt.Errorf("exporting companies: %w", err)
	}
	for _, r := range companyRows {
		data.Companies = append(data.Companies, ExportCompany{
			ID:         r.ID,
			Slug:       r.Slug,
			Name:       r.Name,
			FolderPath: r.FolderPath,
			CreatedAt:  r.CreatedAt.Format(time.RFC3339),
			UpdatedAt:  r.UpdatedAt.Format(time.RFC3339),
		})
	}

	roleRows, err := s.querier.QueryRoles(ctx)
	if err != nil {
		return fmt.Errorf("exporting roles: %w", err)
	}
	for _, r := range roleRows {
		data.Roles = append(data.Roles, ExportRole{
			ID:         r.ID,
			CompanyID:  r.CompanyID,
			Slug:       r.Slug,
			Title:      r.Title,
			Status:     r.Status,
			FolderPath: r.FolderPath,
			CreatedAt:  r.CreatedAt.Format(time.RFC3339),
			UpdatedAt:  r.UpdatedAt.Format(time.RFC3339),
		})
	}

	// Compute company views from roles (domain logic stays in app layer)
	rolesByCompany := make(map[string][]*domain.Role)
	for _, r := range data.Roles {
		rolesByCompany[r.CompanyID] = append(rolesByCompany[r.CompanyID], &domain.Role{
			Status: domain.RoleStatus(r.Status),
		})
	}
	for _, c := range data.Companies {
		status := domain.ComputeCompanyStatus(rolesByCompany[c.ID])
		data.CompanyViews = append(data.CompanyViews, ExportCompanyView{
			CompanyID:      c.ID,
			ComputedStatus: status.String(),
		})
	}

	contactRows, err := s.querier.QueryContacts(ctx)
	if err != nil {
		return fmt.Errorf("exporting contacts: %w", err)
	}
	for _, r := range contactRows {
		data.Contacts = append(data.Contacts, ExportContact{
			ID:          r.ID,
			Name:        r.Name,
			Org:         r.Org,
			LinkedInURL: r.LinkedInURL,
			Email:       r.Email,
			CreatedAt:   r.CreatedAt.Format(time.RFC3339),
			UpdatedAt:   r.UpdatedAt.Format(time.RFC3339),
		})
	}

	meetingRows, err := s.querier.QueryMeetings(ctx)
	if err != nil {
		return fmt.Errorf("exporting meetings: %w", err)
	}
	for _, r := range meetingRows {
		m := ExportMeeting{
			ID:         r.ID,
			OccurredAt: r.OccurredAt,
			Title:      r.Title,
			PathMD:     r.PathMD,
			CreatedAt:  r.CreatedAt.Format(time.RFC3339),
			UpdatedAt:  r.UpdatedAt.Format(time.RFC3339),
		}
		if r.RoleID != nil {
			m.RoleID = *r.RoleID
		}
		if r.ContactID != nil {
			m.ContactID = *r.ContactID
		}
		data.Meetings = append(data.Meetings, m)
	}

	jdRows, err := s.querier.QueryJobDescriptions(ctx)
	if err != nil {
		return fmt.Errorf("exporting job_descriptions: %w", err)
	}
	for _, r := range jdRows {
		jd := ExportJobDescription{RoleID: r.RoleID}
		if r.PathHTML != nil {
			jd.PathHTML = *r.PathHTML
		}
		if r.PathPDF != nil {
			jd.PathPDF = *r.PathPDF
		}
		data.JobDescriptions = append(data.JobDescriptions, jd)
	}

	resumeRows, err := s.querier.QueryResumes(ctx)
	if err != nil {
		return fmt.Errorf("exporting resumes: %w", err)
	}
	for _, r := range resumeRows {
		res := ExportResume{RoleID: r.RoleID}
		if r.PathJSON != nil {
			res.PathJSON = *r.PathJSON
		}
		if r.PathPDF != nil {
			res.PathPDF = *r.PathPDF
		}
		data.Resumes = append(data.Resumes, res)
	}

	artifactRows, err := s.querier.QueryRoleArtifacts(ctx)
	if err != nil {
		return fmt.Errorf("exporting role_artifacts: %w", err)
	}
	for _, r := range artifactRows {
		data.RoleArtifacts = append(data.RoleArtifacts, ExportRoleArtifact{
			ID:        r.ID,
			RoleID:    r.RoleID,
			Name:      r.Name,
			Type:      r.Type,
			Path:      r.Path,
			CreatedAt: r.CreatedAt.Format(time.RFC3339),
			UpdatedAt: r.UpdatedAt.Format(time.RFC3339),
		})
	}

	exportPath := filepath.Join(s.repoRoot, "db", "export.json")
	if err := os.MkdirAll(filepath.Dir(exportPath), 0755); err != nil {
		return fmt.Errorf("creating export directory: %w", err)
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling export data: %w", err)
	}

	if err := os.WriteFile(exportPath, jsonData, 0644); err != nil {
		return fmt.Errorf("writing export file: %w", err)
	}

	return nil
}
