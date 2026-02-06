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
	threadRepo    ports.ThreadRepository
	fileStore     ports.FileStore
}

// NewMeetingV2Service creates a new MeetingV2Service
func NewMeetingV2Service(
	meetingV2Repo ports.MeetingV2Repository,
	companyRepo ports.CompanyRepository,
	roleRepo ports.RoleRepository,
	threadRepo ports.ThreadRepository,
	fileStore ports.FileStore,
) *MeetingV2Service {
	return &MeetingV2Service{
		meetingV2Repo: meetingV2Repo,
		companyRepo:   companyRepo,
		roleRepo:      roleRepo,
		threadRepo:    threadRepo,
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

// CreateThreadMeetingInput is the input for creating a thread-only meeting
type CreateThreadMeetingInput struct {
	ThreadID   string
	OccurredAt string // ISO 8601 format
	Title      string
}

// CreateThreadMeeting creates a new meeting associated with a thread (no role/company)
func (s *MeetingV2Service) CreateThreadMeeting(ctx context.Context, input CreateThreadMeetingInput) (*domain.MeetingV2, error) {
	// Verify thread exists
	thread, err := s.threadRepo.GetByID(ctx, input.ThreadID)
	if err != nil {
		return nil, fmt.Errorf("getting thread: %w", err)
	}
	if thread == nil {
		return nil, fmt.Errorf("thread '%s' not found", input.ThreadID)
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
	pathMD, err := s.fileStore.CreateThreadMeetingNote(ctx, input.ThreadID, input.OccurredAt, input.Title, meetingID)
	if err != nil {
		return nil, fmt.Errorf("creating thread meeting note: %w", err)
	}

	// Create meeting record
	meeting := &domain.MeetingV2{
		ID:         meetingID,
		OccurredAt: occurredAt,
		Title:      input.Title,
		ThreadID:   input.ThreadID,
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

// ListMeetingsByThread returns all thread-only meetings for a thread
func (s *MeetingV2Service) ListMeetingsByThread(ctx context.Context, threadID string) ([]*domain.MeetingV2, error) {
	return s.meetingV2Repo.ListByThread(ctx, threadID)
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
