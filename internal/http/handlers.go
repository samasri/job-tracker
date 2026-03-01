package http

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"path/filepath"
	"sort"
	"strconv"
	"time"

	"jobtracker/internal/app"
	"jobtracker/internal/domain"
	"jobtracker/internal/http/views"

	"github.com/go-chi/chi/v5"
	"github.com/yuin/goldmark"
)

// Handlers holds all HTTP handler dependencies
type Handlers struct {
	companyService   *app.CompanyService
	contactService   *app.ContactService
	meetingService   *app.MeetingService
	meetingV2Service *app.MeetingV2Service
	jdService        *app.JDService
	resumeService    *app.ResumeService
	artifactService  *app.ArtifactService
	exportService    *app.ExportService
	views            *views.Views
}

// NewHandlers creates a new Handlers instance
func NewHandlers(
	companyService *app.CompanyService,
	contactService *app.ContactService,
	meetingService *app.MeetingService,
	meetingV2Service *app.MeetingV2Service,
	jdService *app.JDService,
	resumeService *app.ResumeService,
	artifactService *app.ArtifactService,
	exportService *app.ExportService,
) *Handlers {
	v, err := views.New()
	if err != nil {
		panic("failed to parse templates: " + err.Error())
	}

	return &Handlers{
		companyService:   companyService,
		contactService:   contactService,
		meetingService:   meetingService,
		meetingV2Service: meetingV2Service,
		jdService:        jdService,
		resumeService:    resumeService,
		artifactService:  artifactService,
		exportService:    exportService,
		views:            v,
	}
}

// --- Response Types ---

// CompanyResponse is the response for a company
type CompanyResponse struct {
	ID         string `json:"id"`
	Slug       string `json:"slug"`
	Name       string `json:"name"`
	FolderPath string `json:"folder_path"`
	Status     string `json:"status,omitempty"`
}

// RoleResponse is the response for a role
type RoleResponse struct {
	ID         string `json:"id"`
	CompanyID  string `json:"company_id"`
	Slug       string `json:"slug"`
	Title      string `json:"title"`
	FolderPath string `json:"folder_path"`
}

// MeetingResponse is the response for a meeting (legacy)
type MeetingResponse struct {
	ID         string `json:"id"`
	OccurredAt string `json:"occurred_at"`
	Title      string `json:"title"`
	CompanyID  string `json:"company_id"`
	PathMD     string `json:"path_md"`
}

// MeetingV2Response is the response for a meeting_v2
type MeetingV2Response struct {
	ID         string `json:"id"`
	OccurredAt string `json:"occurred_at"`
	Title      string `json:"title"`
	RoleID    string `json:"role_id,omitempty"`
	ContactID string `json:"contact_id,omitempty"`
	PathMD     string `json:"path_md"`
}

// ContactResponse is the response for a contact
type ContactResponse struct {
	ID          string             `json:"id"`
	Name        string             `json:"name"`
	Org         string             `json:"org,omitempty"`
	LinkedInURL string             `json:"linkedin_url,omitempty"`
	Email       string             `json:"email,omitempty"`
	Code        string             `json:"code,omitempty"`
	Slug        string             `json:"slug,omitempty"`
	FolderPath  string             `json:"folder_path,omitempty"`
	Roles       []RoleWithCompanyResponse `json:"roles,omitempty"`
}

// CompanyWithDetailsResponse is the response for a company with roles and meetings
type CompanyWithDetailsResponse struct {
	Company  CompanyResponse   `json:"company"`
	Roles    []RoleResponse    `json:"roles"`
	Meetings []MeetingResponse `json:"meetings"`
}

// RoleWithCompanyResponse is a role with its company info
type RoleWithCompanyResponse struct {
	Role    RoleResponse    `json:"role"`
	Company CompanyResponse `json:"company"`
}

// --- Company Handlers ---

// CreateCompanyRequest is the request body for creating a company
type CreateCompanyRequest struct {
	Slug string `json:"slug"`
	Name string `json:"name"`
}

// HandleCreateCompany handles POST /api/companies
func (h *Handlers) HandleCreateCompany() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req CreateCompanyRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		if req.Slug == "" || req.Name == "" {
			http.Error(w, "slug and name are required", http.StatusBadRequest)
			return
		}

		company, err := h.companyService.CreateCompany(r.Context(), app.CreateCompanyInput{
			Slug: req.Slug,
			Name: req.Name,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(CompanyResponse{
			ID:         company.ID,
			Slug:       company.Slug,
			Name:       company.Name,
			FolderPath: company.FolderPath,
		})
	}
}

// HandleListCompanies handles GET /api/companies
func (h *Handlers) HandleListCompanies() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		companies, err := h.companyService.ListCompanies(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		result := make([]CompanyResponse, 0, len(companies))
		for _, c := range companies {
			result = append(result, CompanyResponse{
				ID:         c.Company.ID,
				Slug:       c.Company.Slug,
				Name:       c.Company.Name,
				FolderPath: c.Company.FolderPath,
				Status:     c.Status.String(),
			})
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	}
}

// HandleGetCompany handles GET /api/companies/{slug}
func (h *Handlers) HandleGetCompany() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		slug := chi.URLParam(r, "slug")

		company, err := h.companyService.GetCompany(r.Context(), slug)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if company == nil {
			http.Error(w, "company not found", http.StatusNotFound)
			return
		}

		roles := make([]RoleResponse, 0, len(company.Roles))
		for _, role := range company.Roles {
			roles = append(roles, RoleResponse{
				ID:         role.ID,
				CompanyID:  role.CompanyID,
				Slug:       role.Slug,
				Title:      role.Title,
				FolderPath: role.FolderPath,
			})
		}

		meetings := make([]MeetingResponse, 0, len(company.Meetings))
		for _, m := range company.Meetings {
			meetings = append(meetings, MeetingResponse{
				ID:         m.ID,
				OccurredAt: m.OccurredAt.Format(time.RFC3339),
				Title:      m.Title,
				CompanyID:  m.CompanyID,
				PathMD:     m.PathMD,
			})
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(CompanyWithDetailsResponse{
			Company: CompanyResponse{
				ID:         company.Company.ID,
				Slug:       company.Company.Slug,
				Name:       company.Company.Name,
				FolderPath: company.Company.FolderPath,
				Status:     company.Status.String(),
			},
			Roles:    roles,
			Meetings: meetings,
		})
	}
}

