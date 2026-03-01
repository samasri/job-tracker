package app

import (
	"context"
	"fmt"
	"time"

	"jobtracker/internal/domain"
	"jobtracker/internal/ports"
)

// MeetingService handles meeting-related business logic
type MeetingService struct {
	meetingRepo ports.MeetingRepository
	companyRepo ports.CompanyRepository
	fileStore   ports.FileStore
}

// NewMeetingService creates a new MeetingService
func NewMeetingService(
	meetingRepo ports.MeetingRepository,
	companyRepo ports.CompanyRepository,
	fileStore ports.FileStore,
) *MeetingService {
	return &MeetingService{
		meetingRepo: meetingRepo,
		companyRepo: companyRepo,
		fileStore:   fileStore,
	}
}

// CreateMeetingInput is the input for creating a meeting
type CreateMeetingInput struct {
	CompanySlug string
	OccurredAt  string // ISO 8601 format
	Title       string
}

// maxIDGenerationAttempts is the maximum number of attempts to generate a unique meeting ID
const maxIDGenerationAttempts = 5

// CreateMeeting creates a new meeting with a note file
func (s *MeetingService) CreateMeeting(ctx context.Context, input CreateMeetingInput) (*domain.Meeting, error) {
	// Get company
	company, err := s.companyRepo.GetBySlug(ctx, input.CompanySlug)
	if err != nil {
		return nil, fmt.Errorf("getting company: %w", err)
	}
	if company == nil {
		return nil, fmt.Errorf("company '%s' not found", input.CompanySlug)
	}

	// Parse occurred_at
	occurredAt, err := time.Parse(time.RFC3339, input.OccurredAt)
	if err != nil {
		// Try parsing just the date
		occurredAt, err = time.Parse("2006-01-02", input.OccurredAt)
		if err != nil {
			return nil, fmt.Errorf("invalid occurred_at format (use ISO 8601): %w", err)
		}
	}

	// Generate unique 8-char meeting ID with collision retry
	var meetingID string
	for attempt := 0; attempt < maxIDGenerationAttempts; attempt++ {
		candidateID, err := domain.NewShortID8()
		if err != nil {
			return nil, fmt.Errorf("generating meeting ID: %w", err)
		}

		// Check if ID already exists
		existing, err := s.meetingRepo.GetByID(ctx, candidateID)
		if err != nil {
			return nil, fmt.Errorf("checking ID uniqueness: %w", err)
		}
		if existing == nil {
			meetingID = candidateID
			break
		}
		// ID collision, retry with new ID
	}
	if meetingID == "" {
		return nil, fmt.Errorf("failed to generate unique meeting ID after %d attempts", maxIDGenerationAttempts)
	}

	// Create meeting note file
	pathMD, err := s.fileStore.CreateMeetingNote(ctx, input.CompanySlug, meetingID, input.OccurredAt, input.Title)
	if err != nil {
		return nil, fmt.Errorf("creating meeting note: %w", err)
	}

	// Create meeting record
	meeting := &domain.Meeting{
		ID:         meetingID,
		OccurredAt: occurredAt,
		Title:      input.Title,
		CompanyID:  company.ID,
		PathMD:     pathMD,
	}

	if err := s.meetingRepo.Create(ctx, meeting); err != nil {
		return nil, fmt.Errorf("creating meeting record: %w", err)
	}

	return meeting, nil
}

// GetMeeting retrieves a meeting by ID
func (s *MeetingService) GetMeeting(ctx context.Context, id string) (*domain.Meeting, error) {
	return s.meetingRepo.GetByID(ctx, id)
}

// ListMeetingsByCompany returns all meetings for a company
func (s *MeetingService) ListMeetingsByCompany(ctx context.Context, companyID string) ([]*domain.Meeting, error) {
	return s.meetingRepo.ListByCompany(ctx, companyID)
}
