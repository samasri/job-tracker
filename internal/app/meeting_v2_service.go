package app

import (
	"context"
	"fmt"
	"time"

	"jobtracker/internal/domain"
	"jobtracker/internal/ports"
)

// MeetingV2Service handles meeting-related business logic for meetings_v2
type MeetingV2Service struct {
	meetingV2Repo ports.MeetingV2Repository
	companyRepo   ports.CompanyRepository
	roleRepo      ports.RoleRepository
	contactRepo   ports.ContactRepository
	fileStore     ports.FileStore
}

// NewMeetingV2Service creates a new MeetingV2Service
func NewMeetingV2Service(
	meetingV2Repo ports.MeetingV2Repository,
	companyRepo ports.CompanyRepository,
	roleRepo ports.RoleRepository,
	contactRepo ports.ContactRepository,
	fileStore ports.FileStore,
) *MeetingV2Service {
	return &MeetingV2Service{
		meetingV2Repo: meetingV2Repo,
		companyRepo:   companyRepo,
		roleRepo:      roleRepo,
		contactRepo:   contactRepo,
		fileStore:     fileStore,
	}
}

// CreateRoleMeetingInput is the input for creating a role meeting
type CreateRoleMeetingInput struct {
	CompanySlug string
	RoleSlug    string
	OccurredAt  string // ISO 8601 format
	Title       string
}

// maxMeetingIDAttempts is the maximum number of attempts to generate a unique meeting ID
const maxMeetingIDAttempts = 5

// CreateRoleMeeting creates a new meeting associated with a role
func (s *MeetingV2Service) CreateRoleMeeting(ctx context.Context, input CreateRoleMeetingInput) (*domain.MeetingV2, error) {
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

	// Parse occurred_at
	occurredAt, err := parseOccurredAt(input.OccurredAt)
	if err != nil {
		return nil, err
	}

	// Generate unique 8-char meeting ID with collision retry
	meetingID, err := s.generateUniqueMeetingID(ctx)
	if err != nil {
		return nil, err
	}

	// Create meeting note file
	pathMD, err := s.fileStore.CreateRoleMeetingNote(ctx, input.CompanySlug, input.RoleSlug, input.OccurredAt, input.Title, meetingID)
	if err != nil {
		return nil, fmt.Errorf("creating role meeting note: %w", err)
	}

	// Create meeting record
	meeting := &domain.MeetingV2{
		ID:         meetingID,
		OccurredAt: occurredAt,
		Title:      input.Title,
		RoleID:     role.ID,
		PathMD:     pathMD,
	}

	if err := s.meetingV2Repo.Create(ctx, meeting); err != nil {
		return nil, fmt.Errorf("creating meeting record: %w", err)
	}

	return meeting, nil
}

// CreateContactMeetingInput is the input for creating a contact meeting
type CreateContactMeetingInput struct {
	ContactID  string
	OccurredAt string // ISO 8601 format
	Title      string
}

// CreateContactMeeting creates a new meeting associated with a contact
func (s *MeetingV2Service) CreateContactMeeting(ctx context.Context, input CreateContactMeetingInput) (*domain.MeetingV2, error) {
	contact, err := s.contactRepo.GetByID(ctx, input.ContactID)
	if err != nil {
		return nil, fmt.Errorf("getting contact: %w", err)
	}
	if contact == nil {
		return nil, fmt.Errorf("contact '%s' not found", input.ContactID)
	}
	if contact.Slug == "" {
		return nil, fmt.Errorf("contact '%s' has no slug", input.ContactID)
	}

	occurredAt, err := parseOccurredAt(input.OccurredAt)
	if err != nil {
		return nil, err
	}

	meetingID, err := s.generateUniqueMeetingID(ctx)
	if err != nil {
		return nil, err
	}

	pathMD, err := s.fileStore.CreateContactMeetingNote(ctx, contact.Slug, occurredAt, input.Title, meetingID)
	if err != nil {
		return nil, fmt.Errorf("creating contact meeting note: %w", err)
	}

	meeting := &domain.MeetingV2{
		ID:         meetingID,
		OccurredAt: occurredAt,
		Title:      input.Title,
		ContactID:  input.ContactID,
		PathMD:     pathMD,
	}

	if err := s.meetingV2Repo.Create(ctx, meeting); err != nil {
		return nil, fmt.Errorf("creating meeting record: %w", err)
	}

	return meeting, nil
}

// GetMeeting retrieves a meeting by ID
func (s *MeetingV2Service) GetMeeting(ctx context.Context, id string) (*domain.MeetingV2, error) {
	return s.meetingV2Repo.GetByID(ctx, id)
}

// ListMeetingsByRole returns all meetings for a role
func (s *MeetingV2Service) ListMeetingsByRole(ctx context.Context, roleID string) ([]*domain.MeetingV2, error) {
	return s.meetingV2Repo.ListByRole(ctx, roleID)
}

// ListMeetingsByContact returns all contact meetings
func (s *MeetingV2Service) ListMeetingsByContact(ctx context.Context, contactID string) ([]*domain.MeetingV2, error) {
	return s.meetingV2Repo.ListByContact(ctx, contactID)
}

// generateUniqueMeetingID generates a unique 8-char meeting ID
func (s *MeetingV2Service) generateUniqueMeetingID(ctx context.Context) (string, error) {
	for attempt := 0; attempt < maxMeetingIDAttempts; attempt++ {
		candidateID, err := domain.NewShortID8()
		if err != nil {
			return "", fmt.Errorf("generating meeting ID: %w", err)
		}

		// Check if ID already exists
		existing, err := s.meetingV2Repo.GetByID(ctx, candidateID)
		if err != nil {
			return "", fmt.Errorf("checking ID uniqueness: %w", err)
		}
		if existing == nil {
			return candidateID, nil
		}
		// ID collision, retry with new ID
	}
	return "", fmt.Errorf("failed to generate unique meeting ID after %d attempts", maxMeetingIDAttempts)
}

// parseOccurredAt parses an occurred_at string in various formats
func parseOccurredAt(s string) (time.Time, error) {
	// Try RFC3339 first
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	// Try datetime-local format (from HTML forms)
	if t, err := time.Parse("2006-01-02T15:04", s); err == nil {
		return t, nil
	}
	// Try just the date
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return t, nil
	}
	return time.Time{}, fmt.Errorf("invalid occurred_at format (use ISO 8601): %s", s)
}
