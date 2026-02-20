package migrations

import "jobtracker/internal/infra/sqlite"

func init() {
	sqlite.RegisterMigration(sqlite.Migration{
		Version: 13,
		Name:    "artifact_file_type",
		Up: `
			CREATE TABLE role_artifacts_new (
				id TEXT PRIMARY KEY,
				role_id TEXT NOT NULL REFERENCES roles(id),
				name TEXT NOT NULL,
				type TEXT NOT NULL CHECK(type IN ('pdf', 'jsonc', 'text', 'html', 'markdown', 'png', 'file')),
				path TEXT NOT NULL,
				created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
				updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
				UNIQUE(role_id, name)
			);

			INSERT INTO role_artifacts_new SELECT * FROM role_artifacts;
			DROP TABLE role_artifacts;
			ALTER TABLE role_artifacts_new RENAME TO role_artifacts;

			CREATE INDEX idx_role_artifacts_role_id ON role_artifacts(role_id);
		`,
	})
}