// --- Role Handlers ---

// CreateRoleRequest is the request body for creating a role
type CreateRoleRequest struct {
	Slug  string `json:"slug"`
	Title string `json:"title"`
}

// HandleCreateRole handles POST /api/companies/{slug}/roles
func (h *Handlers) HandleCreateRole() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		companySlug := chi.URLParam(r, "slug")

		var req CreateRoleRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		if req.Slug == "" || req.Title == "" {
			http.Error(w, "slug and title are required", http.StatusBadRequest)
			return
		}

		role, err := h.companyService.CreateRole(r.Context(), app.CreateRoleInput{
			CompanySlug: companySlug,
			Slug:        req.Slug,
			Title:       req.Title,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(RoleResponse{
			ID:         role.ID,
			CompanyID:  role.CompanyID,
			Slug:       role.Slug,
			Title:      role.Title,
			FolderPath: role.FolderPath,
		})
	}
}

// UpdateRoleStatusRequest is the request body for updating role status
type UpdateRoleStatusRequest struct {
	Status string `json:"status"`
}

// HandleUpdateRoleStatus handles PATCH /api/companies/{companySlug}/roles/{roleSlug}/status
func (h *Handlers) HandleUpdateRoleStatus() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		companySlug := chi.URLParam(r, "companySlug")
		roleSlug := chi.URLParam(r, "roleSlug")

		var req UpdateRoleStatusRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		if req.Status == "" {
			http.Error(w, "status is required", http.StatusBadRequest)
			return
		}

		err := h.companyService.UpdateRoleStatus(r.Context(), app.UpdateRoleStatusInput{
			CompanySlug: companySlug,
			RoleSlug:    roleSlug,
			Status:      req.Status,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}
}

// --- Contact Handlers ---

// CreateContactRequest is the request body for creating a contact
type CreateContactRequest struct {
	Name        string `json:"name"`
	Org         string `json:"org"`
	LinkedInURL string `json:"linkedin_url"`
	Email       string `json:"email"`
}

// HandleCreateContact handles POST /api/contacts
func (h *Handlers) HandleCreateContact() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req CreateContactRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		if req.Name == "" {
			http.Error(w, "name is required", http.StatusBadRequest)
			return
		}

		contact, err := h.contactService.CreateContact(r.Context(), app.CreateContactInput{
			Name:        req.Name,
			Org:         req.Org,
			LinkedInURL: req.LinkedInURL,
			Email:       req.Email,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(ContactResponse{
			ID:          contact.ID,
			Name:        contact.Name,
			Org:         contact.Org,
			LinkedInURL: contact.LinkedInURL,
			Email:       contact.Email,
			Code:        contact.Code,
			Slug:        contact.Slug,
			FolderPath:  contact.FolderPath,
		})
	}
}

// HandleGetContactAPI handles GET /api/contacts/{id}
func (h *Handlers) HandleGetContactAPI() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")

		details, err := h.contactService.GetContactWithDetails(r.Context(), id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if details == nil {
			http.Error(w, "contact not found", http.StatusNotFound)
			return
		}

		roles := make([]RoleWithCompanyResponse, 0, len(details.Roles))
		for _, rc := range details.Roles {
			roles = append(roles, RoleWithCompanyResponse{
				Role: RoleResponse{
					ID:         rc.Role.ID,
					CompanyID:  rc.Role.CompanyID,
					Slug:       rc.Role.Slug,
					Title:      rc.Role.Title,
					FolderPath: rc.Role.FolderPath,
				},
				Company: CompanyResponse{
					ID:         rc.Company.ID,
					Slug:       rc.Company.Slug,
					Name:       rc.Company.Name,
					FolderPath: rc.Company.FolderPath,
				},
			})
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ContactResponse{
			ID:          details.Contact.ID,
			Name:        details.Contact.Name,
			Org:         details.Contact.Org,
			LinkedInURL: details.Contact.LinkedInURL,
			Email:       details.Contact.Email,
			Code:        details.Contact.Code,
			Slug:        details.Contact.Slug,
			FolderPath:  details.Contact.FolderPath,
			Roles:       roles,
		})
	}
}

// LinkContactRoleRequest is the request body for linking a role to a contact
type LinkContactRoleRequest struct {
	RoleRef string `json:"role_ref"` // Format: "companySlug/roleSlug"
}

