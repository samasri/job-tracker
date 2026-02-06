package ports

import (
	"context"
	"jobtracker/internal/domain"
)

// CompanyRepository defines operations for company persistence
type CompanyRepository interface {
	Create(ctx context.Context, company *domain.Company) error
	GetBySlug(ctx context.Context, slug string) (*domain.Company, error)
	GetByID(ctx context.Context, id string) (*domain.Company, error)
	List(ctx context.Context) ([]*domain.Company, error)
}

// RoleRepository defines operations for role persistence
type RoleRepository interface {
	Create(ctx context.Context, role *domain.Role) error
	GetBySlug(ctx context.Context, companyID, slug string) (*domain.Role, error)
	GetByID(ctx context.Context, id string) (*domain.Role, error)
	ListByCompany(ctx context.Context, companyID string) ([]*domain.Role, error)
	UpdateStatus(ctx context.Context, roleID string, status domain.RoleStatus) error
}

// ContactRepository defines operations for contact persistence
type ContactRepository interface {
	Create(ctx context.Context, contact *domain.Contact) error
	GetByID(ctx context.Context, id string) (*domain.Contact, error)
	List(ctx context.Context) ([]*domain.Contact, error)
}

// ThreadRepository defines operations for thread persistence
type ThreadRepository interface {
	Create(ctx context.Context, thread *domain.Thread) error
	GetByID(ctx context.Context, id string) (*domain.Thread, error)
	List(ctx context.Context) ([]*domain.Thread, error)
	LinkRole(ctx context.Context, threadID, roleID string) error
	GetLinkedRoles(ctx context.Context, threadID string) ([]*domain.Role, error)
}

// MeetingRepository defines operations for meeting persistence
type MeetingRepository interface {
	Create(ctx context.Context, meeting *domain.Meeting) error
	GetByID(ctx context.Context, id string) (*domain.Meeting, error)
	ListByCompany(ctx context.Context, companyID string) ([]*domain.Meeting, error)
	ListByThread(ctx context.Context, threadID string) ([]*domain.Meeting, error)
	LinkThread(ctx context.Context, meetingID, threadID string) error
}

// JobDescriptionRepository defines operations for JD artifacts
type JobDescriptionRepository interface {
	Save(ctx context.Context, jd *domain.RoleJobDescription) error
	GetByRoleID(ctx context.Context, roleID string) (*domain.RoleJobDescription, error)
}

// MeetingV2Repository defines operations for meetings_v2 persistence
// Meetings in v2 belong to exactly one of: Role OR Thread (XOR)
type MeetingV2Repository interface {
	// Create inserts a new meeting (must have either RoleID or ThreadID set, not both)
	Create(ctx context.Context, meeting *domain.MeetingV2) error
	// GetByID retrieves a meeting by ID
	GetByID(ctx context.Context, id string) (*domain.MeetingV2, error)
	// ListByRole retrieves all meetings for a role ordered by occurred_at desc
	ListByRole(ctx context.Context, roleID string) ([]*domain.MeetingV2, error)
	// ListByThread retrieves all thread-only meetings for a thread ordered by occurred_at desc
	ListByThread(ctx context.Context, threadID string) ([]*domain.MeetingV2, error)
}
