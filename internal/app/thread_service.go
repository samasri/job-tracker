package app

import (
	"context"
	"fmt"

	"jobtracker/internal/domain"
	"jobtracker/internal/ports"

	"github.com/google/uuid"
)

// ThreadService handles thread-related business logic
type ThreadService struct {
	threadRepo  ports.ThreadRepository
	meetingRepo ports.MeetingRepository
	contactRepo ports.ContactRepository
}

// NewThreadService creates a new ThreadService
func NewThreadService(
	threadRepo ports.ThreadRepository,
	meetingRepo ports.MeetingRepository,
	contactRepo ports.ContactRepository,
) *ThreadService {
	return &ThreadService{
		threadRepo:  threadRepo,
		meetingRepo: meetingRepo,
		contactRepo: contactRepo,
	}
}

// CreateThreadInput is the input for creating a thread
type CreateThreadInput struct {
	Title     string
	ContactID string
}

// CreateThread creates a new thread
func (s *ThreadService) CreateThread(ctx context.Context, input CreateThreadInput) (*domain.Thread, error) {
	// Generate unique code with retry on collision
	var code string
	for i := 0; i < 10; i++ {
		var err error
		code, err = domain.NewShortID8()
		if err != nil {
			return nil, fmt.Errorf("generating thread code: %w", err)
		}
		exists, err := s.threadRepo.CodeExists(ctx, code)
		if err != nil {
			return nil, fmt.Errorf("checking code existence: %w", err)
		}
		if !exists {
			break
		}
		if i == 9 {
			return nil, fmt.Errorf("failed to generate unique code after 10 attempts")
		}
	}

	// Get contact name if contact is specified
	var contactName string
	if input.ContactID != "" {
		contact, err := s.contactRepo.GetByID(ctx, input.ContactID)
		if err != nil {
			return nil, fmt.Errorf("getting contact: %w", err)
		}
		if contact != nil {
			contactName = contact.Name
		}
	}

	// Generate slug and folder path
	slug := domain.GenerateThreadSlug(contactName, code)
	folderPath := domain.ThreadFolderPath(slug)

	thread := &domain.Thread{
		ID:         uuid.New().String(),
		Code:       code,
		Slug:       slug,
		Title:      input.Title,
		ContactID:  input.ContactID,
		FolderPath: folderPath,
	}

	if err := s.threadRepo.Create(ctx, thread); err != nil {
		return nil, fmt.Errorf("creating thread: %w", err)
	}

	return thread, nil
}

// ThreadWithDetails contains a thread with its meetings and linked roles
type ThreadWithDetails struct {
	Thread   *domain.Thread
	Meetings []*domain.Meeting
	Roles    []*RoleWithCompany
}

// RoleWithCompany contains a role with its company info
type RoleWithCompany struct {
	Role    *domain.Role
	Company *domain.Company
}

// GetThread retrieves a thread by ID with its legacy meetings.
// Threads no longer have linked roles (thread_roles was dropped in migration 015).
func (s *ThreadService) GetThread(ctx context.Context, id string) (*ThreadWithDetails, error) {
	thread, err := s.threadRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("getting thread: %w", err)
	}
	if thread == nil {
		return nil, nil
	}

	meetings, err := s.meetingRepo.ListByThread(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("listing meetings: %w", err)
	}

	return &ThreadWithDetails{
		Thread:   thread,
		Meetings: meetings,
		Roles:    nil,
	}, nil
}

// ListThreads returns all threads
func (s *ThreadService) ListThreads(ctx context.Context) ([]*domain.Thread, error) {
	return s.threadRepo.List(ctx)
}



