package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	"jobtracker/internal/ports"
)

// ExportQuerier implements ports.ExportQuerier using direct SQL queries.
type ExportQuerier struct {
	db *DB
}

// NewExportQuerier creates a new ExportQuerier.
func NewExportQuerier(db *DB) *ExportQuerier {
	return &ExportQuerier{db: db}
}

func (q *ExportQuerier) QueryCompanies(ctx context.Context) ([]ports.ExportCompanyRow, error) {
	rows, err := q.db.QueryContext(ctx, `SELECT id, slug, name, folder_path, created_at, updated_at FROM companies ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("querying companies: %w", err)
	}
	defer rows.Close()

	var result []ports.ExportCompanyRow
	for rows.Next() {
		var r ports.ExportCompanyRow
		if err := rows.Scan(&r.ID, &r.Slug, &r.Name, &r.FolderPath, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning company: %w", err)
		}
		result = append(result, r)
	}
	return result, rows.Err()
}

func (q *ExportQuerier) QueryRoles(ctx context.Context) ([]ports.ExportRoleRow, error) {
	rows, err := q.db.QueryContext(ctx, `SELECT id, company_id, slug, title, status, folder_path, created_at, updated_at FROM roles ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("querying roles: %w", err)
	}
	defer rows.Close()

	var result []ports.ExportRoleRow
	for rows.Next() {
		var r ports.ExportRoleRow
		if err := rows.Scan(&r.ID, &r.CompanyID, &r.Slug, &r.Title, &r.Status, &r.FolderPath, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning role: %w", err)
		}
		result = append(result, r)
	}
	return result, rows.Err()
}

func (q *ExportQuerier) QueryContacts(ctx context.Context) ([]ports.ExportContactRow, error) {
	rows, err := q.db.QueryContext(ctx, `SELECT id, name, org, linkedin_url, email, created_at, updated_at FROM contacts ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("querying contacts: %w", err)
	}
	defer rows.Close()

	var result []ports.ExportContactRow
	for rows.Next() {
		var r ports.ExportContactRow
		if err := rows.Scan(&r.ID, &r.Name, &r.Org, &r.LinkedInURL, &r.Email, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning contact: %w", err)
		}
		result = append(result, r)
	}
	return result, rows.Err()
}

func (q *ExportQuerier) QueryMeetings(ctx context.Context) ([]ports.ExportMeetingRow, error) {
	rows, err := q.db.QueryContext(ctx, `SELECT id, occurred_at, title, role_id, contact_id, path_md, created_at, updated_at FROM meetings ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("querying meetings: %w", err)
	}
	defer rows.Close()

	var result []ports.ExportMeetingRow
	for rows.Next() {
		var r ports.ExportMeetingRow
		var roleID, contactID sql.NullString
		if err := rows.Scan(&r.ID, &r.OccurredAt, &r.Title, &roleID, &contactID, &r.PathMD, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning meeting: %w", err)
		}
		if roleID.Valid {
			r.RoleID = &roleID.String
		}
		if contactID.Valid {
			r.ContactID = &contactID.String
		}
		result = append(result, r)
	}
	return result, rows.Err()
}

func (q *ExportQuerier) QueryJobDescriptions(ctx context.Context) ([]ports.ExportJobDescriptionRow, error) {
	rows, err := q.db.QueryContext(ctx, `SELECT role_id, path_html, path_pdf FROM role_job_descriptions ORDER BY role_id`)
	if err != nil {
		return nil, fmt.Errorf("querying job_descriptions: %w", err)
	}
	defer rows.Close()

	var result []ports.ExportJobDescriptionRow
	for rows.Next() {
		var r ports.ExportJobDescriptionRow
		var pathHTML, pathPDF sql.NullString
		if err := rows.Scan(&r.RoleID, &pathHTML, &pathPDF); err != nil {
			return nil, fmt.Errorf("scanning job_description: %w", err)
		}
		if pathHTML.Valid {
			r.PathHTML = &pathHTML.String
		}
		if pathPDF.Valid {
			r.PathPDF = &pathPDF.String
		}
		result = append(result, r)
	}
	return result, rows.Err()
}

func (q *ExportQuerier) QueryResumes(ctx context.Context) ([]ports.ExportResumeRow, error) {
	rows, err := q.db.QueryContext(ctx, `SELECT role_id, path_json, path_pdf FROM role_resumes ORDER BY role_id`)
	if err != nil {
		return nil, fmt.Errorf("querying resumes: %w", err)
	}
	defer rows.Close()

	var result []ports.ExportResumeRow
	for rows.Next() {
		var r ports.ExportResumeRow
		var pathJSON, pathPDF sql.NullString
		if err := rows.Scan(&r.RoleID, &pathJSON, &pathPDF); err != nil {
			return nil, fmt.Errorf("scanning resume: %w", err)
		}
		if pathJSON.Valid {
			r.PathJSON = &pathJSON.String
		}
		if pathPDF.Valid {
			r.PathPDF = &pathPDF.String
		}
		result = append(result, r)
	}
	return result, rows.Err()
}

func (q *ExportQuerier) QueryRoleArtifacts(ctx context.Context) ([]ports.ExportRoleArtifactRow, error) {
	rows, err := q.db.QueryContext(ctx, `SELECT id, role_id, name, type, path, created_at, updated_at FROM role_artifacts ORDER BY role_id, name`)
	if err != nil {
		return nil, fmt.Errorf("querying role_artifacts: %w", err)
	}
	defer rows.Close()

	var result []ports.ExportRoleArtifactRow
	for rows.Next() {
		var r ports.ExportRoleArtifactRow
		if err := rows.Scan(&r.ID, &r.RoleID, &r.Name, &r.Type, &r.Path, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning role_artifact: %w", err)
		}
		result = append(result, r)
	}
	return result, rows.Err()
}
