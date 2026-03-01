package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"jobtracker/internal/domain"
)

// jobDescriptionRepo implements ports.JobDescriptionRepository
type jobDescriptionRepo struct {
	db *DB
}

// NewJobDescriptionRepo creates a new jobDescriptionRepo
func NewJobDescriptionRepo(db *DB) *jobDescriptionRepo {
	return &jobDescriptionRepo{db: db}
}

// Save upserts a job description record
func (r *jobDescriptionRepo) Save(ctx context.Context, jd *domain.RoleJobDescription) error {
	now := time.Now()

	_, err := r.db.ExecContext(ctx,
		`INSERT INTO role_job_descriptions (role_id, path_html, path_pdf, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?)
		 ON CONFLICT(role_id) DO UPDATE SET
		   path_html = COALESCE(excluded.path_html, path_html),
		   path_pdf = COALESCE(excluded.path_pdf, path_pdf),
		   updated_at = excluded.updated_at`,
		jd.RoleID, nullString(jd.PathHTML), nullString(jd.PathPDF), now, now)

	if err != nil {
		return fmt.Errorf("saving job description: %w", err)
	}
	return nil
}

// GetByRoleID retrieves a job description by role ID
func (r *jobDescriptionRepo) GetByRoleID(ctx context.Context, roleID string) (*domain.RoleJobDescription, error) {
	jd := &domain.RoleJobDescription{RoleID: roleID}
	var pathHTML, pathPDF sql.NullString

	err := r.db.QueryRowContext(ctx,
		`SELECT path_html, path_pdf FROM role_job_descriptions WHERE role_id = ?`, roleID).Scan(
		&pathHTML, &pathPDF)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("querying job description: %w", err)
	}

	jd.PathHTML = pathHTML.String
	jd.PathPDF = pathPDF.String
	return jd, nil
}

func nullString(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}
