package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"jobtracker/internal/domain"
)

// ThreadRepo implements ports.ThreadRepository
type ThreadRepo struct {
	db *DB
}

// NewThreadRepo creates a new ThreadRepo
func NewThreadRepo(db *DB) *ThreadRepo {
	return &ThreadRepo{db: db}
}

// Create inserts a new thread
func (r *ThreadRepo) Create(ctx context.Context, thread *domain.Thread) error {
	now := time.Now()
	thread.CreatedAt = now
	thread.UpdatedAt = now

	var contactID interface{}
	if thread.ContactID != "" {
		contactID = thread.ContactID
	}

	_, err := r.db.ExecContext(ctx,
		`INSERT INTO threads (id, title, contact_id, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?)`,
		thread.ID, thread.Title, contactID,
		thread.CreatedAt, thread.UpdatedAt)

	if err != nil {
		return fmt.Errorf("inserting thread: %w", err)
	}
	return nil
}

// GetByID retrieves a thread by ID
func (r *ThreadRepo) GetByID(ctx context.Context, id string) (*domain.Thread, error) {
	thread := &domain.Thread{}
	var contactID sql.NullString

	err := r.db.QueryRowContext(ctx,
		`SELECT id, title, contact_id, created_at, updated_at
		 FROM threads WHERE id = ?`, id).Scan(
		&thread.ID, &thread.Title, &contactID,
		&thread.CreatedAt, &thread.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("querying thread by id: %w", err)
	}

	if contactID.Valid {
		thread.ContactID = contactID.String
	}

	return thread, nil
}

// List retrieves all threads ordered by created_at desc
func (r *ThreadRepo) List(ctx context.Context) ([]*domain.Thread, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, title, contact_id, created_at, updated_at
		 FROM threads ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("listing threads: %w", err)
	}
	defer rows.Close()

	var threads []*domain.Thread
	for rows.Next() {
		t := &domain.Thread{}
		var contactID sql.NullString
		if err := rows.Scan(&t.ID, &t.Title, &contactID,
			&t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning thread: %w", err)
		}
		if contactID.Valid {
			t.ContactID = contactID.String
		}
		threads = append(threads, t)
	}

	return threads, rows.Err()
}

// LinkRole links a thread to a role (idempotent - ignores if already exists)
func (r *ThreadRepo) LinkRole(ctx context.Context, threadID, roleID string) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT OR IGNORE INTO thread_roles (thread_id, role_id) VALUES (?, ?)`,
		threadID, roleID)
	if err != nil {
		return fmt.Errorf("linking thread to role: %w", err)
	}
	return nil
}

// GetLinkedRoles returns all roles linked to a thread
func (r *ThreadRepo) GetLinkedRoles(ctx context.Context, threadID string) ([]*domain.Role, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT r.id, r.company_id, r.slug, r.title, r.status, r.folder_path, r.created_at, r.updated_at
		 FROM roles r
		 INNER JOIN thread_roles tr ON tr.role_id = r.id
		 WHERE tr.thread_id = ?
		 ORDER BY r.created_at ASC, r.id ASC`, threadID)
	if err != nil {
		return nil, fmt.Errorf("getting linked roles: %w", err)
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
