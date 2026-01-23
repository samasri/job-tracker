package http

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"time"

	"jobtracker/internal/app"
	"jobtracker/internal/http/views"

	"github.com/go-chi/chi/v5"
)

// Handlers holds all HTTP handler dependencies
type Handlers struct {
	companyService *app.CompanyService
	contactService *app.ContactService
	threadService  *app.ThreadService
	meetingService *app.MeetingService
	jdService      *app.JDService
	exportService  *app.ExportService
	views          *views.Views
}

// NewHandlers creates a new Handlers instance
func NewHandlers(
	companyService *app.CompanyService,
	contactService *app.ContactService,
	threadService *app.ThreadService,
	meetingService *app.MeetingService,
	jdService *app.JDService,
	exportService *app.ExportService,
) *Handlers {
	v, err := views.New()
	if err != nil {
		panic("failed to parse templates: " + err.Error())
	}

	return &Handlers{
		companyService: companyService,
		contactService: contactService,
		threadService:  threadService,
		meetingService: meetingService,
		jdService:      jdService,
		exportService:  exportService,
		views:          v,
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

// MeetingResponse is the response for a meeting
type MeetingResponse struct {
	ID         string `json:"id"`
	OccurredAt string `json:"occurred_at"`
	Title      string `json:"title"`
	CompanyID  string `json:"company_id"`
	PathMD     string `json:"path_md"`
}

// ContactResponse is the response for a contact
type ContactResponse struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Org         string `json:"org,omitempty"`
	LinkedInURL string `json:"linkedin_url,omitempty"`
	Email       string `json:"email,omitempty"`
}

// ThreadResponse is the response for a thread
type ThreadResponse struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	ContactID string `json:"contact_id,omitempty"`
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

// ThreadWithDetailsResponse is a thread with its meetings and linked roles
type ThreadWithDetailsResponse struct {
	Thread   ThreadResponse            `json:"thread"`
	Meetings []MeetingResponse         `json:"meetings"`
	Roles    []RoleWithCompanyResponse `json:"roles"`
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
		})
	}
}

// --- Thread Handlers ---

// CreateThreadRequest is the request body for creating a thread
type CreateThreadRequest struct {
	Title     string `json:"title"`
	ContactID string `json:"contact_id"`
}

// HandleCreateThread handles POST /api/threads
func (h *Handlers) HandleCreateThread() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req CreateThreadRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		if req.Title == "" {
			http.Error(w, "title is required", http.StatusBadRequest)
			return
		}

		thread, err := h.threadService.CreateThread(r.Context(), app.CreateThreadInput{
			Title:     req.Title,
			ContactID: req.ContactID,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(ThreadResponse{
			ID:        thread.ID,
			Title:     thread.Title,
			ContactID: thread.ContactID,
		})
	}
}

// HandleGetThread handles GET /api/threads/{id}
func (h *Handlers) HandleGetThread() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")

		thread, err := h.threadService.GetThread(r.Context(), id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if thread == nil {
			http.Error(w, "thread not found", http.StatusNotFound)
			return
		}

		meetings := make([]MeetingResponse, 0, len(thread.Meetings))
		for _, m := range thread.Meetings {
			meetings = append(meetings, MeetingResponse{
				ID:         m.ID,
				OccurredAt: m.OccurredAt.Format(time.RFC3339),
				Title:      m.Title,
				CompanyID:  m.CompanyID,
				PathMD:     m.PathMD,
			})
		}

		roles := make([]RoleWithCompanyResponse, 0, len(thread.Roles))
		for _, rc := range thread.Roles {
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
		json.NewEncoder(w).Encode(ThreadWithDetailsResponse{
			Thread: ThreadResponse{
				ID:        thread.Thread.ID,
				Title:     thread.Thread.Title,
				ContactID: thread.Thread.ContactID,
			},
			Meetings: meetings,
			Roles:    roles,
		})
	}
}

// LinkRoleRequest is the request body for linking a role to a thread
type LinkRoleRequest struct {
	RoleRef string `json:"role_ref"` // Format: "companySlug/roleSlug"
}

