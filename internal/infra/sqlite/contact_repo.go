package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"jobtracker/internal/domain"
)

// ContactRepo implements ports.ContactRepository
type ContactRepo struct {
	db *DB
}

// NewContactRepo creates a new ContactRepo
func NewContactRepo(db *DB) *ContactRepo {
	return &ContactRepo{db: db}
}

// Create inserts a new contact
func (r *ContactRepo) Create(ctx context.Context, contact *domain.Contact) error {
	now := time.Now()
	contact.CreatedAt = now
	contact.UpdatedAt = now

	_, err := r.db.ExecContext(ctx,
		`INSERT INTO contacts (id, name, org, linkedin_url, email, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		contact.ID, contact.Name, contact.Org, contact.LinkedInURL, contact.Email,
		contact.CreatedAt, contact.UpdatedAt)

	if err != nil {
		return fmt.Errorf("inserting contact: %w", err)
	}
	return nil
}

// GetByID retrieves a contact by ID
func (r *ContactRepo) GetByID(ctx context.Context, id string) (*domain.Contact, error) {
	contact := &domain.Contact{}
	err := r.db.QueryRowContext(ctx,
		`SELECT id, name, org, linkedin_url, email, created_at, updated_at
		 FROM contacts WHERE id = ?`, id).Scan(
		&contact.ID, &contact.Name, &contact.Org, &contact.LinkedInURL, &contact.Email,
		&contact.CreatedAt, &contact.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("querying contact by id: %w", err)
	}
	return contact, nil
}

// List retrieves all contacts ordered by name
func (r *ContactRepo) List(ctx context.Context) ([]*domain.Contact, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, name, org, linkedin_url, email, created_at, updated_at
		 FROM contacts ORDER BY name ASC`)
	if err != nil {
		return nil, fmt.Errorf("listing contacts: %w", err)
	}
	defer rows.Close()

	var contacts []*domain.Contact
	for rows.Next() {
		c := &domain.Contact{}
		if err := rows.Scan(&c.ID, &c.Name, &c.Org, &c.LinkedInURL, &c.Email,
			&c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning contact: %w", err)
		}
		contacts = append(contacts, c)
	}

	return contacts, rows.Err()
}
