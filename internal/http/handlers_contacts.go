package http

import (
	"encoding/json"
	"net/http"
	"time"

	"jobtracker/internal/app"
	"jobtracker/internal/domain"

	"github.com/go-chi/chi/v5"
)

// createContactRequest is the request body for creating a contact
type createContactRequest struct {
	Name        string `json:"name"`
	Org         string `json:"org"`
	LinkedInURL string `json:"linkedin_url"`
	Email       string `json:"email"`
}

// HandleCreateContact handles POST /api/contacts
func (h *handlers) HandleCreateContact() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req createContactRequest
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
		json.NewEncoder(w).Encode(contactResponse{
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
func (h *handlers) HandleGetContactAPI() http.HandlerFunc {
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

		roles := make([]roleWithCompanyResponse, 0, len(details.Roles))
		for _, rc := range details.Roles {
			roles = append(roles, roleWithCompanyResponse{
				Role: roleResponse{
					ID:         rc.Role.ID,
					CompanyID:  rc.Role.CompanyID,
					Slug:       rc.Role.Slug,
					Title:      rc.Role.Title,
					FolderPath: rc.Role.FolderPath,
				},
				Company: companyResponse{
					ID:         rc.Company.ID,
					Slug:       rc.Company.Slug,
					Name:       rc.Company.Name,
					FolderPath: rc.Company.FolderPath,
				},
			})
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(contactResponse{
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

// linkContactRoleRequest is the request body for linking a role to a contact
type linkContactRoleRequest struct {
	RoleRef string `json:"role_ref"` // Format: "companySlug/roleSlug"
}

// HandleLinkRoleToContactAPI handles POST /api/contacts/{id}/roles
func (h *handlers) HandleLinkRoleToContactAPI() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		contactID := chi.URLParam(r, "id")

		var req linkContactRoleRequest
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

// createContactMeetingRequest is the request body for creating a contact meeting
type createContactMeetingRequest struct {
	OccurredAt string `json:"occurred_at"`
	Title      string `json:"title"`
}

// HandleCreateContactMeetingAPI handles POST /api/contacts/{id}/meetings
func (h *handlers) HandleCreateContactMeetingAPI() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		contactID := chi.URLParam(r, "id")

		var req createContactMeetingRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		if req.Title == "" || req.OccurredAt == "" {
			http.Error(w, "title and occurred_at are required", http.StatusBadRequest)
			return
		}

		meeting, err := h.meetingService.CreateContactMeeting(r.Context(), app.CreateContactMeetingInput{
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
		json.NewEncoder(w).Encode(meetingResponse{
			ID:         meeting.ID,
			OccurredAt: meeting.OccurredAt.Format(time.RFC3339),
			Title:      meeting.Title,
			PathMD:     meeting.PathMD,
		})
	}
}

// HandleContactsPage handles GET /contacts (HTML page)
func (h *handlers) HandleContactsPage() http.HandlerFunc {
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
func (h *handlers) HandleContactPage() http.HandlerFunc {
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

		contactMeetings, err := h.meetingService.ListMeetingsByContact(r.Context(), id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		type RoleMeetingsGroup struct {
			Role     *domain.Role
			Company  *domain.Company
			Meetings []*domain.Meeting
		}
		var roleMeetingsGroups []RoleMeetingsGroup
		for _, rc := range details.Roles {
			meetings, err := h.meetingService.ListMeetingsByRole(r.Context(), rc.Role.ID)
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

		allRoles, err := h.getAllRolesForDropdown(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

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
func (h *handlers) HandleLinkRoleToContactForm() http.HandlerFunc {
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
func (h *handlers) HandleCreateContactMeetingForm() http.HandlerFunc {
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

		_, err := h.meetingService.CreateContactMeeting(r.Context(), app.CreateContactMeetingInput{
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
func (h *handlers) HandleCreateContactForm() http.HandlerFunc {
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
