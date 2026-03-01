package http

import (
	"context"

	"jobtracker/internal/app"
	"jobtracker/internal/http/views"
)

type handlers struct {
	companyService   *app.CompanyService
	contactService   *app.ContactService
	meetingService *app.MeetingService
	jdService        *app.JDService
	resumeService    *app.ResumeService
	artifactService  *app.ArtifactService
	exportService    *app.ExportService
	views            *views.Views
}

func NewHandlers(
	companyService *app.CompanyService,
	contactService *app.ContactService,
	meetingService *app.MeetingService,
	jdService *app.JDService,
	resumeService *app.ResumeService,
	artifactService *app.ArtifactService,
	exportService *app.ExportService,
) *handlers {
	v, err := views.New()
	if err != nil {
		panic("failed to parse templates: " + err.Error())
	}

	return &handlers{
		companyService:   companyService,
		contactService:   contactService,
		meetingService: meetingService,
		jdService:        jdService,
		resumeService:    resumeService,
		artifactService:  artifactService,
		exportService:    exportService,
		views:            v,
	}
}

// --- Response Types ---

type companyResponse struct {
	ID         string `json:"id"`
	Slug       string `json:"slug"`
	Name       string `json:"name"`
	FolderPath string `json:"folder_path"`
	Status     string `json:"status,omitempty"`
}

type roleResponse struct {
	ID         string `json:"id"`
	CompanyID  string `json:"company_id"`
	Slug       string `json:"slug"`
	Title      string `json:"title"`
	FolderPath string `json:"folder_path"`
}

type meetingResponse struct {
	ID        string `json:"id"`
	OccurredAt string `json:"occurred_at"`
	Title     string `json:"title"`
	RoleID    string `json:"role_id,omitempty"`
	ContactID string `json:"contact_id,omitempty"`
	PathMD    string `json:"path_md"`
}

type contactResponse struct {
	ID          string                    `json:"id"`
	Name        string                    `json:"name"`
	Org         string                    `json:"org,omitempty"`
	LinkedInURL string                    `json:"linkedin_url,omitempty"`
	Email       string                    `json:"email,omitempty"`
	Code        string                    `json:"code,omitempty"`
	Slug        string                    `json:"slug,omitempty"`
	FolderPath  string                    `json:"folder_path,omitempty"`
	Roles       []roleWithCompanyResponse `json:"roles,omitempty"`
}

type companyWithDetailsResponse struct {
	Company companyResponse `json:"company"`
	Roles   []roleResponse  `json:"roles"`
}

type roleWithCompanyResponse struct {
	Role    roleResponse    `json:"role"`
	Company companyResponse `json:"company"`
}

// multipartFileReader wraps a multipart file for io.Reader interface
type multipartFileReader struct {
	file interface {
		Read([]byte) (int, error)
	}
}

func (r *multipartFileReader) Read(p []byte) (int, error) {
	return r.file.Read(p)
}

// roleDropdownItem represents a role for dropdown display
type roleDropdownItem struct {
	CompanySlug string
	CompanyName string
	RoleSlug    string
	RoleTitle   string
}

func splitRoleRef(ref string) []string {
	for i := 0; i < len(ref); i++ {
		if ref[i] == '/' {
			return []string{ref[:i], ref[i+1:]}
		}
	}
	return []string{ref}
}

func (h *handlers) getAllRolesForDropdown(ctx context.Context) ([]roleDropdownItem, error) {
	companies, err := h.companyService.ListCompanies(ctx)
	if err != nil {
		return nil, err
	}
	var roles []roleDropdownItem
	for _, c := range companies {
		details, err := h.companyService.GetCompany(ctx, c.Company.Slug)
		if err != nil {
			return nil, err
		}
		if details != nil {
			for _, r := range details.Roles {
				roles = append(roles, roleDropdownItem{
					CompanySlug: c.Company.Slug,
					CompanyName: c.Company.Name,
					RoleSlug:    r.Slug,
					RoleTitle:   r.Title,
				})
			}
		}
	}
	return roles, nil
}
