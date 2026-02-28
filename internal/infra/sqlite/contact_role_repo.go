package sqlite

import (
	"context"
	"fmt"

	"jobtracker/internal/domain"
)

// ContactRoleRepo implements ports.ContactRoleRepository
type ContactRoleRepo struct {
	db *DB
}

// NewContactRoleRepo creates a new ContactRoleRepo
func NewContactRoleRepo(db *DB) *ContactRoleRepo {
	return &ContactRoleRepo{db: db}
}

// LinkRole links a contact to a role (idempotent via INSERT OR IGNORE)
func (r *ContactRoleRepo) LinkRole(ctx context.Context, contactID, roleID string) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT OR IGNORE INTO contact_roles (contact_id, role_id) VALUES (?, ?)`,
		contactID, roleID)
	if err != nil {
		return fmt.Errorf("linking contact to role: %w", err)
	}
	return nil
}

// GetLinkedRoles returns all roles linked to a contact, ordered by created_at ASC, role_id ASC
func (r *ContactRoleRepo) GetLinkedRoles(ctx context.Context, contactID string) ([]*domain.Role, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT ro.id, ro.company_id, ro.slug, ro.title, ro.status, ro.folder_path, ro.created_at, ro.updated_at
		 FROM roles ro
		 INNER JOIN contact_roles cr ON cr.role_id = ro.id
		 WHERE cr.contact_id = ?
		 ORDER BY cr.created_at ASC, ro.id ASC`, contactID)
	if err != nil {
		return nil, fmt.Errorf("getting linked roles for contact: %w", err)
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
