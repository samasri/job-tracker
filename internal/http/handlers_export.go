package http

import (
	"encoding/json"
	"net/http"
)

// HandleExport handles POST /api/export
func (h *handlers) HandleExport() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := h.exportService.Export(r.Context()); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "exported"})
	}
}

// HandleExportPage handles POST /export (HTML form, redirects with success message)
func (h *handlers) HandleExportPage() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := h.exportService.Export(r.Context()); err != nil {
			http.Redirect(w, r, "/companies?error=Export+failed:+"+err.Error(), http.StatusSeeOther)
			return
		}

		http.Redirect(w, r, "/companies?success=Export+completed+successfully", http.StatusSeeOther)
	}
}
