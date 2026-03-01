package app

import (
	"context"
	"fmt"

	"jobtracker/internal/domain"
	"jobtracker/internal/ports"

	"github.com/google/uuid"
)

// ContactService handles contact-related business logic
type ContactService struct {
	contactRepo     ports.ContactRepository
	contactRoleRepo ports.ContactRoleRepository
	companyRepo     ports.CompanyRepository
	roleRepo        ports.RoleRepository
	fileStore       ports.FileStore
}

// NewContactService creates a new ContactService
func NewContactService(
	contactRepo ports.ContactRepository,
	contactRoleRepo ports.ContactRoleRepository,
	companyRepo ports.CompanyRepository,
	roleRepo ports.RoleRepository,
	fileStore ports.FileStore,
) *ContactService {
	return &ContactService{
		contactRepo:     contactRepo,
		contactRoleRepo: contactRoleRepo,
		companyRepo:     companyRepo,
		roleRepo:        roleRepo,
		fileStore:       fileStore,
	}
}

// CreateContactInput is the input for creating a contact
type CreateContactInput struct {
	Name        string
	Org         string
	LinkedInURL string
	Email       string
}

// CreateContact creates a new contact, generating a code+slug+folder at creation time
func (s *ContactService) CreateContact(ctx context.Context, input CreateContactInput) (*domain.Contact, error) {
	code, err := s.generateUniqueContactCode(ctx)
	if err != nil {
		return nil, err
	}

	slug := domain.GenerateContactSlug(input.Name, code)
	folderPath := domain.ContactFolderPath(slug)

	contact := &domain.Contact{
		ID:          uuid.New().String(),
		Name:        input.Name,
		Org:         input.Org,
		LinkedInURL: input.LinkedInURL,
		Email:       input.Email,
		Code:        code,
		Slug:        slug,
		FolderPath:  folderPath,
	}

	if err := s.contactRepo.Create(ctx, contact); err != nil {
		return nil, fmt.Errorf("creating contact: %w", err)
	}

	if _, err := s.fileStore.CreateContactFolder(ctx, slug); err != nil {
		return nil, fmt.Errorf("creating contact folder: %w", err)
	}

	return contact, nil
}

// RoleWithCompany contains a role with its company info
type RoleWithCompany struct {
	Role    *domain.Role
	Company *domain.Company
}

// ContactWithDetails contains a contact with its linked roles and meetings
type ContactWithDetails struct {
	Contact  *domain.Contact
	Roles    []*RoleWithCompany
	Meetings []*domain.MeetingV2
}

// GetContactWithDetails retrieves a contact with its linked roles (with company).
// Meetings must be fetched separately via meetingV2Service.ListMeetingsByContact.
func (s *ContactService) GetContactWithDetails(ctx context.Context, id string) (*ContactWithDetails, error) {
	contact, err := s.contactRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("getting contact: %w", err)
	}
	if contact == nil {
		return nil, nil
	}

	roles, err := s.contactRoleRepo.GetLinkedRoles(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("getting linked roles: %w", err)
	}

	rolesWithCompany := make([]*RoleWithCompany, 0, len(roles))
	for _, role := range roles {
		company, err := s.companyRepo.GetByID(ctx, role.CompanyID)
		if err != nil {
			return nil, fmt.Errorf("getting company for role: %w", err)
		}
		rolesWithCompany = append(rolesWithCompany, &RoleWithCompany{
			Role:    role,
			Company: company,
		})
	}

	return &ContactWithDetails{
		Contact: contact,
		Roles:   rolesWithCompany,
	}, nil
}

// GetContact retrieves a contact by ID
func (s *ContactService) GetContact(ctx context.Context, id string) (*domain.Contact, error) {
	return s.contactRepo.GetByID(ctx, id)
}

// ListContacts returns all contacts
func (s *ContactService) ListContacts(ctx context.Context) ([]*domain.Contact, error) {
	return s.contactRepo.List(ctx)
}

// LinkContactRoleInput is the input for linking a role to a contact
type LinkContactRoleInput struct {
	ContactID   string
	CompanySlug string
	RoleSlug    string
}

// LinkRole links a role to a contact (idempotent)
func (s *ContactService) LinkRole(ctx context.Context, input LinkContactRoleInput) error {
	contact, err := s.contactRepo.GetByID(ctx, input.ContactID)
	if err != nil {
		return fmt.Errorf("getting contact: %w", err)
	}
	if contact == nil {
		return fmt.Errorf("contact '%s' not found", input.ContactID)
	}

	company, err := s.companyRepo.GetBySlug(ctx, input.CompanySlug)
	if err != nil {
		return fmt.Errorf("getting company: %w", err)
	}
	if company == nil {
		return fmt.Errorf("company '%s' not found", input.CompanySlug)
	}

	role, err := s.roleRepo.GetBySlug(ctx, company.ID, input.RoleSlug)
	if err != nil {
		return fmt.Errorf("getting role: %w", err)
	}
	if role == nil {
		return fmt.Errorf("role '%s' not found for company '%s'", input.RoleSlug, input.CompanySlug)
	}

	if err := s.contactRoleRepo.LinkRole(ctx, input.ContactID, role.ID); err != nil {
		return fmt.Errorf("linking role: %w", err)
	}

	return nil
}


func (s *ContactService) generateUniqueContactCode(ctx context.Context) (string, error) {
	for i := 0; i < 10; i++ {
		code, err := domain.NewShortID8()
		if err != nil {
			return "", fmt.Errorf("generating contact code: %w", err)
		}
		exists, err := s.contactRepo.CodeExists(ctx, code)
		if err != nil {
			return "", fmt.Errorf("checking code existence: %w", err)
		}
		if !exists {
			return code, nil
		}
	}
	return "", fmt.Errorf("failed to generate unique contact code after 10 attempts")
}
