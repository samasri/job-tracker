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
	companyRepo ports.CompanyRepository
	roleRepo    ports.RoleRepository
}

// NewThreadService creates a new ThreadService
func NewThreadService(
	threadRepo ports.ThreadRepository,
	meetingRepo ports.MeetingRepository,
	companyRepo ports.CompanyRepository,
	roleRepo ports.RoleRepository,
) *ThreadService {
	return &ThreadService{
		threadRepo:  threadRepo,
		meetingRepo: meetingRepo,
		companyRepo: companyRepo,
		roleRepo:    roleRepo,
	}
}

// CreateThreadInput is the input for creating a thread
type CreateThreadInput struct {
	Title     string
	ContactID string
}

// CreateThread creates a new thread
func (s *ThreadService) CreateThread(ctx context.Context, input CreateThreadInput) (*domain.Thread, error) {
	thread := &domain.Thread{
		ID:        uuid.New().String(),
		Title:     input.Title,
		ContactID: input.ContactID,
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

// GetThread retrieves a thread by ID with its meetings and linked roles
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

	roles, err := s.threadRepo.GetLinkedRoles(ctx, id)
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

	return &ThreadWithDetails{
		Thread:   thread,
		Meetings: meetings,
		Roles:    rolesWithCompany,
	}, nil
}

// ListThreads returns all threads
func (s *ThreadService) ListThreads(ctx context.Context) ([]*domain.Thread, error) {
	return s.threadRepo.List(ctx)
}

// LinkRoleInput is the input for linking a role to a thread
type LinkRoleInput struct {
	ThreadID    string
	CompanySlug string
	RoleSlug    string
}

// LinkRole links a role to a thread (idempotent)
func (s *ThreadService) LinkRole(ctx context.Context, input LinkRoleInput) error {
	// Verify thread exists
	thread, err := s.threadRepo.GetByID(ctx, input.ThreadID)
	if err != nil {
		return fmt.Errorf("getting thread: %w", err)
	}
	if thread == nil {
		return fmt.Errorf("thread '%s' not found", input.ThreadID)
	}

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

	// Link thread to role (idempotent)
	if err := s.threadRepo.LinkRole(ctx, input.ThreadID, role.ID); err != nil {
		return fmt.Errorf("linking role: %w", err)
	}

	return nil
}
