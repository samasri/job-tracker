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
		`INSERT INTO threads (id, code, slug, title, contact_id, folder_path, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		thread.ID, thread.Code, thread.Slug, thread.Title, contactID, thread.FolderPath,
		thread.CreatedAt, thread.UpdatedAt)

	if err != nil {
		return fmt.Errorf("inserting thread: %w", err)
	}
	return nil
}

// GetByID retrieves a thread by ID
func (r *ThreadRepo) GetByID(ctx context.Context, id string) (*domain.Thread, error) {
	thread := &domain.Thread{}
	var contactID, code, slug, folderPath sql.NullString

	err := r.db.QueryRowContext(ctx,
		`SELECT id, code, slug, title, contact_id, folder_path, created_at, updated_at
		 FROM threads WHERE id = ?`, id).Scan(
		&thread.ID, &code, &slug, &thread.Title, &contactID, &folderPath,
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
	if code.Valid {
		thread.Code = code.String
	}
	if slug.Valid {
		thread.Slug = slug.String
	}
	if folderPath.Valid {
		thread.FolderPath = folderPath.String
	}

	return thread, nil
}

// GetBySlug retrieves a thread by slug
func (r *ThreadRepo) GetBySlug(ctx context.Context, slug string) (*domain.Thread, error) {
	thread := &domain.Thread{}
	var contactID, code, folderPath sql.NullString

	err := r.db.QueryRowContext(ctx,
		`SELECT id, code, slug, title, contact_id, folder_path, created_at, updated_at
		 FROM threads WHERE slug = ?`, slug).Scan(
		&thread.ID, &code, &thread.Slug, &thread.Title, &contactID, &folderPath,
		&thread.CreatedAt, &thread.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("querying thread by slug: %w", err)
	}

	if contactID.Valid {
		thread.ContactID = contactID.String
	}
	if code.Valid {
		thread.Code = code.String
	}
	if folderPath.Valid {
		thread.FolderPath = folderPath.String
	}

	return thread, nil
}

// List retrieves all threads ordered by created_at desc
func (r *ThreadRepo) List(ctx context.Context) ([]*domain.Thread, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, code, slug, title, contact_id, folder_path, created_at, updated_at
		 FROM threads ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("listing threads: %w", err)
	}
	defer rows.Close()

	var threads []*domain.Thread
	for rows.Next() {
		t := &domain.Thread{}
		var contactID, code, slug, folderPath sql.NullString
		if err := rows.Scan(&t.ID, &code, &slug, &t.Title, &contactID, &folderPath,
			&t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning thread: %w", err)
		}
		if contactID.Valid {
			t.ContactID = contactID.String
		}
		if code.Valid {
			t.Code = code.String
		}
		if slug.Valid {
			t.Slug = slug.String
		}
		if folderPath.Valid {
			t.FolderPath = folderPath.String
		}
		threads = append(threads, t)
	}

	return threads, rows.Err()
}

// UpdateCodeSlug updates the code, slug, and folder_path for a thread
func (r *ThreadRepo) UpdateCodeSlug(ctx context.Context, threadID, code, slug, folderPath string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE threads SET code = ?, slug = ?, folder_path = ?, updated_at = ? WHERE id = ?`,
		code, slug, folderPath, time.Now(), threadID)
	if err != nil {
		return fmt.Errorf("updating thread code/slug: %w", err)
	}
	return nil
}

// CodeExists checks if a code already exists
func (r *ThreadRepo) CodeExists(ctx context.Context, code string) (bool, error) {
	var count int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM threads WHERE code = ?`, code).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("checking code existence: %w", err)
	}
	return count > 0, nil
}
