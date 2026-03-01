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
	RoleStatusCancelled           RoleStatus = "cancelled"
)

// RoleStatusOption pairs a status value with its display label, for use in UI dropdowns
type RoleStatusOption struct {
	Value string
	Label string
}

// AllRoleStatusesWithLabels returns all valid role statuses with display labels, in display order
func AllRoleStatusesWithLabels() []RoleStatusOption {
	options := make([]RoleStatusOption, 0, len(AllRoleStatuses()))
	for _, s := range AllRoleStatuses() {
		options = append(options, RoleStatusOption{Value: s.String(), Label: s.Label()})
	}
	return options
}

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
		RoleStatusCancelled,
	}
}

// String returns the string representation of the status
func (s RoleStatus) String() string {
	return string(s)
}

// Label returns the human-readable display label for the status
func (s RoleStatus) Label() string {
	switch s {
	case RoleStatusRecruiterReachedOut:
		return "Recruiter Reached Out"
	case RoleStatusHRInterview:
		return "HR Interview"
	case RoleStatusPairingInterview:
		return "Pairing Interview"
	case RoleStatusTakeHomeAssignment:
		return "Take Home Assignment"
	case RoleStatusDesignInterview:
		return "Design Interview"
	case RoleStatusInProgress:
		return "In Progress"
	case RoleStatusOffer:
		return "Offer"
	case RoleStatusRejected:
		return "Rejected"
	case RoleStatusCancelled:
		return "Cancelled"
	default:
		return string(s)
	}
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

// IsTerminal returns true if the status is a terminal state (rejected, offer, or cancelled)
func (s RoleStatus) IsTerminal() bool {
	return s == RoleStatusRejected || s == RoleStatusOffer || s == RoleStatusCancelled
}

// CompanyStatus represents the computed status of a company based on its roles
type CompanyStatus string

const (
	CompanyStatusInProgress CompanyStatus = "in_progress"
	CompanyStatusOffer      CompanyStatus = "offer"
	CompanyStatusRejected   CompanyStatus = "rejected"
	CompanyStatusCancelled  CompanyStatus = "cancelled"
)

// String returns the string representation of the company status
func (s CompanyStatus) String() string {
	return string(s)
}

// ComputeCompanyStatus computes the aggregate company status from its roles.
// Rules:
// - If ANY role is not terminal => in_progress
// - Else if ANY role is offer => offer
// - Else if ANY role is rejected => rejected
// - Else => cancelled (all terminal roles are cancelled)
func ComputeCompanyStatus(roles []*Role) CompanyStatus {
	if len(roles) == 0 {
		return CompanyStatusInProgress
	}

	hasOffer := false
	hasRejected := false
	for _, role := range roles {
		if !role.Status.IsTerminal() {
			return CompanyStatusInProgress
		}
		if role.Status == RoleStatusOffer {
			hasOffer = true
		}
		if role.Status == RoleStatusRejected {
			hasRejected = true
		}
	}

	if hasOffer {
		return CompanyStatusOffer
	}
	if hasRejected {
		return CompanyStatusRejected
	}
	return CompanyStatusCancelled
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
	Code        string
	Slug        string
	FolderPath  string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// RoleJobDescription represents JD artifacts for a role
type RoleJobDescription struct {
	RoleID   string
	PathHTML string
	PathPDF  string
}

// RoleResume represents the current resume attached to a role
type RoleResume struct {
	RoleID   string
	PathJSON string
	PathPDF  string
}

// Meeting represents a meeting belonging to
// exactly one of: Role OR Contact (XOR constraint enforced by DB).
type Meeting struct {
	ID         string
	OccurredAt time.Time
	Title      string
	RoleID    string // Set for role meetings
	ContactID string // Set for contact meetings
	PathMD    string // relative path to markdown file
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// IsRoleMeeting returns true if this is a role meeting
func (m *Meeting) IsRoleMeeting() bool {
	return m.RoleID != ""
}

// IsContactMeeting returns true if this is a contact meeting
func (m *Meeting) IsContactMeeting() bool {
	return m.ContactID != ""
}

// ArtifactType represents the type of artifact content
type ArtifactType string

const (
	ArtifactTypePDF      ArtifactType = "pdf"
	ArtifactTypeJSONC    ArtifactType = "jsonc"
	ArtifactTypeText     ArtifactType = "text"
	ArtifactTypeHTML     ArtifactType = "html"
	ArtifactTypeMarkdown ArtifactType = "markdown"
	ArtifactTypePNG      ArtifactType = "png"
	ArtifactTypeFile     ArtifactType = "file"
)

// AllArtifactTypes returns all valid artifact types
func AllArtifactTypes() []ArtifactType {
	return []ArtifactType{
		ArtifactTypePDF,
		ArtifactTypeJSONC,
		ArtifactTypeText,
		ArtifactTypeHTML,
		ArtifactTypeMarkdown,
		ArtifactTypePNG,
		ArtifactTypeFile,
	}
}

// String returns the string representation of the artifact type
func (t ArtifactType) String() string {
	return string(t)
}

// ParseArtifactType parses a string into an ArtifactType, returning an error for invalid values
func ParseArtifactType(s string) (ArtifactType, error) {
	t := ArtifactType(s)
	for _, valid := range AllArtifactTypes() {
		if t == valid {
			return t, nil
		}
	}
	return "", fmt.Errorf("invalid artifact type: %q (must be pdf, jsonc, text, html, markdown, png, or file)", s)
}

// IsValid returns true if the type is a valid artifact type
func (t ArtifactType) IsValid() bool {
	_, err := ParseArtifactType(string(t))
	return err == nil
}

// Extension returns the file extension for this artifact type.
// Returns empty string for ArtifactTypeFile since extension is determined by the uploaded file.
func (t ArtifactType) Extension() string {
	switch t {
	case ArtifactTypePDF:
		return ".pdf"
	case ArtifactTypeJSONC:
		return ".jsonc"
	case ArtifactTypeText:
		return ".txt"
	case ArtifactTypeHTML:
		return ".html"
	case ArtifactTypeMarkdown:
		return ".md"
	case ArtifactTypePNG:
		return ".png"
	case ArtifactTypeFile:
		return ""
	default:
		return ".txt"
	}
}

// RoleArtifact represents a generic artifact attached to a role
type RoleArtifact struct {
	ID        string
	RoleID    string
	Name      string
	Type      ArtifactType
	Path      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// ValidateArtifactName validates an artifact name
func ValidateArtifactName(name string) error {
	name = trimSpaces(name)
	if name == "" {
		return fmt.Errorf("artifact name cannot be empty")
	}
	if len(name) > 255 {
		return fmt.Errorf("artifact name too long (max 255 characters)")
	}
	return nil
}

// trimSpaces trims leading and trailing whitespace
func trimSpaces(s string) string {
	// Simple trim - Go's strings.TrimSpace handles this but we avoid import
	start := 0
	end := len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n' || s[start] == '\r') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n' || s[end-1] == '\r') {
		end--
	}
	return s[start:end]
}