// HandleLinkRoleToContactAPI handles POST /api/contacts/{id}/roles
func (h *Handlers) HandleLinkRoleToContactAPI() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		contactID := chi.URLParam(r, "id")

		var req LinkContactRoleRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		if req.RoleRef == "" {
			http.Error(w, "role_ref is required", http.StatusBadRequest)
			return
		}

		parts := splitRoleRef(req.RoleRef)
		if len(parts) != 2 {
			http.Error(w, "role_ref must be in format 'companySlug/roleSlug'", http.StatusBadRequest)
			return
		}

		err := h.contactService.LinkRole(r.Context(), app.LinkContactRoleInput{
			ContactID:   contactID,
			CompanySlug: parts[0],
			RoleSlug:    parts[1],
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

// CreateContactMeetingRequest is the request body for creating a contact meeting
type CreateContactMeetingRequest struct {
	OccurredAt string `json:"occurred_at"`
	Title      string `json:"title"`
}

// HandleCreateContactMeetingAPI handles POST /api/contacts/{id}/meetings
func (h *Handlers) HandleCreateContactMeetingAPI() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		contactID := chi.URLParam(r, "id")

		var req CreateContactMeetingRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		if req.Title == "" || req.OccurredAt == "" {
			http.Error(w, "title and occurred_at are required", http.StatusBadRequest)
			return
		}

		meeting, err := h.meetingV2Service.CreateContactMeeting(r.Context(), app.CreateContactMeetingInput{
			ContactID:  contactID,
			OccurredAt: req.OccurredAt,
			Title:      req.Title,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(MeetingV2Response{
			ID:         meeting.ID,
			OccurredAt: meeting.OccurredAt.Format(time.RFC3339),
			Title:      meeting.Title,
			PathMD:     meeting.PathMD,
		})
	}
}

func splitRoleRef(ref string) []string {
	for i := 0; i < len(ref); i++ {
		if ref[i] == '/' {
			return []string{ref[:i], ref[i+1:]}
		}
	}
	return []string{ref}
}

// --- Meeting Handlers ---

// CreateMeetingRequest is the request body for creating a meeting
type CreateMeetingRequest struct {
	CompanySlug string `json:"company_slug"`
	OccurredAt  string `json:"occurred_at"`
	Title       string `json:"title"`
}

// HandleCreateMeeting handles POST /api/meetings
func (h *Handlers) HandleCreateMeeting() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req CreateMeetingRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		if req.CompanySlug == "" || req.OccurredAt == "" || req.Title == "" {
			http.Error(w, "company_slug, occurred_at, and title are required", http.StatusBadRequest)
			return
		}

		meeting, err := h.meetingService.CreateMeeting(r.Context(), app.CreateMeetingInput{
			CompanySlug: req.CompanySlug,
			OccurredAt:  req.OccurredAt,
			Title:       req.Title,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(MeetingResponse{
			ID:         meeting.ID,
			OccurredAt: meeting.OccurredAt.Format(time.RFC3339),
			Title:      meeting.Title,
			CompanyID:  meeting.CompanyID,
			PathMD:     meeting.PathMD,
		})
	}
}

// --- JD Handlers ---

// JDResponse is the response for a job description
type JDResponse struct {
	RoleID   string `json:"role_id"`
	PathHTML string `json:"path_html,omitempty"`
	PathPDF  string `json:"path_pdf,omitempty"`
}

// HandleAttachJD handles POST /api/roles/{companySlug}/{roleSlug}/jd
func (h *Handlers) HandleAttachJD() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		companySlug := chi.URLParam(r, "companySlug")
		roleSlug := chi.URLParam(r, "roleSlug")

		// Parse multipart form (max 10MB)
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			http.Error(w, "failed to parse multipart form: "+err.Error(), http.StatusBadRequest)
			return
		}

		var htmlContent string
		var pdfReader *multipartFileReader

		// Get HTML content
		htmlContent = r.FormValue("html")

		// Get PDF file if provided
		pdfFile, _, err := r.FormFile("pdf")
		if err == nil {
			defer pdfFile.Close()
			pdfReader = &multipartFileReader{pdfFile}
		}

		if htmlContent == "" && pdfReader == nil {
			http.Error(w, "at least html or pdf must be provided", http.StatusBadRequest)
			return
		}

		input := app.AttachJDInput{
			CompanySlug: companySlug,
			RoleSlug:    roleSlug,
			HTMLContent: htmlContent,
		}
		if pdfReader != nil {
			input.PDFContent = pdfReader
		}

		jd, err := h.jdService.AttachJD(r.Context(), input)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(JDResponse{
			RoleID:   jd.RoleID,
			PathHTML: jd.PathHTML,
			PathPDF:  jd.PathPDF,
		})
	}
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

// --- Export Handler ---

// HandleExport handles POST /api/export
func (h *Handlers) HandleExport() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := h.exportService.Export(r.Context()); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "exported"})
	}
}

// --- HTML Page Handlers ---

// CompanyListItem represents a company for the list view
type CompanyListItem struct {
	Company   *app.CompanyWithDetails
	LastTouch string
}

