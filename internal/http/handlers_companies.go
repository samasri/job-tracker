package http

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"time"

	"jobtracker/internal/app"

	"github.com/go-chi/chi/v5"
)

// CompanyListItem represents a company for the list view
type CompanyListItem struct {
	Company   interface{}
	Status    string
	LastTouch string
}

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

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(CompanyWithDetailsResponse{
			Company: CompanyResponse{
				ID:         company.Company.ID,
				Slug:       company.Company.Slug,
				Name:       company.Company.Name,
				FolderPath: company.Company.FolderPath,
				Status:     company.Status.String(),
			},
			Roles: roles,
		})
	}
}

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

// HandleCompaniesPage handles GET /companies (HTML page)
func (h *Handlers) HandleCompaniesPage() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		items, err := h.buildCompanyListItems(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

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

		successMsg := r.URL.Query().Get("success")
		errorMsg := r.URL.Query().Get("error")

		data := map[string]interface{}{
			"Title":   company.Company.Name,
			"Company": company.Company,
			"Roles":   company.Roles,
			"Status":  company.Status.String(),
			"Success": successMsg,
			"Error":   errorMsg,
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := h.views.Render(w, "company", data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

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

func (h *Handlers) buildCompanyListItems(r *http.Request) ([]CompanyListItem, error) {
	companies, err := h.companyService.ListCompanies(r.Context())
	if err != nil {
		return nil, err
	}

	items := make([]CompanyListItem, 0, len(companies))
	for _, c := range companies {
		var lastTouch string
		var latest time.Time
		for _, role := range c.Roles {
			meetings, err := h.meetingService.ListMeetingsByRole(r.Context(), role.ID)
			if err != nil {
				return nil, fmt.Errorf("listing meetings for role %s: %w", role.ID, err)
			}
			for _, m := range meetings {
				if m.OccurredAt.After(latest) {
					latest = m.OccurredAt
				}
			}
		}
		if !latest.IsZero() {
			lastTouch = latest.Format("2006-01-02")
		}

		items = append(items, CompanyListItem{
			Company:   c.Company,
			Status:    c.Status.String(),
			LastTouch: lastTouch,
		})
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].LastTouch == "" {
			return false
		}
		if items[j].LastTouch == "" {
			return true
		}
		return items[i].LastTouch > items[j].LastTouch
	})

	return items, nil
}

func (h *Handlers) renderCompaniesPageWithError(w http.ResponseWriter, r *http.Request, errMsg, formSlug, formName string) {
	items, err := h.buildCompanyListItems(r)
	if err != nil {
		http.Error(w, fmt.Sprintf("loading companies: %s; original error: %s", err.Error(), errMsg), http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Title":     "Companies",
		"Companies": items,
		"Error":     errMsg,
		"FormSlug":  formSlug,
		"FormName":  formName,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.views.Render(w, "companies", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
