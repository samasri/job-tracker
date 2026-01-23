package migrations

import "jobtracker/internal/infra/sqlite"

func init() {
	sqlite.RegisterMigration(sqlite.Migration{
		Version: 4,
		Name:    "job_descriptions",
		Up: `
			CREATE TABLE role_job_descriptions (
				role_id TEXT PRIMARY KEY REFERENCES roles(id),
				path_html TEXT,
				path_pdf TEXT,
				created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
				updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
			);
		`,
	})
}
