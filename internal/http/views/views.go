package views

import (
	"embed"
	"html/template"
	"io"
	"strings"
)

//go:embed *.html
var templateFS embed.FS

// Views holds parsed templates
type Views struct {
	templates map[string]*template.Template
}

// New creates a new Views instance with parsed templates
func New() (*Views, error) {
	funcMap := template.FuncMap{
		"statusClass": statusClass,
	}

	templates := make(map[string]*template.Template)

	// Parse each page template with the layout
	pages := []string{"companies", "company", "thread", "threads", "role", "jd_viewer", "contacts", "contact"}
	for _, page := range pages {
		tmpl, err := template.New("").Funcs(funcMap).ParseFS(templateFS, "layout.html", page+".html")
		if err != nil {
			return nil, err
		}
		templates[page] = tmpl
	}

	return &Views{templates: templates}, nil
}

// Render renders a template with the given data
func (v *Views) Render(w io.Writer, name string, data interface{}) error {
	tmpl, ok := v.templates[name]
	if !ok {
		return nil
	}
	return tmpl.ExecuteTemplate(w, "layout", data)
}

// statusClass returns a CSS class for a status
func statusClass(status string) string {
	lower := strings.ToLower(status)
	switch lower {
	case "offer":
		return "offer"
	case "rejected":
		return "rejected"
	case "cancelled":
		return "cancelled"
	case "in_progress":
		return "in-progress"
	case "hr_interview", "pairing_interview", "design_interview", "take_home_assignment":
		return "active"
	case "recruiter_reached_out":
		return "pending"
	default:
		// Legacy status handling
		switch {
		case strings.Contains(lower, "active") || strings.Contains(lower, "interviewing"):
			return "active"
		case strings.Contains(lower, "pending") || strings.Contains(lower, "applied"):
			return "pending"
		case strings.Contains(lower, "reject") || strings.Contains(lower, "declined"):
			return "rejected"
		default:
			return "default"
		}
	}
}