// HandleCompaniesPage handles GET /companies (HTML page)
func (h *Handlers) HandleCompaniesPage() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		companies, err := h.companyService.ListCompanies(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Build list with last touch info
		items := make([]struct {
			Company   interface{}
			Status    string
			LastTouch string
		}, 0, len(companies))

		for _, c := range companies {
			// Get meetings from all roles to find last touch
			var lastTouch string
			var latest time.Time
			for _, role := range c.Roles {
				meetings, _ := h.meetingV2Service.ListMeetingsByRole(r.Context(), role.ID)
				for _, m := range meetings {
					if m.OccurredAt.After(latest) {
						latest = m.OccurredAt
					}
				}
			}
			if !latest.IsZero() {
				lastTouch = latest.Format("2006-01-02")
			}

			items = append(items, struct {
				Company   interface{}
				Status    string
				LastTouch string
			}{
				Company:   c.Company,
				Status:    c.Status.String(),
				LastTouch: lastTouch,
			})
		}

		// Sort by last touch descending (most recent first, empty dates last)
		sort.Slice(items, func(i, j int) bool {
			if items[i].LastTouch == "" {
				return false
			}
			if items[j].LastTouch == "" {
				return true
			}
			return items[i].LastTouch > items[j].LastTouch
		})

		// Check for success/error query params (from redirects)
		successMsg := r.URL.Query().Get("success")
		errorMsg := r.URL.Query().Get("error")

		data := map[string]interface{}{
			"Title":     "Companies",
			"Companies": items,
			"Success":   successMsg,
			"Error":     errorMsg,
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := h.views.Render(w, "companies", data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

// HandleCompanyPage handles GET /companies/{slug} (HTML page)
func (h *Handlers) HandleCompanyPage() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		slug := chi.URLParam(r, "slug")

		company, err := h.companyService.GetCompany(r.Context(), slug)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if company == nil {
			http.Error(w, "company not found", http.StatusNotFound)
			return
		}

		// Check for success/error query params
		successMsg := r.URL.Query().Get("success")
		errorMsg := r.URL.Query().Get("error")

		data := map[string]interface{}{
			"Title":    company.Company.Name,
			"Company":  company.Company,
			"Roles":    company.Roles,
			"Meetings": company.Meetings,
			"Status":   company.Status.String(),
			"Success":  successMsg,
			"Error":    errorMsg,
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := h.views.Render(w, "company", data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

// RoleDropdownItem represents a role for dropdown display
type RoleDropdownItem struct {
	CompanySlug string
	CompanyName string
	RoleSlug    string
	RoleTitle   string
}

// getAllRolesForDropdown returns all roles formatted for dropdown
func (h *Handlers) getAllRolesForDropdown(ctx context.Context) []RoleDropdownItem {
	companies, _ := h.companyService.ListCompanies(ctx)
	var roles []RoleDropdownItem
	for _, c := range companies {
		details, _ := h.companyService.GetCompany(ctx, c.Company.Slug)
		if details != nil {
			for _, r := range details.Roles {
				roles = append(roles, RoleDropdownItem{
					CompanySlug: c.Company.Slug,
					CompanyName: c.Company.Name,
					RoleSlug:    r.Slug,
					RoleTitle:   r.Title,
				})
			}
		}
	}
	return roles
}

// --- HTML Form POST Handlers ---

// HandleCreateCompanyForm handles POST /companies/new (HTML form)
func (h *Handlers) HandleCreateCompanyForm() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			h.renderCompaniesPageWithError(w, r, "Invalid form data", "", "")
			return
		}

		slug := r.FormValue("slug")
		name := r.FormValue("name")

		if slug == "" || name == "" {
			h.renderCompaniesPageWithError(w, r, "Slug and name are required", slug, name)
			return
		}

		_, err := h.companyService.CreateCompany(r.Context(), app.CreateCompanyInput{
			Slug: slug,
			Name: name,
		})
		if err != nil {
			h.renderCompaniesPageWithError(w, r, err.Error(), slug, name)
			return
		}

		http.Redirect(w, r, "/companies", http.StatusSeeOther)
	}
}

// renderCompaniesPageWithError renders the companies page with an error message
func (h *Handlers) renderCompaniesPageWithError(w http.ResponseWriter, r *http.Request, errMsg, formSlug, formName string) {
	companies, _ := h.companyService.ListCompanies(r.Context())

	items := make([]struct {
		Company   interface{}
		Status    string
		LastTouch string
	}, 0, len(companies))

	for _, c := range companies {
		var lastTouch string
		var latest time.Time
		for _, role := range c.Roles {
			meetings, _ := h.meetingV2Service.ListMeetingsByRole(r.Context(), role.ID)
			for _, m := range meetings {
				if m.OccurredAt.After(latest) {
					latest = m.OccurredAt
				}
			}
		}
		if !latest.IsZero() {
			lastTouch = latest.Format("2006-01-02")
		}

		items = append(items, struct {
			Company   interface{}
			Status    string
			LastTouch string
		}{
			Company:   c.Company,
			Status:    c.Status.String(),
			LastTouch: lastTouch,
		})
	}

	// Sort by last touch descending (most recent first, empty dates last)
	sort.Slice(items, func(i, j int) bool {
		if items[i].LastTouch == "" {
			return false
		}
		if items[j].LastTouch == "" {
			return true
		}
		return items[i].LastTouch > items[j].LastTouch
	})

	data := map[string]interface{}{
		"Title":     "Companies",
		"Companies": items,
		"Error":     errMsg,
		"FormSlug":  formSlug,
		"FormName":  formName,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	h.views.Render(w, "companies", data)
}

// HandleExportPage handles POST /export (HTML form, redirects with success message)
func (h *Handlers) HandleExportPage() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := h.exportService.Export(r.Context()); err != nil {
			// Redirect back with error
			http.Redirect(w, r, "/companies?error=Export+failed:+"+err.Error(), http.StatusSeeOther)
			return
		}

		// Redirect back with success
		http.Redirect(w, r, "/companies?success=Export+completed+successfully", http.StatusSeeOther)
	}
}

// HandleCreateRoleForm handles POST /companies/{slug}/roles/new (HTML form)
func (h *Handlers) HandleCreateRoleForm() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		companySlug := chi.URLParam(r, "slug")

		if err := r.ParseForm(); err != nil {
			http.Redirect(w, r, "/companies/"+companySlug+"?error=Invalid+form+data", http.StatusSeeOther)
			return
		}

		slug := r.FormValue("slug")
		title := r.FormValue("title")

		if slug == "" || title == "" {
			http.Redirect(w, r, "/companies/"+companySlug+"?error=Slug+and+title+are+required", http.StatusSeeOther)
			return
		}

		_, err := h.companyService.CreateRole(r.Context(), app.CreateRoleInput{
			CompanySlug: companySlug,
			Slug:        slug,
			Title:       title,
		})
		if err != nil {
			http.Redirect(w, r, "/companies/"+companySlug+"?error="+err.Error(), http.StatusSeeOther)
			return
		}

		http.Redirect(w, r, "/companies/"+companySlug+"?success=Role+created", http.StatusSeeOther)
	}
}

// HandleCreateMeetingForm handles POST /companies/{slug}/meetings/new (HTML form)
func (h *Handlers) HandleCreateMeetingForm() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		companySlug := chi.URLParam(r, "slug")

		if err := r.ParseForm(); err != nil {
			http.Redirect(w, r, "/companies/"+companySlug+"?error=Invalid+form+data", http.StatusSeeOther)
			return
		}

		title := r.FormValue("title")
		occurredAt := r.FormValue("occurred_at")

		if title == "" || occurredAt == "" {
			http.Redirect(w, r, "/companies/"+companySlug+"?error=Title+and+date+are+required", http.StatusSeeOther)
			return
		}

		// Convert datetime-local to RFC3339
		// datetime-local format is "2024-01-15T10:00"
		occurredAtRFC := occurredAt + ":00Z"

		_, err := h.meetingService.CreateMeeting(r.Context(), app.CreateMeetingInput{
			CompanySlug: companySlug,
			OccurredAt:  occurredAtRFC,
			Title:       title,
		})
		if err != nil {
			http.Redirect(w, r, "/companies/"+companySlug+"?error="+err.Error(), http.StatusSeeOther)
			return
		}

		http.Redirect(w, r, "/companies/"+companySlug+"?success=Meeting+created", http.StatusSeeOther)
	}
}

