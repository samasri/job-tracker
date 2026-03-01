package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"jobtracker/internal/domain"
)

// resumeRepo implements ports.ResumeRepository
type resumeRepo struct {
	db *DB
}

// NewResumeRepo creates a new resumeRepo
func NewResumeRepo(db *DB) *resumeRepo {
	return &resumeRepo{db: db}
}

// Save upserts a role resume record
func (r *resumeRepo) Save(ctx context.Context, resume *domain.RoleResume) error {
	now := time.Now()

	_, err := r.db.ExecContext(ctx,
		`INSERT INTO role_resumes (role_id, path_json, path_pdf, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?)
		 ON CONFLICT(role_id) DO UPDATE SET
		   path_json = COALESCE(excluded.path_json, path_json),
		   path_pdf = COALESCE(excluded.path_pdf, path_pdf),
		   updated_at = excluded.updated_at`,
		resume.RoleID, nullString(resume.PathJSON), nullString(resume.PathPDF), now, now)

	if err != nil {
		return fmt.Errorf("saving role resume: %w", err)
	}
	return nil
}

// GetByRoleID retrieves a role resume by role ID
func (r *resumeRepo) GetByRoleID(ctx context.Context, roleID string) (*domain.RoleResume, error) {
	resume := &domain.RoleResume{RoleID: roleID}
	var pathJSON, pathPDF sql.NullString

	err := r.db.QueryRowContext(ctx,
		`SELECT path_json, path_pdf FROM role_resumes WHERE role_id = ?`, roleID).Scan(
		&pathJSON, &pathPDF)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("querying role resume: %w", err)
	}

	resume.PathJSON = pathJSON.String
	resume.PathPDF = pathPDF.String
	return resume, nil
}
