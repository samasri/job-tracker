package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"jobtracker/internal/domain"
)

// roleRepo implements ports.RoleRepository
type roleRepo struct {
	db *DB
}

// NewRoleRepo creates a new roleRepo
func NewRoleRepo(db *DB) *roleRepo {
	return &roleRepo{db: db}
}

// Create inserts a new role
func (r *roleRepo) Create(ctx context.Context, role *domain.Role) error {
	now := time.Now()
	role.CreatedAt = now
	role.UpdatedAt = now

	// Default status if not set
	if role.Status == "" {
		role.Status = domain.RoleStatusRecruiterReachedOut
	}

	_, err := r.db.ExecContext(ctx,
		`INSERT INTO roles (id, company_id, slug, title, status, folder_path, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		role.ID, role.CompanyID, role.Slug, role.Title, role.Status, role.FolderPath,
		role.CreatedAt, role.UpdatedAt)

	if err != nil {
		return fmt.Errorf("inserting role: %w", err)
	}
	return nil
}

// GetBySlug retrieves a role by company ID and slug
func (r *roleRepo) GetBySlug(ctx context.Context, companyID, slug string) (*domain.Role, error) {
	role := &domain.Role{}
	err := r.db.QueryRowContext(ctx,
		`SELECT id, company_id, slug, title, status, folder_path, created_at, updated_at
		 FROM roles WHERE company_id = ? AND slug = ?`, companyID, slug).Scan(
		&role.ID, &role.CompanyID, &role.Slug, &role.Title, &role.Status, &role.FolderPath,
		&role.CreatedAt, &role.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("querying role by slug: %w", err)
	}
	return role, nil
}

// GetByID retrieves a role by ID
func (r *roleRepo) GetByID(ctx context.Context, id string) (*domain.Role, error) {
	role := &domain.Role{}
	err := r.db.QueryRowContext(ctx,
		`SELECT id, company_id, slug, title, status, folder_path, created_at, updated_at
		 FROM roles WHERE id = ?`, id).Scan(
		&role.ID, &role.CompanyID, &role.Slug, &role.Title, &role.Status, &role.FolderPath,
		&role.CreatedAt, &role.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("querying role by id: %w", err)
	}
	return role, nil
}

// ListByCompany retrieves all roles for a company ordered by created_at ASC, id ASC
func (r *roleRepo) ListByCompany(ctx context.Context, companyID string) ([]*domain.Role, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, company_id, slug, title, status, folder_path, created_at, updated_at
		 FROM roles WHERE company_id = ? ORDER BY created_at ASC, id ASC`, companyID)
	if err != nil {
		return nil, fmt.Errorf("listing roles: %w", err)
	}
	defer rows.Close()

	var roles []*domain.Role
	for rows.Next() {
		role := &domain.Role{}
		if err := rows.Scan(&role.ID, &role.CompanyID, &role.Slug, &role.Title, &role.Status,
			&role.FolderPath, &role.CreatedAt, &role.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning role: %w", err)
		}
		roles = append(roles, role)
	}

	return roles, rows.Err()
}

// UpdateStatus updates the status of a role
func (r *roleRepo) UpdateStatus(ctx context.Context, roleID string, status domain.RoleStatus) error {
	result, err := r.db.ExecContext(ctx,
		`UPDATE roles SET status = ?, updated_at = ? WHERE id = ?`,
		status, time.Now(), roleID)
	if err != nil {
		return fmt.Errorf("updating role status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("role not found: %s", roleID)
	}

	return nil
}