// HandleContactsPage handles GET /contacts (HTML page)
func (h *Handlers) HandleContactsPage() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		contacts, err := h.contactService.ListContacts(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		successMsg := r.URL.Query().Get("success")
		errorMsg := r.URL.Query().Get("error")

		data := map[string]interface{}{
			"Title":    "Contacts",
			"Contacts": contacts,
			"Success":  successMsg,
			"Error":    errorMsg,
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := h.views.Render(w, "contacts", data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

// HandleContactPage handles GET /contacts/{id} (HTML page)
func (h *Handlers) HandleContactPage() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")

		details, err := h.contactService.GetContactWithDetails(r.Context(), id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if details == nil {
			http.Error(w, "contact not found", http.StatusNotFound)
			return
		}

		// Get contact meetings
		contactMeetings, err := h.meetingV2Service.ListMeetingsByContact(r.Context(), id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Get role meetings grouped by role
		type RoleMeetingsGroup struct {
			Role     *domain.Role
			Company  *domain.Company
			Meetings []*domain.MeetingV2
		}
		var roleMeetingsGroups []RoleMeetingsGroup
		for _, rc := range details.Roles {
			meetings, err := h.meetingV2Service.ListMeetingsByRole(r.Context(), rc.Role.ID)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			roleMeetingsGroups = append(roleMeetingsGroups, RoleMeetingsGroup{
				Role:     rc.Role,
				Company:  rc.Company,
				Meetings: meetings,
			})
		}

		allRoles := h.getAllRolesForDropdown(r.Context())

		successMsg := r.URL.Query().Get("success")
		errorMsg := r.URL.Query().Get("error")

		data := map[string]interface{}{
			"Title":              details.Contact.Name,
			"Contact":            details.Contact,
			"Roles":              details.Roles,
			"Meetings":           contactMeetings,
			"RoleMeetingsGroups": roleMeetingsGroups,
			"AllRoles":           allRoles,
			"Success":            successMsg,
			"Error":              errorMsg,
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := h.views.Render(w, "contact", data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

// HandleLinkRoleToContactForm handles POST /contacts/{id}/roles/link (HTML form)
func (h *Handlers) HandleLinkRoleToContactForm() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		contactID := chi.URLParam(r, "id")

		if err := r.ParseForm(); err != nil {
			http.Redirect(w, r, "/contacts/"+contactID+"?error=Invalid+form+data", http.StatusSeeOther)
			return
		}

		roleRef := r.FormValue("role_ref")
		if roleRef == "" {
			http.Redirect(w, r, "/contacts/"+contactID+"?error=Role+is+required", http.StatusSeeOther)
			return
		}

		parts := splitRoleRef(roleRef)
		if len(parts) != 2 {
			http.Redirect(w, r, "/contacts/"+contactID+"?error=Invalid+role+reference", http.StatusSeeOther)
			return
		}

		err := h.contactService.LinkRole(r.Context(), app.LinkContactRoleInput{
			ContactID:   contactID,
			CompanySlug: parts[0],
			RoleSlug:    parts[1],
		})
		if err != nil {
			http.Redirect(w, r, "/contacts/"+contactID+"?error="+err.Error(), http.StatusSeeOther)
			return
		}

		http.Redirect(w, r, "/contacts/"+contactID+"?success=Role+linked", http.StatusSeeOther)
	}
}

// HandleCreateContactMeetingForm handles POST /contacts/{id}/meetings/new (HTML form)
func (h *Handlers) HandleCreateContactMeetingForm() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		contactID := chi.URLParam(r, "id")
		redirectURL := "/contacts/" + contactID

		if err := r.ParseForm(); err != nil {
			http.Redirect(w, r, redirectURL+"?error=Failed+to+parse+form", http.StatusSeeOther)
			return
		}

		title := r.FormValue("title")
		occurredAt := r.FormValue("occurred_at")

		if title == "" || occurredAt == "" {
			http.Redirect(w, r, redirectURL+"?error=Title+and+date+are+required", http.StatusSeeOther)
			return
		}

		occurredAtRFC := occurredAt + ":00Z"

		_, err := h.meetingV2Service.CreateContactMeeting(r.Context(), app.CreateContactMeetingInput{
			ContactID:  contactID,
			OccurredAt: occurredAtRFC,
			Title:      title,
		})
		if err != nil {
			http.Redirect(w, r, redirectURL+"?error="+err.Error(), http.StatusSeeOther)
			return
		}

		http.Redirect(w, r, redirectURL+"?success=Meeting+created", http.StatusSeeOther)
	}
}

// HandleCreateContactForm handles POST /contacts/new (HTML form)
func (h *Handlers) HandleCreateContactForm() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Redirect(w, r, "/contacts?error=Invalid+form+data", http.StatusSeeOther)
			return
		}

		name := r.FormValue("name")
		org := r.FormValue("org")
		email := r.FormValue("email")
		linkedinURL := r.FormValue("linkedin_url")

		if name == "" {
			http.Redirect(w, r, "/contacts?error=Name+is+required", http.StatusSeeOther)
			return
		}

		_, err := h.contactService.CreateContact(r.Context(), app.CreateContactInput{
			Name:        name,
			Org:         org,
			Email:       email,
			LinkedInURL: linkedinURL,
		})
		if err != nil {
			http.Redirect(w, r, "/contacts?error="+err.Error(), http.StatusSeeOther)
			return
		}

		http.Redirect(w, r, "/contacts?success=Contact+created", http.StatusSeeOther)
	}
}

