package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"jobtracker/internal/domain"
)

// RoleArtifactRepo implements ports.RoleArtifactRepository
type RoleArtifactRepo struct {
	db *DB
}

// NewRoleArtifactRepo creates a new RoleArtifactRepo
func NewRoleArtifactRepo(db *DB) *RoleArtifactRepo {
	return &RoleArtifactRepo{db: db}
}

// Upsert creates or updates an artifact by (role_id, name).
// Returns the artifact with its ID (stable if existing, new if created).
func (r *RoleArtifactRepo) Upsert(ctx context.Context, artifact *domain.RoleArtifact) (*domain.RoleArtifact, error) {
	now := time.Now()

	// Check if artifact with same role_id and name exists
	existing, err := r.GetByName(ctx, artifact.RoleID, artifact.Name)
	if err != nil {
		return nil, fmt.Errorf("checking existing artifact: %w", err)
	}

	if existing != nil {
		// Update existing record
		_, err := r.db.ExecContext(ctx,
			`UPDATE role_artifacts SET type = ?, path = ?, updated_at = ? WHERE id = ?`,
			artifact.Type, artifact.Path, now, existing.ID)
		if err != nil {
			return nil, fmt.Errorf("updating artifact: %w", err)
		}
		existing.Type = artifact.Type
		existing.Path = artifact.Path
		existing.UpdatedAt = now
		return existing, nil
	}

	// Insert new record
	artifact.ID = uuid.New().String()
	artifact.CreatedAt = now
	artifact.UpdatedAt = now

	_, err = r.db.ExecContext(ctx,
		`INSERT INTO role_artifacts (id, role_id, name, type, path, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		artifact.ID, artifact.RoleID, artifact.Name, artifact.Type, artifact.Path,
		artifact.CreatedAt, artifact.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("inserting artifact: %w", err)
	}

	return artifact, nil
}

// List returns all artifacts for a role, ordered by name ASC
func (r *RoleArtifactRepo) List(ctx context.Context, roleID string) ([]*domain.RoleArtifact, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, role_id, name, type, path, created_at, updated_at
		 FROM role_artifacts WHERE role_id = ? ORDER BY name ASC`, roleID)
	if err != nil {
		return nil, fmt.Errorf("listing artifacts: %w", err)
	}
	defer rows.Close()

	var artifacts []*domain.RoleArtifact
	for rows.Next() {
		a := &domain.RoleArtifact{}
		if err := rows.Scan(&a.ID, &a.RoleID, &a.Name, &a.Type, &a.Path,
			&a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning artifact: %w", err)
		}
		artifacts = append(artifacts, a)
	}

	return artifacts, rows.Err()
}

// GetByName retrieves an artifact by role_id and name
func (r *RoleArtifactRepo) GetByName(ctx context.Context, roleID, name string) (*domain.RoleArtifact, error) {
	a := &domain.RoleArtifact{}
	err := r.db.QueryRowContext(ctx,
		`SELECT id, role_id, name, type, path, created_at, updated_at
		 FROM role_artifacts WHERE role_id = ? AND name = ?`, roleID, name).Scan(
		&a.ID, &a.RoleID, &a.Name, &a.Type, &a.Path, &a.CreatedAt, &a.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("querying artifact by name: %w", err)
	}
	return a, nil
}

// Delete removes an artifact by role_id and name
func (r *RoleArtifactRepo) Delete(ctx context.Context, roleID, name string) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM role_artifacts WHERE role_id = ? AND name = ?`, roleID, name)
	if err != nil {
		return fmt.Errorf("deleting artifact: %w", err)
	}
	return nil
}
