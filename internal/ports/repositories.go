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
	GetBySlug(ctx context.Context, slug string) (*domain.Contact, error)
	List(ctx context.Context) ([]*domain.Contact, error)
	UpdateCodeSlug(ctx context.Context, id, code, slug, folderPath string) error
	CodeExists(ctx context.Context, code string) (bool, error)
}

// ContactRoleRepository defines operations for contact-role links
type ContactRoleRepository interface {
	LinkRole(ctx context.Context, contactID, roleID string) error
	GetLinkedRoles(ctx context.Context, contactID string) ([]*domain.Role, error)
}

// JobDescriptionRepository defines operations for JD artifacts
type JobDescriptionRepository interface {
	Save(ctx context.Context, jd *domain.RoleJobDescription) error
	GetByRoleID(ctx context.Context, roleID string) (*domain.RoleJobDescription, error)
}

// MeetingRepository defines operations for meetings persistence
type MeetingRepository interface {
	// Create inserts a new meeting
	Create(ctx context.Context, meeting *domain.Meeting) error
	// GetByID retrieves a meeting by ID
	GetByID(ctx context.Context, id string) (*domain.Meeting, error)
	// ListByRole retrieves all meetings for a role ordered by occurred_at desc
	ListByRole(ctx context.Context, roleID string) ([]*domain.Meeting, error)
	// ListByContact retrieves all contact meetings ordered by occurred_at desc
	ListByContact(ctx context.Context, contactID string) ([]*domain.Meeting, error)
	// UpdatePathMD updates the path_md for a meeting
	UpdatePathMD(ctx context.Context, meetingID, newPath string) error
	// SetContactID sets the contact_id on a meeting
	SetContactID(ctx context.Context, meetingID, contactID string) error
}

// ResumeRepository defines operations for role resume artifacts
type ResumeRepository interface {
	Save(ctx context.Context, resume *domain.RoleResume) error
	GetByRoleID(ctx context.Context, roleID string) (*domain.RoleResume, error)
}

// RoleArtifactRepository defines operations for generic role artifacts
type RoleArtifactRepository interface {
	// Upsert creates or updates an artifact by (role_id, name).
	// If an artifact with the same role_id and name exists, it updates the record.
	// Otherwise, it creates a new record with a new ID.
	Upsert(ctx context.Context, artifact *domain.RoleArtifact) (*domain.RoleArtifact, error)
	// List returns all artifacts for a role, ordered by name ASC
	List(ctx context.Context, roleID string) ([]*domain.RoleArtifact, error)
	// GetByName retrieves an artifact by role_id and name
	GetByName(ctx context.Context, roleID, name string) (*domain.RoleArtifact, error)
	// Delete removes an artifact by role_id and name
	Delete(ctx context.Context, roleID, name string) error
}