// HandleRolePage handles GET /companies/{companySlug}/roles/{roleSlug} (HTML page)
func (h *Handlers) HandleRolePage() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		companySlug := chi.URLParam(r, "companySlug")
		roleSlug := chi.URLParam(r, "roleSlug")

		// Get company
		company, err := h.companyService.GetCompany(r.Context(), companySlug)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if company == nil {
			http.Error(w, "company not found", http.StatusNotFound)
			return
		}

		// Find the role
		var role *domain.Role
		for _, ro := range company.Roles {
			if ro.Slug == roleSlug {
				role = ro
				break
			}
		}
		if role == nil {
			http.Error(w, "role not found", http.StatusNotFound)
			return
		}

		// Get meetings for this role (v2)
		meetings, err := h.meetingV2Service.ListMeetingsByRole(r.Context(), role.ID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Get JD info
		jd, _ := h.jdService.GetJD(r.Context(), companySlug, roleSlug)
		jdData := struct {
			PathHTML string
			PathPDF  string
		}{}
		if jd != nil {
			jdData.PathHTML = jd.PathHTML
			jdData.PathPDF = jd.PathPDF
		}

		// Get Resume info
		resume, _ := h.resumeService.GetResume(r.Context(), companySlug, roleSlug)
		resumeData := struct {
			PathJSON string
			PathPDF  string
		}{}
		if resume != nil {
			resumeData.PathJSON = resume.PathJSON
			resumeData.PathPDF = resume.PathPDF
		}

		// Get artifacts
		artifacts, _ := h.artifactService.ListArtifacts(r.Context(), companySlug, roleSlug)

		// Check for success/error query params
		successMsg := r.URL.Query().Get("success")
		errorMsg := r.URL.Query().Get("error")

		allStatuses := domain.AllRoleStatusesWithLabels()

		data := map[string]interface{}{
			"Title":       company.Company.Name + " - " + roleSlug,
			"Company":     company.Company,
			"Role":        role,
			"JD":          jdData,
			"Resume":      resumeData,
			"Artifacts":   artifacts,
			"AllStatuses": allStatuses,
			"Meetings":    meetings,
			"Success":     successMsg,
			"Error":       errorMsg,
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := h.views.Render(w, "role", data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

// HandleUpdateRoleStatusForm handles POST /companies/{companySlug}/roles/{roleSlug}/status (HTML form)
func (h *Handlers) HandleUpdateRoleStatusForm() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		companySlug := chi.URLParam(r, "companySlug")
		roleSlug := chi.URLParam(r, "roleSlug")
		redirectURL := "/companies/" + companySlug + "/roles/" + roleSlug

		if err := r.ParseForm(); err != nil {
			http.Redirect(w, r, redirectURL+"?error=Failed+to+parse+form", http.StatusSeeOther)
			return
		}

		status := r.FormValue("status")
		if status == "" {
			http.Redirect(w, r, redirectURL+"?error=Status+is+required", http.StatusSeeOther)
			return
		}

		err := h.companyService.UpdateRoleStatus(r.Context(), app.UpdateRoleStatusInput{
			CompanySlug: companySlug,
			RoleSlug:    roleSlug,
			Status:      status,
		})
		if err != nil {
			http.Redirect(w, r, redirectURL+"?error="+url.QueryEscape(err.Error()), http.StatusSeeOther)
			return
		}

		http.Redirect(w, r, redirectURL+"?success=Status+updated", http.StatusSeeOther)
	}
}

// HandleAttachJDForm handles POST /companies/{companySlug}/roles/{roleSlug}/jd (HTML form)
func (h *Handlers) HandleAttachJDForm() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		companySlug := chi.URLParam(r, "companySlug")
		roleSlug := chi.URLParam(r, "roleSlug")
		redirectURL := "/companies/" + companySlug + "/roles/" + roleSlug

		// Parse multipart form (max 10MB)
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			http.Redirect(w, r, redirectURL+"?error=Failed+to+parse+form", http.StatusSeeOther)
			return
		}

		htmlContent := r.FormValue("html")

		// Get PDF file if provided
		var pdfReader *multipartFileReader
		pdfFile, _, err := r.FormFile("pdf")
		if err == nil {
			defer pdfFile.Close()
			pdfReader = &multipartFileReader{pdfFile}
		}

		if htmlContent == "" && pdfReader == nil {
			http.Redirect(w, r, redirectURL+"?error=At+least+HTML+or+PDF+must+be+provided", http.StatusSeeOther)
			return
		}

		input := app.AttachJDInput{
			CompanySlug: companySlug,
			RoleSlug:    roleSlug,
			HTMLContent: htmlContent,
		}
		if pdfReader != nil {
			input.PDFContent = pdfReader
		}

		_, err = h.jdService.AttachJD(r.Context(), input)
		if err != nil {
			http.Redirect(w, r, redirectURL+"?error="+err.Error(), http.StatusSeeOther)
			return
		}

		http.Redirect(w, r, redirectURL+"?success=Job+description+attached", http.StatusSeeOther)
	}
}

// HandleAttachResumeForm handles POST /companies/{companySlug}/roles/{roleSlug}/resume (HTML form)
func (h *Handlers) HandleAttachResumeForm() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		companySlug := chi.URLParam(r, "companySlug")
		roleSlug := chi.URLParam(r, "roleSlug")
		redirectURL := "/companies/" + companySlug + "/roles/" + roleSlug

		// Parse multipart form (max 20MB for resumes)
		if err := r.ParseMultipartForm(20 << 20); err != nil {
			http.Redirect(w, r, redirectURL+"?error=Failed+to+parse+form", http.StatusSeeOther)
			return
		}

		// Get JSON content from textarea
		jsonContent := r.FormValue("resume_json")

		// Get PDF file if provided
		var pdfReader *multipartFileReader
		pdfFile, _, err := r.FormFile("pdf")
		if err == nil {
			defer pdfFile.Close()
			pdfReader = &multipartFileReader{pdfFile}
		}

		if jsonContent == "" && pdfReader == nil {
			http.Redirect(w, r, redirectURL+"?error=At+least+JSON+or+PDF+must+be+provided", http.StatusSeeOther)
			return
		}

		input := app.AttachResumeInput{
			CompanySlug: companySlug,
			RoleSlug:    roleSlug,
			JSONContent: jsonContent,
		}
		if pdfReader != nil {
			input.PDFContent = pdfReader
		}

		_, err = h.resumeService.AttachResume(r.Context(), input)
		if err != nil {
			http.Redirect(w, r, redirectURL+"?error="+url.QueryEscape(err.Error()), http.StatusSeeOther)
			return
		}

		http.Redirect(w, r, redirectURL+"?success=Resume+attached", http.StatusSeeOther)
	}
}