// HandleLinkRole handles POST /api/threads/{id}/roles
func (h *Handlers) HandleLinkRole() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		threadID := chi.URLParam(r, "id")

		var req LinkRoleRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		if req.RoleRef == "" {
			http.Error(w, "role_ref is required", http.StatusBadRequest)
			return
		}

		// Parse role_ref (format: "companySlug/roleSlug")
		parts := splitRoleRef(req.RoleRef)
		if len(parts) != 2 {
			http.Error(w, "role_ref must be in format 'companySlug/roleSlug'", http.StatusBadRequest)
			return
		}

		err := h.threadService.LinkRole(r.Context(), app.LinkRoleInput{
			ThreadID:    threadID,
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
	ThreadID    string `json:"thread_id"`
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
			ThreadID:    req.ThreadID,
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
			// Get meetings to find last touch
			details, _ := h.companyService.GetCompany(r.Context(), c.Company.Slug)
			var lastTouch string
			if details != nil && len(details.Meetings) > 0 {
				// Find most recent meeting
				var latest time.Time
				for _, m := range details.Meetings {
					if m.OccurredAt.After(latest) {
						latest = m.OccurredAt
					}
				}
				lastTouch = latest.Format("Jan 02, 2006")
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

		// Get threads for the meeting form dropdown
		threads, _ := h.threadService.ListThreads(r.Context())

		// Check for success/error query params
		successMsg := r.URL.Query().Get("success")
		errorMsg := r.URL.Query().Get("error")

		data := map[string]interface{}{
			"Title":    company.Company.Name,
			"Company":  company.Company,
			"Roles":    company.Roles,
			"Meetings": company.Meetings,
			"Status":   company.Status.String(),
			"Threads":  threads,
			"Success":  successMsg,
			"Error":    errorMsg,
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := h.views.Render(w, "company", data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

// HandleThreadPage handles GET /threads/{id} (HTML page)
func (h *Handlers) HandleThreadPage() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")

		thread, err := h.threadService.GetThread(r.Context(), id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if thread == nil {
			http.Error(w, "thread not found", http.StatusNotFound)
			return
		}

		// Get all roles for the link role dropdown
		allRoles := h.getAllRolesForDropdown(r.Context())

		// Get all companies for the meeting form
		companies, _ := h.companyService.ListCompanies(r.Context())
		var companyList []struct {
			Slug string
			Name string
		}
		for _, c := range companies {
			companyList = append(companyList, struct {
				Slug string
				Name string
			}{Slug: c.Company.Slug, Name: c.Company.Name})
		}

		// Check for success/error query params
		successMsg := r.URL.Query().Get("success")
		errorMsg := r.URL.Query().Get("error")

		data := map[string]interface{}{
			"Title":     thread.Thread.Title,
			"Thread":    thread.Thread,
			"Meetings":  thread.Meetings,
			"Roles":     thread.Roles,
			"AllRoles":  allRoles,
			"Companies": companyList,
			"Success":   successMsg,
			"Error":     errorMsg,
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := h.views.Render(w, "thread", data); err != nil {
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
		details, _ := h.companyService.GetCompany(r.Context(), c.Company.Slug)
		var lastTouch string
		if details != nil && len(details.Meetings) > 0 {
			var latest time.Time
			for _, m := range details.Meetings {
				if m.OccurredAt.After(latest) {
					latest = m.OccurredAt
				}
			}
			lastTouch = latest.Format("Jan 02, 2006")
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
		threadID := r.FormValue("thread_id")

		if title == "" || occurredAt == "" {
			http.Redirect(w, r, "/companies/"+companySlug+"?error=Title+and+date+are+required", http.StatusSeeOther)
			return
		}

		// Convert datetime-local to RFC3339
		// datetime-local format is "2024-01-15T10:00"
		occurredAtRFC := occurredAt + ":00Z"

		_, err := h.meetingService.CreateMeeting(r.Context(), app.CreateMeetingInput{
			CompanySlug: companySlug,
			ThreadID:    threadID,
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

// ThreadWithContact represents a thread with its optional contact info
type ThreadWithContact struct {
	Thread  interface{}
	Contact interface{}
}

// HandleThreadsPage handles GET /threads (HTML page)
func (h *Handlers) HandleThreadsPage() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		threads, err := h.threadService.ListThreads(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Build thread list with contact info
		var threadList []ThreadWithContact
		for _, t := range threads {
			var contact interface{}
			if t.ContactID != "" {
				c, _ := h.contactService.GetContact(r.Context(), t.ContactID)
				contact = c
			}
			threadList = append(threadList, ThreadWithContact{
				Thread:  t,
				Contact: contact,
			})
		}

		// Get contacts for dropdown
		contacts, _ := h.contactService.ListContacts(r.Context())

		// Check for success/error query params
		successMsg := r.URL.Query().Get("success")
		errorMsg := r.URL.Query().Get("error")

		data := map[string]interface{}{
			"Title":    "Threads",
			"Threads":  threadList,
			"Contacts": contacts,
			"Success":  successMsg,
			"Error":    errorMsg,
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := h.views.Render(w, "threads", data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

// HandleCreateContactForm handles POST /contacts/new (HTML form)
func (h *Handlers) HandleCreateContactForm() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Redirect(w, r, "/threads?error=Invalid+form+data", http.StatusSeeOther)
			return
		}

		name := r.FormValue("name")
		org := r.FormValue("org")
		email := r.FormValue("email")
		linkedinURL := r.FormValue("linkedin_url")

		if name == "" {
			http.Redirect(w, r, "/threads?error=Name+is+required", http.StatusSeeOther)
			return
		}

		_, err := h.contactService.CreateContact(r.Context(), app.CreateContactInput{
			Name:        name,
			Org:         org,
			Email:       email,
			LinkedInURL: linkedinURL,
		})
		if err != nil {
			http.Redirect(w, r, "/threads?error="+err.Error(), http.StatusSeeOther)
			return
		}

		http.Redirect(w, r, "/threads?success=Contact+created", http.StatusSeeOther)
	}
}

// HandleCreateThreadForm handles POST /threads/new (HTML form)
func (h *Handlers) HandleCreateThreadForm() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Redirect(w, r, "/threads?error=Invalid+form+data", http.StatusSeeOther)
			return
		}

		title := r.FormValue("title")
		contactID := r.FormValue("contact_id")

		if title == "" {
			http.Redirect(w, r, "/threads?error=Title+is+required", http.StatusSeeOther)
			return
		}

		_, err := h.threadService.CreateThread(r.Context(), app.CreateThreadInput{
			Title:     title,
			ContactID: contactID,
		})
		if err != nil {
			http.Redirect(w, r, "/threads?error="+err.Error(), http.StatusSeeOther)
			return
		}

		http.Redirect(w, r, "/threads?success=Thread+created", http.StatusSeeOther)
	}
}

// HandleLinkRoleForm handles POST /threads/{id}/roles/link (HTML form)
func (h *Handlers) HandleLinkRoleForm() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		threadID := chi.URLParam(r, "id")

		if err := r.ParseForm(); err != nil {
			http.Redirect(w, r, "/threads/"+threadID+"?error=Invalid+form+data", http.StatusSeeOther)
			return
		}

		roleRef := r.FormValue("role_ref")
		if roleRef == "" {
			http.Redirect(w, r, "/threads/"+threadID+"?error=Role+is+required", http.StatusSeeOther)
			return
		}

		// Parse role_ref (format: "companySlug/roleSlug")
		parts := splitRoleRef(roleRef)
		if len(parts) != 2 {
			http.Redirect(w, r, "/threads/"+threadID+"?error=Invalid+role+reference", http.StatusSeeOther)
			return
		}

		err := h.threadService.LinkRole(r.Context(), app.LinkRoleInput{
			ThreadID:    threadID,
			CompanySlug: parts[0],
			RoleSlug:    parts[1],
		})
		if err != nil {
			http.Redirect(w, r, "/threads/"+threadID+"?error="+err.Error(), http.StatusSeeOther)
			return
		}

		http.Redirect(w, r, "/threads/"+threadID+"?success=Role+linked", http.StatusSeeOther)
	}
}

// HandleCreateMeetingFromThreadForm handles POST /threads/{id}/meetings/new (HTML form)
func (h *Handlers) HandleCreateMeetingFromThreadForm() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		threadID := chi.URLParam(r, "id")

		if err := r.ParseForm(); err != nil {
			http.Redirect(w, r, "/threads/"+threadID+"?error=Invalid+form+data", http.StatusSeeOther)
			return
		}

		title := r.FormValue("title")
		occurredAt := r.FormValue("occurred_at")
		companySlug := r.FormValue("company_slug")

		if title == "" || occurredAt == "" || companySlug == "" {
			http.Redirect(w, r, "/threads/"+threadID+"?error=Title,+date,+and+company+are+required", http.StatusSeeOther)
			return
		}

		// Convert datetime-local to RFC3339
		occurredAtRFC := occurredAt + ":00Z"

		_, err := h.meetingService.CreateMeeting(r.Context(), app.CreateMeetingInput{
			CompanySlug: companySlug,
			ThreadID:    threadID,
			OccurredAt:  occurredAtRFC,
			Title:       title,
		})
		if err != nil {
			http.Redirect(w, r, "/threads/"+threadID+"?error="+err.Error(), http.StatusSeeOther)
			return
		}

		http.Redirect(w, r, "/threads/"+threadID+"?success=Meeting+created", http.StatusSeeOther)
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
		var role interface{}
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

		// Check for success/error query params
		successMsg := r.URL.Query().Get("success")
		errorMsg := r.URL.Query().Get("error")

		// Build list of all statuses for dropdown with friendly labels
		allStatuses := []struct {
			Value string
			Label string
		}{
			{"recruiter_reached_out", "Recruiter Reached Out"},
			{"hr_interview", "HR Interview"},
			{"pairing_interview", "Pairing Interview"},
			{"take_home_assignment", "Take Home Assignment"},
			{"design_interview", "Design Interview"},
			{"in_progress", "In Progress"},
			{"offer", "Offer"},
			{"rejected", "Rejected"},
		}

		data := map[string]interface{}{
			"Title":       company.Company.Name + " - " + roleSlug,
			"Company":     company.Company,
			"Role":        role,
			"JD":          jdData,
			"AllStatuses": allStatuses,
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
