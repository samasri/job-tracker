package migrations

import "jobtracker/internal/infra/sqlite"

func init() {
	sqlite.RegisterMigration(sqlite.Migration{
		Version: 3,
		Name:    "thread_roles",
		Up: `
			CREATE TABLE thread_roles (
				thread_id TEXT NOT NULL REFERENCES threads(id),
				role_id TEXT NOT NULL REFERENCES roles(id),
				created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
				PRIMARY KEY (thread_id, role_id)
			);

			CREATE INDEX idx_thread_roles_thread_id ON thread_roles(thread_id);
			CREATE INDEX idx_thread_roles_role_id ON thread_roles(role_id);
		`,
	})
}