// HandleViewJD handles GET /companies/{companySlug}/roles/{roleSlug}/jd (JD viewer page)
func (h *Handlers) HandleViewJD() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		companySlug := chi.URLParam(r, "companySlug")
		roleSlug := chi.URLParam(r, "roleSlug")

		company, err := h.companyService.GetCompany(r.Context(), companySlug)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if company == nil {
			http.Error(w, "company not found", http.StatusNotFound)
			return
		}

		// Find the role
		var role *domain.Role
		for _, r := range company.Roles {
			if r.Slug == roleSlug {
				role = r
				break
			}
		}
		if role == nil {
			http.Error(w, "role not found", http.StatusNotFound)
			return
		}

		// Get JD
		jd, err := h.jdService.GetJobDescription(r.Context(), role.ID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if jd == nil || jd.PathHTML == "" {
			http.Error(w, "no job description HTML attached to this role", http.StatusNotFound)
			return
		}

		data := map[string]interface{}{
			"Title":   role.Title + " - Job Description",
			"Company": company.Company,
			"Role":    role,
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := h.views.Render(w, "jd_viewer", data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

// HandleViewJDRaw handles GET /companies/{companySlug}/roles/{roleSlug}/jd/raw (raw HTML with CSP)
func (h *Handlers) HandleViewJDRaw() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		companySlug := chi.URLParam(r, "companySlug")
		roleSlug := chi.URLParam(r, "roleSlug")

		company, err := h.companyService.GetCompany(r.Context(), companySlug)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if company == nil {
			http.Error(w, "company not found", http.StatusNotFound)
			return
		}

		// Find the role
		var role *domain.Role
		for _, r := range company.Roles {
			if r.Slug == roleSlug {
				role = r
				break
			}
		}
		if role == nil {
			http.Error(w, "role not found", http.StatusNotFound)
			return
		}

		// Get JD
		jd, err := h.jdService.GetJobDescription(r.Context(), role.ID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if jd == nil || jd.PathHTML == "" {
			http.Error(w, "no job description HTML attached to this role", http.StatusNotFound)
			return
		}

		// Read the HTML file
		htmlContent, err := h.jdService.ReadJobDescriptionHTML(r.Context(), jd.PathHTML)
		if err != nil {
			http.Error(w, "failed to read job description: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Set strict security headers
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		// Strict CSP that blocks scripts and external resources
		w.Header().Set("Content-Security-Policy", "default-src 'none'; img-src 'self' data:; style-src 'self' 'unsafe-inline'; font-src 'self' data:; frame-ancestors 'self'; base-uri 'none'; form-action 'none'")

		w.Write([]byte(htmlContent))
	}
}

// --- MeetingV2 Handlers ---

// CreateRoleMeetingRequest is the request body for creating a role meeting (v2)
type CreateRoleMeetingRequest struct {
	OccurredAt string `json:"occurred_at"`
	Title      string `json:"title"`
}

// HandleCreateRoleMeetingV2 handles POST /api/companies/{companySlug}/roles/{roleSlug}/meetings (JSON)
func (h *Handlers) HandleCreateRoleMeetingV2() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		companySlug := chi.URLParam(r, "companySlug")
		roleSlug := chi.URLParam(r, "roleSlug")

		var req CreateRoleMeetingRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		if req.Title == "" || req.OccurredAt == "" {
			http.Error(w, "title and occurred_at are required", http.StatusBadRequest)
			return
		}

		meeting, err := h.meetingV2Service.CreateRoleMeeting(r.Context(), app.CreateRoleMeetingInput{
			CompanySlug: companySlug,
			RoleSlug:    roleSlug,
			OccurredAt:  req.OccurredAt,
			Title:       req.Title,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(MeetingV2Response{
			ID:         meeting.ID,
			OccurredAt: meeting.OccurredAt.Format(time.RFC3339),
			Title:      meeting.Title,
			RoleID:     meeting.RoleID,
			PathMD:     meeting.PathMD,
		})
	}
}


// HandleCreateRoleMeetingV2Form handles POST /companies/{companySlug}/roles/{roleSlug}/meetings/new (HTML form)
func (h *Handlers) HandleCreateRoleMeetingV2Form() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		companySlug := chi.URLParam(r, "companySlug")
		roleSlug := chi.URLParam(r, "roleSlug")
		redirectURL := "/companies/" + companySlug + "/roles/" + roleSlug

		if err := r.ParseForm(); err != nil {
			http.Redirect(w, r, redirectURL+"?error=Failed+to+parse+form", http.StatusSeeOther)
			return
		}

		title := r.FormValue("title")
		occurredAt := r.FormValue("occurred_at")

		if title == "" || occurredAt == "" {
			http.Redirect(w, r, redirectURL+"?error=Title+and+date+are+required", http.StatusSeeOther)
			return
		}

		// Convert datetime-local to RFC3339
		occurredAtRFC := occurredAt + ":00Z"

		_, err := h.meetingV2Service.CreateRoleMeeting(r.Context(), app.CreateRoleMeetingInput{
			CompanySlug: companySlug,
			RoleSlug:    roleSlug,
			OccurredAt:  occurredAtRFC,
			Title:       title,
		})
		if err != nil {
			http.Redirect(w, r, redirectURL+"?error="+url.QueryEscape(err.Error()), http.StatusSeeOther)
			return
		}

		http.Redirect(w, r, redirectURL+"?success=Meeting+created", http.StatusSeeOther)
	}
}


// --- Artifact Handlers ---

// HandleUpsertArtifactForm handles POST /companies/{companySlug}/roles/{roleSlug}/artifacts (HTML form)
func (h *Handlers) HandleUpsertArtifactForm() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		companySlug := chi.URLParam(r, "companySlug")
		roleSlug := chi.URLParam(r, "roleSlug")
		redirectURL := "/companies/" + companySlug + "/roles/" + roleSlug

		// Parse multipart form (max 20MB)
		if err := r.ParseMultipartForm(20 << 20); err != nil {
			http.Redirect(w, r, redirectURL+"?error=Failed+to+parse+form", http.StatusSeeOther)
			return
		}

		name := r.FormValue("name")
		artifactType := r.FormValue("type")
		textContent := r.FormValue("content")

		if name == "" {
			http.Redirect(w, r, redirectURL+"?error=Artifact+name+is+required", http.StatusSeeOther)
			return
		}

		input := app.UpsertArtifactInput{
			CompanySlug: companySlug,
			RoleSlug:    roleSlug,
			Name:        name,
			Type:        artifactType,
		}

		switch artifactType {
		case "pdf", "png":
			file, _, err := r.FormFile("file")
			if err != nil {
				http.Redirect(w, r, redirectURL+"?error=File+is+required+for+"+artifactType+"+artifacts", http.StatusSeeOther)
				return
			}
			defer file.Close()
			input.FileContent = file
		case "text", "jsonc", "html", "markdown":
			if textContent == "" {
				http.Redirect(w, r, redirectURL+"?error=Content+is+required+for+"+artifactType+"+artifacts", http.StatusSeeOther)
				return
			}
			input.TextContent = textContent
		case "file":
			file, header, err := r.FormFile("file")
			if err != nil {
				http.Redirect(w, r, redirectURL+"?error=File+is+required+for+file+artifacts", http.StatusSeeOther)
				return
			}
			defer file.Close()
			input.FileContent = file
			input.FileExtension = filepath.Ext(header.Filename)
		default:
			http.Redirect(w, r, redirectURL+"?error=Invalid+artifact+type", http.StatusSeeOther)
			return
		}

		_, err := h.artifactService.UpsertArtifact(r.Context(), input)
		if err != nil {
			http.Redirect(w, r, redirectURL+"?error="+url.QueryEscape(err.Error()), http.StatusSeeOther)
			return
		}

		http.Redirect(w, r, redirectURL+"?success=Artifact+saved", http.StatusSeeOther)
	}
}

