package app

import (
	"context"
	"fmt"

	"jobtracker/internal/domain"
	"jobtracker/internal/ports"

	"github.com/google/uuid"
)

// CompanyService handles company-related business logic
type CompanyService struct {
	companyRepo ports.CompanyRepository
	roleRepo    ports.RoleRepository
	fileStore   ports.FileStore
}

// NewCompanyService creates a new CompanyService
func NewCompanyService(
	companyRepo ports.CompanyRepository,
	roleRepo ports.RoleRepository,
	fileStore ports.FileStore,
) *CompanyService {
	return &CompanyService{
		companyRepo: companyRepo,
		roleRepo:    roleRepo,
		fileStore:   fileStore,
	}
}

// CreateCompanyInput is the input for creating a company
type CreateCompanyInput struct {
	Slug string
	Name string
}

// CreateCompany creates a new company with filesystem scaffolding
func (s *CompanyService) CreateCompany(ctx context.Context, input CreateCompanyInput) (*domain.Company, error) {
	// Check if company already exists
	existing, err := s.companyRepo.GetBySlug(ctx, input.Slug)
	if err != nil {
		return nil, fmt.Errorf("checking existing company: %w", err)
	}
	if existing != nil {
		return nil, fmt.Errorf("company with slug '%s' already exists", input.Slug)
	}

	// Create filesystem structure
	folderPath, err := s.fileStore.CreateCompanyFolder(ctx, input.Slug)
	if err != nil {
		return nil, fmt.Errorf("creating company folder: %w", err)
	}

	// Create company record
	company := &domain.Company{
		ID:         uuid.New().String(),
		Slug:       input.Slug,
		Name:       input.Name,
		FolderPath: folderPath,
	}

	if err := s.companyRepo.Create(ctx, company); err != nil {
		return nil, fmt.Errorf("creating company record: %w", err)
	}

	return company, nil
}

// CreateRoleInput is the input for creating a role
type CreateRoleInput struct {
	CompanySlug string
	Slug        string
	Title       string
}

// CreateRole creates a new role under a company
func (s *CompanyService) CreateRole(ctx context.Context, input CreateRoleInput) (*domain.Role, error) {
	// Get company
	company, err := s.companyRepo.GetBySlug(ctx, input.CompanySlug)
	if err != nil {
		return nil, fmt.Errorf("getting company: %w", err)
	}
	if company == nil {
		return nil, fmt.Errorf("company '%s' not found", input.CompanySlug)
	}

	// Check if role already exists
	existing, err := s.roleRepo.GetBySlug(ctx, company.ID, input.Slug)
	if err != nil {
		return nil, fmt.Errorf("checking existing role: %w", err)
	}
	if existing != nil {
		return nil, fmt.Errorf("role with slug '%s' already exists for company '%s'", input.Slug, input.CompanySlug)
	}

	// Create filesystem structure
	folderPath, err := s.fileStore.CreateRoleFolder(ctx, input.CompanySlug, input.Slug)
	if err != nil {
		return nil, fmt.Errorf("creating role folder: %w", err)
	}

	// Create role record
	role := &domain.Role{
		ID:         uuid.New().String(),
		CompanyID:  company.ID,
		Slug:       input.Slug,
		Title:      input.Title,
		FolderPath: folderPath,
	}

	if err := s.roleRepo.Create(ctx, role); err != nil {
		return nil, fmt.Errorf("creating role record: %w", err)
	}

	return role, nil
}

// CompanyWithDetails represents a company with its roles and computed status
type CompanyWithDetails struct {
	Company *domain.Company
	Roles   []*domain.Role
	Status  domain.CompanyStatus
}

// ListCompanies returns all companies with computed status
func (s *CompanyService) ListCompanies(ctx context.Context) ([]*CompanyWithDetails, error) {
	companies, err := s.companyRepo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing companies: %w", err)
	}

	result := make([]*CompanyWithDetails, 0, len(companies))
	for _, c := range companies {
		// Load roles to compute status
		roles, err := s.roleRepo.ListByCompany(ctx, c.ID)
		if err != nil {
			return nil, fmt.Errorf("listing roles for company %s: %w", c.Slug, err)
		}

		result = append(result, &CompanyWithDetails{
			Company: c,
			Roles:   roles,
			Status:  domain.ComputeCompanyStatus(roles),
		})
	}

	return result, nil
}

// GetCompany returns a company by slug with its roles and computed status
func (s *CompanyService) GetCompany(ctx context.Context, slug string) (*CompanyWithDetails, error) {
	company, err := s.companyRepo.GetBySlug(ctx, slug)
	if err != nil {
		return nil, fmt.Errorf("getting company: %w", err)
	}
	if company == nil {
		return nil, nil
	}

	roles, err := s.roleRepo.ListByCompany(ctx, company.ID)
	if err != nil {
		return nil, fmt.Errorf("listing roles: %w", err)
	}

	return &CompanyWithDetails{
		Company: company,
		Roles:   roles,
		Status:  domain.ComputeCompanyStatus(roles),
	}, nil
}

// UpdateRoleStatusInput is the input for updating a role's status
type UpdateRoleStatusInput struct {
	CompanySlug string
	RoleSlug    string
	Status      string
}

// UpdateRoleStatus updates the status of a role
func (s *CompanyService) UpdateRoleStatus(ctx context.Context, input UpdateRoleStatusInput) error {
	// Get company
	company, err := s.companyRepo.GetBySlug(ctx, input.CompanySlug)
	if err != nil {
		return fmt.Errorf("getting company: %w", err)
	}
	if company == nil {
		return fmt.Errorf("company '%s' not found", input.CompanySlug)
	}

	// Get role
	role, err := s.roleRepo.GetBySlug(ctx, company.ID, input.RoleSlug)
	if err != nil {
		return fmt.Errorf("getting role: %w", err)
	}
	if role == nil {
		return fmt.Errorf("role '%s' not found for company '%s'", input.RoleSlug, input.CompanySlug)
	}

	// Parse and validate status
	status, err := domain.ParseRoleStatus(input.Status)
	if err != nil {
		return fmt.Errorf("invalid status: %w", err)
	}

	// Update status in repository
	if err := s.roleRepo.UpdateStatus(ctx, role.ID, status); err != nil {
		return fmt.Errorf("updating role status: %w", err)
	}

	return nil
}
