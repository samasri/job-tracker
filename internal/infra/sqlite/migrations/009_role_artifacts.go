package migrations

import "jobtracker/internal/infra/sqlite"

func init() {
	sqlite.RegisterMigration(sqlite.Migration{
		Version: 9,
		Name:    "role_artifacts",
		Up: `
			CREATE TABLE role_artifacts (
				id TEXT PRIMARY KEY,
				role_id TEXT NOT NULL REFERENCES roles(id),
				name TEXT NOT NULL,
				type TEXT NOT NULL CHECK(type IN ('pdf', 'jsonc', 'text')),
				path TEXT NOT NULL,
				created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
				updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
				UNIQUE(role_id, name)
			);

			CREATE INDEX idx_role_artifacts_role_id ON role_artifacts(role_id);
		`,
	})
}
