package app

import (
	"context"
	"fmt"
	"io"

	"jobtracker/internal/domain"
	"jobtracker/internal/ports"
)

// JDService handles job description-related business logic
type JDService struct {
	jdRepo      ports.JobDescriptionRepository
	companyRepo ports.CompanyRepository
	roleRepo    ports.RoleRepository
	fileStore   ports.FileStore
}

// NewJDService creates a new JDService
func NewJDService(
	jdRepo ports.JobDescriptionRepository,
	companyRepo ports.CompanyRepository,
	roleRepo ports.RoleRepository,
	fileStore ports.FileStore,
) *JDService {
	return &JDService{
		jdRepo:      jdRepo,
		companyRepo: companyRepo,
		roleRepo:    roleRepo,
		fileStore:   fileStore,
	}
}

// AttachJDInput is the input for attaching a JD to a role
type AttachJDInput struct {
	CompanySlug string
	RoleSlug    string
	HTMLContent string
	PDFContent  io.Reader
}

// AttachJD attaches a job description to a role
func (s *JDService) AttachJD(ctx context.Context, input AttachJDInput) (*domain.RoleJobDescription, error) {
	// Get company
	company, err := s.companyRepo.GetBySlug(ctx, input.CompanySlug)
	if err != nil {
		return nil, fmt.Errorf("getting company: %w", err)
	}
	if company == nil {
		return nil, fmt.Errorf("company '%s' not found", input.CompanySlug)
	}

	// Get role
	role, err := s.roleRepo.GetBySlug(ctx, company.ID, input.RoleSlug)
	if err != nil {
		return nil, fmt.Errorf("getting role: %w", err)
	}
	if role == nil {
		return nil, fmt.Errorf("role '%s' not found for company '%s'", input.RoleSlug, input.CompanySlug)
	}

	jd := &domain.RoleJobDescription{RoleID: role.ID}

	// Save HTML if provided
	if input.HTMLContent != "" {
		pathHTML, err := s.fileStore.SaveJobDescriptionHTML(ctx, input.CompanySlug, input.RoleSlug, input.HTMLContent)
		if err != nil {
			return nil, fmt.Errorf("saving HTML: %w", err)
		}
		jd.PathHTML = pathHTML
	}

	// Save PDF if provided
	if input.PDFContent != nil {
		pathPDF, err := s.fileStore.SaveJobDescriptionPDF(ctx, input.CompanySlug, input.RoleSlug, input.PDFContent)
		if err != nil {
			return nil, fmt.Errorf("saving PDF: %w", err)
		}
		jd.PathPDF = pathPDF
	}

	// Save to database
	if err := s.jdRepo.Save(ctx, jd); err != nil {
		return nil, fmt.Errorf("saving JD record: %w", err)
	}

	return jd, nil
}

// GetJD retrieves a job description for a role by company and role slugs
func (s *JDService) GetJD(ctx context.Context, companySlug, roleSlug string) (*domain.RoleJobDescription, error) {
	// Get company
	company, err := s.companyRepo.GetBySlug(ctx, companySlug)
	if err != nil {
		return nil, fmt.Errorf("getting company: %w", err)
	}
	if company == nil {
		return nil, nil
	}

	// Get role
	role, err := s.roleRepo.GetBySlug(ctx, company.ID, roleSlug)
	if err != nil {
		return nil, fmt.Errorf("getting role: %w", err)
	}
	if role == nil {
		return nil, nil
	}

	return s.jdRepo.GetByRoleID(ctx, role.ID)
}

// GetJobDescription retrieves a job description by role ID
func (s *JDService) GetJobDescription(ctx context.Context, roleID string) (*domain.RoleJobDescription, error) {
	return s.jdRepo.GetByRoleID(ctx, roleID)
}

// ReadJobDescriptionHTML reads the HTML content from a JD file
func (s *JDService) ReadJobDescriptionHTML(ctx context.Context, pathHTML string) (string, error) {
	return s.fileStore.ReadFile(ctx, pathHTML)
}
