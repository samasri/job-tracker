package migrations

import "jobtracker/internal/infra/sqlite"

func init() {
	sqlite.RegisterMigration(sqlite.Migration{
		Version: 8,
		Name:    "role_resumes",
		Up: `
			CREATE TABLE role_resumes (
				role_id TEXT PRIMARY KEY REFERENCES roles(id),
				path_json TEXT,
				path_pdf TEXT,
				created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
				updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
			);
		`,
	})
}
