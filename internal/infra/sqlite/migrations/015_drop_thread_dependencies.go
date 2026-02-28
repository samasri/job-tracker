package migrations

import "jobtracker/internal/infra/sqlite"

func init() {
	sqlite.RegisterMigration(sqlite.Migration{
		Version: 15,
		Name:    "drop_thread_dependencies",
		Up: `
			-- Recreate meetings_v2 without thread_id: strict XOR role_id OR contact_id.
			-- Rows that were thread-only (no contact_id) are dropped; the backfill that ran
			-- at startup in the previous version should have already set contact_id on them.
			CREATE TABLE meetings_v2_new (
				id TEXT PRIMARY KEY,
				occurred_at TEXT NOT NULL,
				title TEXT NOT NULL,
				role_id    TEXT REFERENCES roles(id),
				contact_id TEXT REFERENCES contacts(id),
				path_md TEXT NOT NULL,
				created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
				updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
				CHECK(
					(role_id IS NOT NULL AND contact_id IS NULL) OR
					(role_id IS NULL    AND contact_id IS NOT NULL)
				)
			);
			INSERT INTO meetings_v2_new
				SELECT id, occurred_at, title, role_id, contact_id, path_md, created_at, updated_at
				FROM meetings_v2
				WHERE role_id IS NOT NULL OR contact_id IS NOT NULL;
			DROP TABLE meetings_v2;
			ALTER TABLE meetings_v2_new RENAME TO meetings_v2;
			CREATE INDEX idx_meetings_v2_role_occurred    ON meetings_v2(role_id,    occurred_at) WHERE role_id    IS NOT NULL;
			CREATE INDEX idx_meetings_v2_contact_occurred ON meetings_v2(contact_id, occurred_at) WHERE contact_id IS NOT NULL;

			-- Drop thread_roles: relationship data has been migrated to contact_roles
			DROP TABLE thread_roles;
		`,
	})
}
