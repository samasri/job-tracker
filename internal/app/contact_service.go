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
	contactRepo ports.ContactRepository
}

// NewContactService creates a new ContactService
func NewContactService(contactRepo ports.ContactRepository) *ContactService {
	return &ContactService{
		contactRepo: contactRepo,
	}
}

// CreateContactInput is the input for creating a contact
type CreateContactInput struct {
	Name        string
	Org         string
	LinkedInURL string
	Email       string
}

// CreateContact creates a new contact
func (s *ContactService) CreateContact(ctx context.Context, input CreateContactInput) (*domain.Contact, error) {
	contact := &domain.Contact{
		ID:          uuid.New().String(),
		Name:        input.Name,
		Org:         input.Org,
		LinkedInURL: input.LinkedInURL,
		Email:       input.Email,
	}

	if err := s.contactRepo.Create(ctx, contact); err != nil {
		return nil, fmt.Errorf("creating contact: %w", err)
	}

	return contact, nil
}

// GetContact retrieves a contact by ID
func (s *ContactService) GetContact(ctx context.Context, id string) (*domain.Contact, error) {
	return s.contactRepo.GetByID(ctx, id)
}

// ListContacts returns all contacts
func (s *ContactService) ListContacts(ctx context.Context) ([]*domain.Contact, error) {
	return s.contactRepo.List(ctx)
}
