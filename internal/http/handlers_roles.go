package http

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"time"

	"jobtracker/internal/app"
	"jobtracker/internal/domain"

	"github.com/go-chi/chi/v5"
	"github.com/yuin/goldmark"
)

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

		if err := r.ParseMultipartForm(10 << 20); err != nil {
			http.Error(w, "failed to parse multipart form: "+err.Error(), http.StatusBadRequest)
			return
		}

		htmlContent := r.FormValue("html")

		var pdfReader *multipartFileReader
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

// CreateRoleMeetingRequest is the request body for creating a role meeting (v2)
type CreateRoleMeetingRequest struct {
	OccurredAt string `json:"occurred_at"`
	Title      string `json:"title"`
}

// HandleCreateRoleMeeting handles POST /api/companies/{companySlug}/roles/{roleSlug}/meetings (JSON)
func (h *Handlers) HandleCreateRoleMeeting() http.HandlerFunc {
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

		meeting, err := h.meetingService.CreateRoleMeeting(r.Context(), app.CreateRoleMeetingInput{
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
		json.NewEncoder(w).Encode(MeetingResponse{
			ID:         meeting.ID,
			OccurredAt: meeting.OccurredAt.Format(time.RFC3339),
			Title:      meeting.Title,
			RoleID:     meeting.RoleID,
			PathMD:     meeting.PathMD,
		})
	}
}

// HandleRolePage handles GET /companies/{companySlug}/roles/{roleSlug} (HTML page)
func (h *Handlers) HandleRolePage() http.HandlerFunc {
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

		meetings, err := h.meetingService.ListMeetingsByRole(r.Context(), role.ID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		jd, err := h.jdService.GetJD(r.Context(), companySlug, roleSlug)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		jdData := struct {
			PathHTML string
			PathPDF  string
		}{}
		if jd != nil {
			jdData.PathHTML = jd.PathHTML
			jdData.PathPDF = jd.PathPDF
		}

		resume, err := h.resumeService.GetResume(r.Context(), companySlug, roleSlug)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		resumeData := struct {
			PathJSON string
			PathPDF  string
		}{}
		if resume != nil {
			resumeData.PathJSON = resume.PathJSON
			resumeData.PathPDF = resume.PathPDF
		}

		artifacts, err := h.artifactService.ListArtifacts(r.Context(), companySlug, roleSlug)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

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

		if err := r.ParseMultipartForm(10 << 20); err != nil {
			http.Redirect(w, r, redirectURL+"?error=Failed+to+parse+form", http.StatusSeeOther)
			return
		}

		htmlContent := r.FormValue("html")

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

		if err := r.ParseMultipartForm(20 << 20); err != nil {
			http.Redirect(w, r, redirectURL+"?error=Failed+to+parse+form", http.StatusSeeOther)
			return
		}

		jsonContent := r.FormValue("resume_json")

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

		jd, err := h.jdService.GetJobDescription(r.Context(), role.ID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if jd == nil || jd.PathHTML == "" {
			http.Error(w, "no job description HTML attached to this role", http.StatusNotFound)
			return
		}

		htmlContent, err := h.jdService.ReadJobDescriptionHTML(r.Context(), jd.PathHTML)
		if err != nil {
			http.Error(w, "failed to read job description: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		// Strict CSP that blocks scripts and external resources
		w.Header().Set("Content-Security-Policy", "default-src 'none'; img-src 'self' data:; style-src 'self' 'unsafe-inline'; font-src 'self' data:; frame-ancestors 'self'; base-uri 'none'; form-action 'none'")

		w.Write([]byte(htmlContent))
	}
}

// HandleCreateRoleMeetingForm handles POST /companies/{companySlug}/roles/{roleSlug}/meetings/new (HTML form)
func (h *Handlers) HandleCreateRoleMeetingForm() http.HandlerFunc {
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

		_, err := h.meetingService.CreateRoleMeeting(r.Context(), app.CreateRoleMeetingInput{
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

// HandleUpsertArtifactForm handles POST /companies/{companySlug}/roles/{roleSlug}/artifacts (HTML form)
func (h *Handlers) HandleUpsertArtifactForm() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		companySlug := chi.URLParam(r, "companySlug")
		roleSlug := chi.URLParam(r, "roleSlug")
		redirectURL := "/companies/" + companySlug + "/roles/" + roleSlug

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

		artifact, err := h.artifactService.GetArtifactByName(r.Context(), companySlug, roleSlug, name)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if artifact == nil {
			http.Error(w, "artifact not found", http.StatusNotFound)
			return
		}

		content, err := h.artifactService.ReadArtifactContent(r.Context(), artifact.Path)
		if err != nil {
			http.Error(w, "failed to read artifact: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("X-Content-Type-Options", "nosniff")

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
			var buf bytes.Buffer
			if err := goldmark.Convert(content, &buf); err != nil {
				http.Error(w, "failed to render markdown: "+err.Error(), http.StatusInternalServerError)
				return
			}
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
