package migrations

import "jobtracker/internal/infra/sqlite"

func init() {
	sqlite.RegisterMigration(sqlite.Migration{
		Version: 6,
		Name:    "meetings_v2",
		Up: `
			CREATE TABLE meetings_v2 (
				id TEXT PRIMARY KEY,
				occurred_at TEXT NOT NULL,
				title TEXT NOT NULL,
				role_id TEXT REFERENCES roles(id),
				thread_id TEXT REFERENCES threads(id),
				path_md TEXT NOT NULL,
				created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
				updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
				CHECK(
					(role_id IS NOT NULL AND thread_id IS NULL) OR
					(role_id IS NULL AND thread_id IS NOT NULL)
				)
			);

			CREATE INDEX idx_meetings_v2_role_occurred ON meetings_v2(role_id, occurred_at)
				WHERE role_id IS NOT NULL;
			CREATE INDEX idx_meetings_v2_thread_occurred ON meetings_v2(thread_id, occurred_at)
				WHERE thread_id IS NOT NULL;
		`,
	})
}
