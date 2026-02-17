package app

import (
	"context"
	"fmt"
	"io"
	"strings"

	"jobtracker/internal/domain"
	"jobtracker/internal/ports"
)

// ArtifactService handles artifact-related business logic
type ArtifactService struct {
	artifactRepo ports.RoleArtifactRepository
	companyRepo  ports.CompanyRepository
	roleRepo     ports.RoleRepository
	fileStore    ports.FileStore
}

// NewArtifactService creates a new ArtifactService
func NewArtifactService(
	artifactRepo ports.RoleArtifactRepository,
	companyRepo ports.CompanyRepository,
	roleRepo ports.RoleRepository,
	fileStore ports.FileStore,
) *ArtifactService {
	return &ArtifactService{
		artifactRepo: artifactRepo,
		companyRepo:  companyRepo,
		roleRepo:     roleRepo,
		fileStore:    fileStore,
	}
}

// UpsertArtifactInput is the input for creating or updating an artifact
type UpsertArtifactInput struct {
	CompanySlug string
	RoleSlug    string
	Name        string
	Type        string // "pdf", "png", "jsonc", "text", "html", or "markdown"
	TextContent string // For text/jsonc types
	FileContent io.Reader // For pdf type
}

// UpsertArtifact creates or updates an artifact for a role
func (s *ArtifactService) UpsertArtifact(ctx context.Context, input UpsertArtifactInput) (*domain.RoleArtifact, error) {
	// Validate name
	name := strings.TrimSpace(input.Name)
	if err := domain.ValidateArtifactName(name); err != nil {
		return nil, err
	}

	// Validate type
	artifactType, err := domain.ParseArtifactType(input.Type)
	if err != nil {
		return nil, err
	}

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

	// Determine content source and save file
	var contentReader io.Reader
	switch artifactType {
	case domain.ArtifactTypePDF, domain.ArtifactTypePNG:
		if input.FileContent == nil {
			return nil, fmt.Errorf("%s artifacts require file upload", artifactType)
		}
		contentReader = input.FileContent
	case domain.ArtifactTypeJSONC, domain.ArtifactTypeText:
		if strings.TrimSpace(input.TextContent) == "" {
			return nil, fmt.Errorf("%s artifacts require text content", artifactType)
		}
		contentReader = strings.NewReader(input.TextContent)
	}

	// Save file to disk
	path, err := s.fileStore.SaveRoleArtifact(ctx, input.CompanySlug, input.RoleSlug, name, string(artifactType), contentReader)
	if err != nil {
		return nil, fmt.Errorf("saving artifact file: %w", err)
	}

	// Upsert to database
	artifact := &domain.RoleArtifact{
		RoleID: role.ID,
		Name:   name,
		Type:   artifactType,
		Path:   path,
	}
	result, err := s.artifactRepo.Upsert(ctx, artifact)
	if err != nil {
		return nil, fmt.Errorf("saving artifact record: %w", err)
	}

	return result, nil
}

// ListArtifacts retrieves all artifacts for a role
func (s *ArtifactService) ListArtifacts(ctx context.Context, companySlug, roleSlug string) ([]*domain.RoleArtifact, error) {
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

	return s.artifactRepo.List(ctx, role.ID)
}

// GetArtifactByName retrieves an artifact by name for a role
func (s *ArtifactService) GetArtifactByName(ctx context.Context, companySlug, roleSlug, name string) (*domain.RoleArtifact, error) {
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

	return s.artifactRepo.GetByName(ctx, role.ID, name)
}

// ReadArtifactContent reads the content of an artifact file
func (s *ArtifactService) ReadArtifactContent(ctx context.Context, path string) ([]byte, error) {
	return s.fileStore.ReadFileBytes(ctx, path)
}

// DeleteArtifact removes an artifact
func (s *ArtifactService) DeleteArtifact(ctx context.Context, companySlug, roleSlug, name string) error {
	// Get company
	company, err := s.companyRepo.GetBySlug(ctx, companySlug)
	if err != nil {
		return fmt.Errorf("getting company: %w", err)
	}
	if company == nil {
		return fmt.Errorf("company '%s' not found", companySlug)
	}

	// Get role
	role, err := s.roleRepo.GetBySlug(ctx, company.ID, roleSlug)
	if err != nil {
		return fmt.Errorf("getting role: %w", err)
	}
	if role == nil {
		return fmt.Errorf("role '%s' not found for company '%s'", roleSlug, companySlug)
	}

	// Get artifact to find file path
	artifact, err := s.artifactRepo.GetByName(ctx, role.ID, name)
	if err != nil {
		return fmt.Errorf("getting artifact: %w", err)
	}
	if artifact == nil {
		return nil // Already deleted
	}

	// Delete file from disk (ignore errors for missing files)
	_ = s.fileStore.DeleteFile(ctx, artifact.Path)

	// Delete from database
	if err := s.artifactRepo.Delete(ctx, role.ID, name); err != nil {
		return fmt.Errorf("deleting artifact record: %w", err)
	}

	return nil
}
