package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"jobtracker/internal/domain"
)

// contactRepo implements ports.ContactRepository
type contactRepo struct {
	db *DB
}

// NewContactRepo creates a new contactRepo
func NewContactRepo(db *DB) *contactRepo {
	return &contactRepo{db: db}
}

// Create inserts a new contact
func (r *contactRepo) Create(ctx context.Context, contact *domain.Contact) error {
	now := time.Now()
	contact.CreatedAt = now
	contact.UpdatedAt = now

	var code, slug, folderPath interface{}
	if contact.Code != "" {
		code = contact.Code
	}
	if contact.Slug != "" {
		slug = contact.Slug
	}
	if contact.FolderPath != "" {
		folderPath = contact.FolderPath
	}

	_, err := r.db.ExecContext(ctx,
		`INSERT INTO contacts (id, name, org, linkedin_url, email, code, slug, folder_path, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		contact.ID, contact.Name, contact.Org, contact.LinkedInURL, contact.Email,
		code, slug, folderPath, contact.CreatedAt, contact.UpdatedAt)

	if err != nil {
		return fmt.Errorf("inserting contact: %w", err)
	}
	return nil
}

// GetByID retrieves a contact by ID
func (r *contactRepo) GetByID(ctx context.Context, id string) (*domain.Contact, error) {
	contact := &domain.Contact{}
	var code, slug, folderPath sql.NullString

	err := r.db.QueryRowContext(ctx,
		`SELECT id, name, org, linkedin_url, email, code, slug, folder_path, created_at, updated_at
		 FROM contacts WHERE id = ?`, id).Scan(
		&contact.ID, &contact.Name, &contact.Org, &contact.LinkedInURL, &contact.Email,
		&code, &slug, &folderPath, &contact.CreatedAt, &contact.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("querying contact by id: %w", err)
	}

	if code.Valid {
		contact.Code = code.String
	}
	if slug.Valid {
		contact.Slug = slug.String
	}
	if folderPath.Valid {
		contact.FolderPath = folderPath.String
	}

	return contact, nil
}

// GetBySlug retrieves a contact by slug
func (r *contactRepo) GetBySlug(ctx context.Context, slug string) (*domain.Contact, error) {
	contact := &domain.Contact{}
	var code, folderPath sql.NullString

	err := r.db.QueryRowContext(ctx,
		`SELECT id, name, org, linkedin_url, email, code, slug, folder_path, created_at, updated_at
		 FROM contacts WHERE slug = ?`, slug).Scan(
		&contact.ID, &contact.Name, &contact.Org, &contact.LinkedInURL, &contact.Email,
		&code, &contact.Slug, &folderPath, &contact.CreatedAt, &contact.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("querying contact by slug: %w", err)
	}

	if code.Valid {
		contact.Code = code.String
	}
	if folderPath.Valid {
		contact.FolderPath = folderPath.String
	}

	return contact, nil
}

// List retrieves all contacts ordered by creation date descending
func (r *contactRepo) List(ctx context.Context) ([]*domain.Contact, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, name, org, linkedin_url, email, code, slug, folder_path, created_at, updated_at
		 FROM contacts ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("listing contacts: %w", err)
	}
	defer rows.Close()

	var contacts []*domain.Contact
	for rows.Next() {
		c := &domain.Contact{}
		var code, slug, folderPath sql.NullString
		if err := rows.Scan(&c.ID, &c.Name, &c.Org, &c.LinkedInURL, &c.Email,
			&code, &slug, &folderPath, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning contact: %w", err)
		}
		if code.Valid {
			c.Code = code.String
		}
		if slug.Valid {
			c.Slug = slug.String
		}
		if folderPath.Valid {
			c.FolderPath = folderPath.String
		}
		contacts = append(contacts, c)
	}

	return contacts, rows.Err()
}

// UpdateCodeSlug updates the code, slug, and folder_path for a contact
func (r *contactRepo) UpdateCodeSlug(ctx context.Context, id, code, slug, folderPath string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE contacts SET code = ?, slug = ?, folder_path = ?, updated_at = ? WHERE id = ?`,
		code, slug, folderPath, time.Now(), id)
	if err != nil {
		return fmt.Errorf("updating contact code/slug: %w", err)
	}
	return nil
}

// CodeExists checks if a code already exists among contacts
func (r *contactRepo) CodeExists(ctx context.Context, code string) (bool, error) {
	var count int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM contacts WHERE code = ?`, code).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("checking contact code existence: %w", err)
	}
	return count > 0, nil
}
