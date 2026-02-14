package http

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// Server holds HTTP server dependencies
type Server struct {
	router   *chi.Mux
	handlers *Handlers
}

// NewServer creates a new HTTP server
func NewServer(handlers *Handlers) *Server {
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)

	s := &Server{
		router:   r,
		handlers: handlers,
	}
	s.routes()

	return s
}

func (s *Server) routes() {
	// Health check
	s.router.Get("/health", s.handleHealth())

	// API routes
	s.router.Route("/api", func(r chi.Router) {
		// Companies
		r.Post("/companies", s.handlers.HandleCreateCompany())
		r.Get("/companies", s.handlers.HandleListCompanies())
		r.Get("/companies/{slug}", s.handlers.HandleGetCompany())
		r.Post("/companies/{slug}/roles", s.handlers.HandleCreateRole())
		r.Patch("/companies/{companySlug}/roles/{roleSlug}/status", s.handlers.HandleUpdateRoleStatus())

		// Contacts
		r.Post("/contacts", s.handlers.HandleCreateContact())

		// Threads
		r.Post("/threads", s.handlers.HandleCreateThread())
		r.Get("/threads/{id}", s.handlers.HandleGetThread())
		r.Post("/threads/{id}/roles", s.handlers.HandleLinkRole())

		// Meetings (legacy)
		r.Post("/meetings", s.handlers.HandleCreateMeeting())

		// Meetings V2 - role meetings
		r.Post("/companies/{companySlug}/roles/{roleSlug}/meetings", s.handlers.HandleCreateRoleMeetingV2())
		// Meetings V2 - thread-only meetings
		r.Post("/threads/{id}/meetings", s.handlers.HandleCreateThreadMeetingV2())

		// Job Descriptions
		r.Post("/roles/{companySlug}/{roleSlug}/jd", s.handlers.HandleAttachJD())

		// Export
		r.Post("/export", s.handlers.HandleExport())
		r.Get("/export", s.handlers.HandleExport())
	})

	// HTML pages
	s.router.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/companies", http.StatusFound)
	})
	s.router.Get("/companies", s.handlers.HandleCompaniesPage())
	s.router.Post("/companies/new", s.handlers.HandleCreateCompanyForm())
	s.router.Get("/companies/{slug}", s.handlers.HandleCompanyPage())
	s.router.Post("/companies/{slug}/roles/new", s.handlers.HandleCreateRoleForm())
	s.router.Post("/companies/{slug}/meetings/new", s.handlers.HandleCreateMeetingForm())
	s.router.Get("/companies/{companySlug}/roles/{roleSlug}", s.handlers.HandleRolePage())
	s.router.Post("/companies/{companySlug}/roles/{roleSlug}/status", s.handlers.HandleUpdateRoleStatusForm())
	s.router.Post("/companies/{companySlug}/roles/{roleSlug}/jd", s.handlers.HandleAttachJDForm())
	s.router.Get("/companies/{companySlug}/roles/{roleSlug}/jd", s.handlers.HandleViewJD())
	s.router.Get("/companies/{companySlug}/roles/{roleSlug}/jd/raw", s.handlers.HandleViewJDRaw())
	s.router.Post("/companies/{companySlug}/roles/{roleSlug}/resume", s.handlers.HandleAttachResumeForm())
	s.router.Post("/companies/{companySlug}/roles/{roleSlug}/artifacts", s.handlers.HandleUpsertArtifactForm())
	s.router.Get("/companies/{companySlug}/roles/{roleSlug}/artifacts/{name}", s.handlers.HandleViewArtifact())
	s.router.Post("/companies/{companySlug}/roles/{roleSlug}/artifacts/{name}/delete", s.handlers.HandleDeleteArtifact())
	s.router.Post("/companies/{companySlug}/roles/{roleSlug}/meetings/new", s.handlers.HandleCreateRoleMeetingV2Form())
	s.router.Get("/threads", s.handlers.HandleThreadsPage())
	s.router.Post("/threads/new", s.handlers.HandleCreateThreadForm())
	s.router.Get("/threads/{id}", s.handlers.HandleThreadPage())
	s.router.Post("/threads/{id}/roles/link", s.handlers.HandleLinkRoleForm())
	s.router.Post("/threads/{id}/meetings/new", s.handlers.HandleCreateMeetingFromThreadForm())
	s.router.Post("/threads/{id}/meetings/v2/new", s.handlers.HandleCreateThreadMeetingV2Form())
	s.router.Post("/contacts/new", s.handlers.HandleCreateContactForm())
	s.router.Post("/export", s.handlers.HandleExportPage())
}

func (s *Server) handleHealth() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}
}

// ServeHTTP implements http.Handler
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}

// Router returns the chi router for testing
func (s *Server) Router() *chi.Mux {
	return s.router
}
