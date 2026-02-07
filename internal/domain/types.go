package domain

import (
	"fmt"
	"time"
)

// RoleStatus represents the status of a job role
type RoleStatus string

// Valid role status values
const (
	RoleStatusRecruiterReachedOut RoleStatus = "recruiter_reached_out"
	RoleStatusHRInterview         RoleStatus = "hr_interview"
	RoleStatusPairingInterview    RoleStatus = "pairing_interview"
	RoleStatusTakeHomeAssignment  RoleStatus = "take_home_assignment"
	RoleStatusDesignInterview     RoleStatus = "design_interview"
	RoleStatusInProgress          RoleStatus = "in_progress"
	RoleStatusOffer               RoleStatus = "offer"
	RoleStatusRejected            RoleStatus = "rejected"
)

// AllRoleStatuses returns all valid role statuses in display order
func AllRoleStatuses() []RoleStatus {
	return []RoleStatus{
		RoleStatusRecruiterReachedOut,
		RoleStatusHRInterview,
		RoleStatusPairingInterview,
		RoleStatusTakeHomeAssignment,
		RoleStatusDesignInterview,
		RoleStatusInProgress,
		RoleStatusOffer,
		RoleStatusRejected,
	}
}

// String returns the string representation of the status
func (s RoleStatus) String() string {
	return string(s)
}

// ParseRoleStatus parses a string into a RoleStatus, returning an error for invalid values
func ParseRoleStatus(s string) (RoleStatus, error) {
	status := RoleStatus(s)
	for _, valid := range AllRoleStatuses() {
		if status == valid {
			return status, nil
		}
	}
	return "", fmt.Errorf("invalid role status: %q", s)
}

// IsValid returns true if the status is a valid role status
func (s RoleStatus) IsValid() bool {
	_, err := ParseRoleStatus(string(s))
	return err == nil
}

// IsTerminal returns true if the status is a terminal state (rejected or offer)
func (s RoleStatus) IsTerminal() bool {
	return s == RoleStatusRejected || s == RoleStatusOffer
}

// CompanyStatus represents the computed status of a company based on its roles
type CompanyStatus string

const (
	CompanyStatusInProgress CompanyStatus = "in_progress"
	CompanyStatusOffer      CompanyStatus = "offer"
	CompanyStatusRejected   CompanyStatus = "rejected"
)

// String returns the string representation of the company status
func (s CompanyStatus) String() string {
	return string(s)
}

// ComputeCompanyStatus computes the aggregate company status from its roles.
// Rules:
// - If ANY role is not in {rejected, offer} => in_progress
// - Else if ANY role is offer => offer
// - Else => rejected
func ComputeCompanyStatus(roles []*Role) CompanyStatus {
	if len(roles) == 0 {
		return CompanyStatusInProgress
	}

	hasOffer := false
	for _, role := range roles {
		if !role.Status.IsTerminal() {
			return CompanyStatusInProgress
		}
		if role.Status == RoleStatusOffer {
			hasOffer = true
		}
	}

	if hasOffer {
		return CompanyStatusOffer
	}
	return CompanyStatusRejected
}

// Company represents a company being tracked
type Company struct {
	ID         string
	Slug       string
	Name       string
	FolderPath string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// Role represents a job role at a company
type Role struct {
	ID         string
	CompanyID  string
	Slug       string
	Title      string
	Status     RoleStatus
	FolderPath string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// Contact represents a person (recruiter, hiring manager, etc.)
type Contact struct {
	ID          string
	Name        string
	Org         string
	LinkedInURL string
	Email       string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Thread represents a conversation/relationship container
type Thread struct {
	ID         string
	Code       string // 8-char unique code (e.g., "6PPEZJPW")
	Slug       string // folder slug: "<contact-slug>-<code>" or "thread-<code>"
	Title      string
	ContactID  string // optional
	FolderPath string // relative path to thread folder (e.g., "data/threads/brent-maclerie-6PPEZJPW")
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// Meeting represents a meeting or conversation
type Meeting struct {
	ID         string
	OccurredAt time.Time
	Title      string
	CompanyID  string
	PathMD     string // relative path to markdown file
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// RoleJobDescription represents JD artifacts for a role
type RoleJobDescription struct {
	RoleID   string
	PathHTML string
	PathPDF  string
}

// RoleResume represents a resume sent for a role
type RoleResume struct {
	ID       string
	RoleID   string
	SentAt   time.Time
	PathJSON string
}

// MeetingV2 represents a meeting in the new model where a meeting belongs to
// exactly one of: Role OR Thread (XOR constraint).
// - Role meeting: belongs to a Role, has no direct thread association
// - Thread-only meeting: belongs to a Thread, has no company/role association
type MeetingV2 struct {
	ID         string
	OccurredAt time.Time
	Title      string
	RoleID     string // Set for role meetings, empty for thread-only meetings
	ThreadID   string // Set for thread-only meetings, empty for role meetings
	PathMD     string // relative path to markdown file
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// IsRoleMeeting returns true if this is a role meeting
func (m *MeetingV2) IsRoleMeeting() bool {
	return m.RoleID != ""
}

// IsThreadMeeting returns true if this is a thread-only meeting
func (m *MeetingV2) IsThreadMeeting() bool {
	return m.ThreadID != ""
}
