package app

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strings"

	"jobtracker/internal/domain"
	"jobtracker/internal/ports"
)

// stripJSONCComments removes JSONC comments and trailing commas from JSON content.
// Supports:
// - Single-line comments: // comment
// - Multi-line comments: /* comment */
// - Trailing commas before ] and }
func stripJSONCComments(input string) string {
	// Process character by character to handle strings correctly
	var result strings.Builder
	inString := false
	inSingleComment := false
	inMultiComment := false
	i := 0

	for i < len(input) {
		// Check for end of multi-line comment
		if inMultiComment {
			if i+1 < len(input) && input[i] == '*' && input[i+1] == '/' {
				inMultiComment = false
				i += 2
				continue
			}
			i++
			continue
		}

		// Check for end of single-line comment
		if inSingleComment {
			if input[i] == '\n' {
				inSingleComment = false
				result.WriteByte('\n')
			}
			i++
			continue
		}

		// Check for string boundaries (handle escaped quotes)
		if input[i] == '"' && (i == 0 || input[i-1] != '\\') {
			inString = !inString
			result.WriteByte(input[i])
			i++
			continue
		}

		// If in string, just copy
		if inString {
			result.WriteByte(input[i])
			i++
			continue
		}

		// Check for comment start
		if i+1 < len(input) {
			if input[i] == '/' && input[i+1] == '/' {
				inSingleComment = true
				i += 2
				continue
			}
			if input[i] == '/' && input[i+1] == '*' {
				inMultiComment = true
				i += 2
				continue
			}
		}

		// Copy character
		result.WriteByte(input[i])
		i++
	}

	// Remove trailing commas before ] or }
	cleaned := result.String()
	trailingCommaRe := regexp.MustCompile(`,(\s*[}\]])`)
	cleaned = trailingCommaRe.ReplaceAllString(cleaned, "$1")

	return cleaned
}

// ResumeService handles resume-related business logic
type ResumeService struct {
	resumeRepo  ports.ResumeRepository
	companyRepo ports.CompanyRepository
	roleRepo    ports.RoleRepository
	fileStore   ports.FileStore
}

// NewResumeService creates a new ResumeService
func NewResumeService(
	resumeRepo ports.ResumeRepository,
	companyRepo ports.CompanyRepository,
	roleRepo ports.RoleRepository,
	fileStore ports.FileStore,
) *ResumeService {
	return &ResumeService{
		resumeRepo:  resumeRepo,
		companyRepo: companyRepo,
		roleRepo:    roleRepo,
		fileStore:   fileStore,
	}
}

// AttachResumeInput is the input for attaching a resume to a role
type AttachResumeInput struct {
	CompanySlug string
	RoleSlug    string
	JSONContent string    // JSON content from textarea
	PDFContent  io.Reader // PDF file upload (optional)
}

// AttachResume attaches a resume to a role
func (s *ResumeService) AttachResume(ctx context.Context, input AttachResumeInput) (*domain.RoleResume, error) {
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

	// Validate that at least one input is provided
	if input.JSONContent == "" && input.PDFContent == nil {
		return nil, fmt.Errorf("at least one of JSON or PDF must be provided")
	}

	// Validate JSON if provided (supports JSONC with comments)
	if input.JSONContent != "" {
		// Strip comments for validation only
		cleanedJSON := stripJSONCComments(input.JSONContent)
		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(cleanedJSON), &parsed); err != nil {
			return nil, fmt.Errorf("invalid JSON: %w", err)
		}
	}

	resume := &domain.RoleResume{RoleID: role.ID}

	// Save JSON if provided (preserve original content with comments)
	if input.JSONContent != "" {
		pathJSON, err := s.fileStore.SaveRoleResumeJSON(ctx, input.CompanySlug, input.RoleSlug, input.JSONContent)
		if err != nil {
			return nil, fmt.Errorf("saving JSON: %w", err)
		}
		resume.PathJSON = pathJSON
	}

	// Save PDF if provided
	if input.PDFContent != nil {
		pathPDF, err := s.fileStore.SaveRoleResumePDF(ctx, input.CompanySlug, input.RoleSlug, input.PDFContent)
		if err != nil {
			return nil, fmt.Errorf("saving PDF: %w", err)
		}
		resume.PathPDF = pathPDF
	}

	// Save to database
	if err := s.resumeRepo.Save(ctx, resume); err != nil {
		return nil, fmt.Errorf("saving resume record: %w", err)
	}

	return resume, nil
}

// GetResume retrieves a resume for a role by company and role slugs
func (s *ResumeService) GetResume(ctx context.Context, companySlug, roleSlug string) (*domain.RoleResume, error) {
	// Get company
	company, err := s.companyRepo.GetBySlug(ctx, companySlug)
	if err != nil {
		return nil, fmt.Errorf("getting company: %w", err)
	}
	if company == nil {
		return nil, nil
	}

	// Get role
	role, err := s.roleRepo.GetBySlug(ctx, company.ID, roleSlug)
	if err != nil {
		return nil, fmt.Errorf("getting role: %w", err)
	}
	if role == nil {
		return nil, nil
	}

	return s.resumeRepo.GetByRoleID(ctx, role.ID)
}

// GetResumeByRoleID retrieves a resume by role ID
func (s *ResumeService) GetResumeByRoleID(ctx context.Context, roleID string) (*domain.RoleResume, error) {
	return s.resumeRepo.GetByRoleID(ctx, roleID)
}
