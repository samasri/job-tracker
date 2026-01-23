package migrations

import "jobtracker/internal/infra/sqlite"

func init() {
	sqlite.RegisterMigration(sqlite.Migration{
		Version: 1,
		Name:    "companies_and_roles",
		Up: `
			CREATE TABLE companies (
				id TEXT PRIMARY KEY,
				slug TEXT NOT NULL UNIQUE,
				name TEXT NOT NULL,
				folder_path TEXT NOT NULL,
				created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
				updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
			);

			CREATE INDEX idx_companies_slug ON companies(slug);

			CREATE TABLE roles (
				id TEXT PRIMARY KEY,
				company_id TEXT NOT NULL REFERENCES companies(id),
				slug TEXT NOT NULL,
				title TEXT NOT NULL,
				folder_path TEXT NOT NULL,
				created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
				updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
				UNIQUE(company_id, slug)
			);

			CREATE INDEX idx_roles_company_id ON roles(company_id);
			CREATE INDEX idx_roles_slug ON roles(company_id, slug);
		`,
	})
}
