package app

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"jobtracker/internal/domain"
	"jobtracker/internal/infra/sqlite"
)

// ExportService handles database export to JSON
type ExportService struct {
	db       *sqlite.DB
	repoRoot string
}

// NewExportService creates a new ExportService
func NewExportService(db *sqlite.DB, repoRoot string) *ExportService {
	return &ExportService{
		db:       db,
		repoRoot: repoRoot,
	}
}

// ExportData represents the complete database export
type ExportData struct {
	ExportedAt      string                  `json:"exported_at"`
	Companies       []ExportCompany         `json:"companies"`
	CompanyViews    []ExportCompanyView     `json:"company_views"`
	Roles           []ExportRole            `json:"roles"`
	Contacts        []ExportContact         `json:"contacts"`
	Threads         []ExportThread          `json:"threads"`
	Meetings        []ExportMeeting         `json:"meetings"`
	MeetingsV2      []ExportMeetingV2       `json:"meetings_v2"`
	MeetingThreads  []ExportMeetingThread   `json:"meeting_threads"`
	ThreadRoles     []ExportThreadRole      `json:"thread_roles"`
	JobDescriptions []ExportJobDescription  `json:"job_descriptions"`
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

// ExportThread represents a thread in the export
type ExportThread struct {
	ID         string `json:"id"`
	Code       string `json:"code,omitempty"`
	Slug       string `json:"slug,omitempty"`
	Title      string `json:"title"`
	ContactID  string `json:"contact_id,omitempty"`
	FolderPath string `json:"folder_path,omitempty"`
	CreatedAt  string `json:"created_at"`
	UpdatedAt  string `json:"updated_at"`
}

// ExportMeeting represents a meeting in the export
type ExportMeeting struct {
	ID         string `json:"id"`
	OccurredAt string `json:"occurred_at"`
	Title      string `json:"title"`
	CompanyID  string `json:"company_id"`
	PathMD     string `json:"path_md"`
	CreatedAt  string `json:"created_at"`
	UpdatedAt  string `json:"updated_at"`
}

// ExportMeetingV2 represents a meeting_v2 in the export
type ExportMeetingV2 struct {
	ID         string `json:"id"`
	OccurredAt string `json:"occurred_at"`
	Title      string `json:"title"`
	RoleID     string `json:"role_id,omitempty"`
	ThreadID   string `json:"thread_id,omitempty"`
	PathMD     string `json:"path_md"`
	CreatedAt  string `json:"created_at"`
	UpdatedAt  string `json:"updated_at"`
}

// ExportMeetingThread represents a meeting-thread link in the export
type ExportMeetingThread struct {
	MeetingID string `json:"meeting_id"`
	ThreadID  string `json:"thread_id"`
}

// ExportThreadRole represents a thread-role link in the export
type ExportThreadRole struct {
	ThreadID string `json:"thread_id"`
	RoleID   string `json:"role_id"`
}

// ExportJobDescription represents a job description in the export
type ExportJobDescription struct {
	RoleID   string `json:"role_id"`
	PathHTML string `json:"path_html,omitempty"`
	PathPDF  string `json:"path_pdf,omitempty"`
}

// Export exports the database to db/export.json
func (s *ExportService) Export(ctx context.Context) error {
	data := &ExportData{
		ExportedAt:      time.Now().UTC().Format(time.RFC3339),
		Companies:       []ExportCompany{},
		CompanyViews:    []ExportCompanyView{},
		Roles:           []ExportRole{},
		Contacts:        []ExportContact{},
		Threads:         []ExportThread{},
		Meetings:        []ExportMeeting{},
		MeetingsV2:      []ExportMeetingV2{},
		MeetingThreads:  []ExportMeetingThread{},
		ThreadRoles:     []ExportThreadRole{},
		JobDescriptions: []ExportJobDescription{},
	}

	// Export companies (ordered by id for determinism)
	rows, err := s.db.QueryContext(ctx, `SELECT id, slug, name, folder_path, created_at, updated_at FROM companies ORDER BY id`)
	if err != nil {
		return fmt.Errorf("exporting companies: %w", err)
	}
	for rows.Next() {
		var c ExportCompany
		var createdAt, updatedAt time.Time
		if err := rows.Scan(&c.ID, &c.Slug, &c.Name, &c.FolderPath, &createdAt, &updatedAt); err != nil {
			rows.Close()
			return fmt.Errorf("scanning company: %w", err)
		}
		c.CreatedAt = createdAt.Format(time.RFC3339)
		c.UpdatedAt = updatedAt.Format(time.RFC3339)
		data.Companies = append(data.Companies, c)
	}
	rows.Close()

	// Export roles (ordered by id for determinism)
	rows, err = s.db.QueryContext(ctx, `SELECT id, company_id, slug, title, status, folder_path, created_at, updated_at FROM roles ORDER BY id`)
	if err != nil {
		return fmt.Errorf("exporting roles: %w", err)
	}
	for rows.Next() {
		var r ExportRole
		var createdAt, updatedAt time.Time
		if err := rows.Scan(&r.ID, &r.CompanyID, &r.Slug, &r.Title, &r.Status, &r.FolderPath, &createdAt, &updatedAt); err != nil {
			rows.Close()
			return fmt.Errorf("scanning role: %w", err)
		}
		r.CreatedAt = createdAt.Format(time.RFC3339)
		r.UpdatedAt = updatedAt.Format(time.RFC3339)
		data.Roles = append(data.Roles, r)
	}
	rows.Close()

	// Compute company views (computed status based on roles)
	// Group roles by company for status computation
	rolesByCompany := make(map[string][]*domain.Role)
	for _, r := range data.Roles {
		rolesByCompany[r.CompanyID] = append(rolesByCompany[r.CompanyID], &domain.Role{
			Status: domain.RoleStatus(r.Status),
		})
	}
	// Generate company views ordered by company ID for determinism
	for _, c := range data.Companies {
		roles := rolesByCompany[c.ID]
		status := domain.ComputeCompanyStatus(roles)
		data.CompanyViews = append(data.CompanyViews, ExportCompanyView{
			CompanyID:      c.ID,
			ComputedStatus: status.String(),
		})
	}

	// Export contacts (ordered by id for determinism)
	rows, err = s.db.QueryContext(ctx, `SELECT id, name, org, linkedin_url, email, created_at, updated_at FROM contacts ORDER BY id`)
	if err != nil {
		return fmt.Errorf("exporting contacts: %w", err)
	}
	for rows.Next() {
		var c ExportContact
		var createdAt, updatedAt time.Time
		if err := rows.Scan(&c.ID, &c.Name, &c.Org, &c.LinkedInURL, &c.Email, &createdAt, &updatedAt); err != nil {
			rows.Close()
			return fmt.Errorf("scanning contact: %w", err)
		}
		c.CreatedAt = createdAt.Format(time.RFC3339)
		c.UpdatedAt = updatedAt.Format(time.RFC3339)
		data.Contacts = append(data.Contacts, c)
	}
	rows.Close()

	// Export threads (ordered by id for determinism)
	rows, err = s.db.QueryContext(ctx, `SELECT id, code, slug, title, contact_id, folder_path, created_at, updated_at FROM threads ORDER BY id`)
	if err != nil {
		return fmt.Errorf("exporting threads: %w", err)
	}
	for rows.Next() {
		var t ExportThread
		var code, slug, contactID, folderPath *string
		var createdAt, updatedAt time.Time
		if err := rows.Scan(&t.ID, &code, &slug, &t.Title, &contactID, &folderPath, &createdAt, &updatedAt); err != nil {
			rows.Close()
			return fmt.Errorf("scanning thread: %w", err)
		}
		if code != nil {
			t.Code = *code
		}
		if slug != nil {
			t.Slug = *slug
		}
		if contactID != nil {
			t.ContactID = *contactID
		}
		if folderPath != nil {
			t.FolderPath = *folderPath
		}
		t.CreatedAt = createdAt.Format(time.RFC3339)
		t.UpdatedAt = updatedAt.Format(time.RFC3339)
		data.Threads = append(data.Threads, t)
	}
	rows.Close()

	// Export meetings (ordered by id for determinism) - legacy table
	rows, err = s.db.QueryContext(ctx, `SELECT id, occurred_at, title, company_id, path_md, created_at, updated_at FROM meetings ORDER BY id`)
	if err != nil {
		return fmt.Errorf("exporting meetings: %w", err)
	}
	for rows.Next() {
		var m ExportMeeting
		var createdAt, updatedAt time.Time
		if err := rows.Scan(&m.ID, &m.OccurredAt, &m.Title, &m.CompanyID, &m.PathMD, &createdAt, &updatedAt); err != nil {
			rows.Close()
			return fmt.Errorf("scanning meeting: %w", err)
		}
		m.CreatedAt = createdAt.Format(time.RFC3339)
		m.UpdatedAt = updatedAt.Format(time.RFC3339)
		data.Meetings = append(data.Meetings, m)
	}
	rows.Close()

	// Export meetings_v2 (ordered by id for determinism) - new ROLE/THREAD model
	rows, err = s.db.QueryContext(ctx, `SELECT id, occurred_at, title, role_id, thread_id, path_md, created_at, updated_at FROM meetings_v2 ORDER BY id`)
	if err != nil {
		return fmt.Errorf("exporting meetings_v2: %w", err)
	}
	for rows.Next() {
		var m ExportMeetingV2
		var roleID, threadID *string
		var createdAt, updatedAt time.Time
		if err := rows.Scan(&m.ID, &m.OccurredAt, &m.Title, &roleID, &threadID, &m.PathMD, &createdAt, &updatedAt); err != nil {
			rows.Close()
			return fmt.Errorf("scanning meeting_v2: %w", err)
		}
		if roleID != nil {
			m.RoleID = *roleID
		}
		if threadID != nil {
			m.ThreadID = *threadID
		}
		m.CreatedAt = createdAt.Format(time.RFC3339)
		m.UpdatedAt = updatedAt.Format(time.RFC3339)
		data.MeetingsV2 = append(data.MeetingsV2, m)
	}
	rows.Close()

	// Export meeting_threads (ordered for determinism)
	rows, err = s.db.QueryContext(ctx, `SELECT meeting_id, thread_id FROM meeting_threads ORDER BY meeting_id, thread_id`)
	if err != nil {
		return fmt.Errorf("exporting meeting_threads: %w", err)
	}
	for rows.Next() {
		var mt ExportMeetingThread
		if err := rows.Scan(&mt.MeetingID, &mt.ThreadID); err != nil {
			rows.Close()
			return fmt.Errorf("scanning meeting_thread: %w", err)
		}
		data.MeetingThreads = append(data.MeetingThreads, mt)
	}
	rows.Close()

	// Export thread_roles (ordered for determinism)
	rows, err = s.db.QueryContext(ctx, `SELECT thread_id, role_id FROM thread_roles ORDER BY thread_id, role_id`)
	if err != nil {
		return fmt.Errorf("exporting thread_roles: %w", err)
	}
	for rows.Next() {
		var tr ExportThreadRole
		if err := rows.Scan(&tr.ThreadID, &tr.RoleID); err != nil {
			rows.Close()
			return fmt.Errorf("scanning thread_role: %w", err)
		}
		data.ThreadRoles = append(data.ThreadRoles, tr)
	}
	rows.Close()

	// Export job_descriptions (ordered by role_id for determinism)
	rows, err = s.db.QueryContext(ctx, `SELECT role_id, path_html, path_pdf FROM role_job_descriptions ORDER BY role_id`)
	if err != nil {
		return fmt.Errorf("exporting job_descriptions: %w", err)
	}
	for rows.Next() {
		var jd ExportJobDescription
		var pathHTML, pathPDF *string
		if err := rows.Scan(&jd.RoleID, &pathHTML, &pathPDF); err != nil {
			rows.Close()
			return fmt.Errorf("scanning job_description: %w", err)
		}
		if pathHTML != nil {
			jd.PathHTML = *pathHTML
		}
		if pathPDF != nil {
			jd.PathPDF = *pathPDF
		}
		data.JobDescriptions = append(data.JobDescriptions, jd)
	}
	rows.Close()

	// Write to file
	exportPath := filepath.Join(s.repoRoot, "db", "export.json")
	if err := os.MkdirAll(filepath.Dir(exportPath), 0755); err != nil {
		return fmt.Errorf("creating export directory: %w", err)
	}

	// Use indented JSON for readability
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling export data: %w", err)
	}

	if err := os.WriteFile(exportPath, jsonData, 0644); err != nil {
		return fmt.Errorf("writing export file: %w", err)
	}

	return nil
}
