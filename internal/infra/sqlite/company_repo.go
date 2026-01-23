package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"jobtracker/internal/domain"
)

// CompanyRepo implements ports.CompanyRepository
type CompanyRepo struct {
	db *DB
}

// NewCompanyRepo creates a new CompanyRepo
func NewCompanyRepo(db *DB) *CompanyRepo {
	return &CompanyRepo{db: db}
}

// Create inserts a new company
func (r *CompanyRepo) Create(ctx context.Context, company *domain.Company) error {
	now := time.Now()
	company.CreatedAt = now
	company.UpdatedAt = now

	_, err := r.db.ExecContext(ctx,
		`INSERT INTO companies (id, slug, name, folder_path, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		company.ID, company.Slug, company.Name, company.FolderPath,
		company.CreatedAt, company.UpdatedAt)

	if err != nil {
		return fmt.Errorf("inserting company: %w", err)
	}
	return nil
}

// GetBySlug retrieves a company by slug
func (r *CompanyRepo) GetBySlug(ctx context.Context, slug string) (*domain.Company, error) {
	company := &domain.Company{}
	err := r.db.QueryRowContext(ctx,
		`SELECT id, slug, name, folder_path, created_at, updated_at
		 FROM companies WHERE slug = ?`, slug).Scan(
		&company.ID, &company.Slug, &company.Name, &company.FolderPath,
		&company.CreatedAt, &company.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("querying company by slug: %w", err)
	}
	return company, nil
}

// GetByID retrieves a company by ID
func (r *CompanyRepo) GetByID(ctx context.Context, id string) (*domain.Company, error) {
	company := &domain.Company{}
	err := r.db.QueryRowContext(ctx,
		`SELECT id, slug, name, folder_path, created_at, updated_at
		 FROM companies WHERE id = ?`, id).Scan(
		&company.ID, &company.Slug, &company.Name, &company.FolderPath,
		&company.CreatedAt, &company.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("querying company by id: %w", err)
	}
	return company, nil
}

// List retrieves all companies ordered by name
func (r *CompanyRepo) List(ctx context.Context) ([]*domain.Company, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, slug, name, folder_path, created_at, updated_at
		 FROM companies ORDER BY name ASC`)
	if err != nil {
		return nil, fmt.Errorf("listing companies: %w", err)
	}
	defer rows.Close()

	var companies []*domain.Company
	for rows.Next() {
		c := &domain.Company{}
		if err := rows.Scan(&c.ID, &c.Slug, &c.Name, &c.FolderPath,
			&c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning company: %w", err)
		}
		companies = append(companies, c)
	}

	return companies, rows.Err()
}