// HandleViewArtifact handles GET /companies/{companySlug}/roles/{roleSlug}/artifacts/{name}
func (h *Handlers) HandleViewArtifact() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		companySlug := chi.URLParam(r, "companySlug")
		roleSlug := chi.URLParam(r, "roleSlug")
		name := chi.URLParam(r, "name")

		// Get artifact by name
		artifact, err := h.artifactService.GetArtifactByName(r.Context(), companySlug, roleSlug, name)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if artifact == nil {
			http.Error(w, "artifact not found", http.StatusNotFound)
			return
		}

		// Read artifact content
		content, err := h.artifactService.ReadArtifactContent(r.Context(), artifact.Path)
		if err != nil {
			http.Error(w, "failed to read artifact: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Set security headers
		w.Header().Set("X-Content-Type-Options", "nosniff")

		// Serve based on type
		switch artifact.Type {
		case "pdf":
			w.Header().Set("Content-Type", "application/pdf")
			w.Header().Set("Content-Disposition", "inline; filename=\""+name+".pdf\"")
			w.Header().Set("Content-Length", strconv.Itoa(len(content)))
			w.Write(content)
		case "png":
			w.Header().Set("Content-Type", "image/png")
			w.Header().Set("Content-Disposition", "inline; filename=\""+name+".png\"")
			w.Header().Set("Content-Length", strconv.Itoa(len(content)))
			w.Write(content)
		case "jsonc":
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'")
			w.Write(content)
		case "text":
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'")
			w.Write(content)
		case "html":
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'")
			w.Write(content)
		case "markdown":
			// Convert markdown to HTML
			var buf bytes.Buffer
			if err := goldmark.Convert(content, &buf); err != nil {
				http.Error(w, "failed to render markdown: "+err.Error(), http.StatusInternalServerError)
				return
			}
			// Wrap in styled HTML template
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'")
			w.Write([]byte(`<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<style>
body {
	font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif;
	line-height: 1.6;
	max-width: 800px;
	margin: 0 auto;
	padding: 20px;
	color: #333;
}
h1, h2, h3, h4, h5, h6 { margin-top: 1.5em; margin-bottom: 0.5em; }
h1 { font-size: 2em; border-bottom: 1px solid #eee; padding-bottom: 0.3em; }
h2 { font-size: 1.5em; border-bottom: 1px solid #eee; padding-bottom: 0.3em; }
code { background: #f4f4f4; padding: 2px 6px; border-radius: 3px; font-size: 0.9em; }
pre { background: #f4f4f4; padding: 16px; border-radius: 6px; overflow-x: auto; }
pre code { background: none; padding: 0; }
blockquote { border-left: 4px solid #ddd; margin: 0; padding-left: 16px; color: #666; }
a { color: #0366d6; }
ul, ol { padding-left: 2em; }
table { border-collapse: collapse; width: 100%; }
th, td { border: 1px solid #ddd; padding: 8px; text-align: left; }
th { background: #f4f4f4; }
</style>
</head>
<body>
`))
			w.Write(buf.Bytes())
			w.Write([]byte(`
</body>
</html>`))
		case "file":
			ext := filepath.Ext(artifact.Path)
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Header().Set("Content-Disposition", "attachment; filename=\""+name+ext+"\"")
			w.Header().Set("Content-Length", strconv.Itoa(len(content)))
			w.Write(content)
		default:
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.Write(content)
		}
	}
}

// HandleDeleteArtifact handles POST /companies/{companySlug}/roles/{roleSlug}/artifacts/{name}/delete
func (h *Handlers) HandleDeleteArtifact() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		companySlug := chi.URLParam(r, "companySlug")
		roleSlug := chi.URLParam(r, "roleSlug")
		name := chi.URLParam(r, "name")
		redirectURL := "/companies/" + companySlug + "/roles/" + roleSlug

		err := h.artifactService.DeleteArtifact(r.Context(), companySlug, roleSlug, name)
		if err != nil {
			http.Redirect(w, r, redirectURL+"?error="+url.QueryEscape(err.Error()), http.StatusSeeOther)
			return
		}

		http.Redirect(w, r, redirectURL+"?success=Artifact+deleted", http.StatusSeeOther)
	}
}
