package migrations

import "jobtracker/internal/infra/sqlite"

func init() {
	sqlite.RegisterMigration(sqlite.Migration{
		Version: 14,
		Name:    "contact_infrastructure",
		Up: `
			-- Recreate meetings_v2 with relaxed XOR: thread_id rows may also have contact_id during backfill
			CREATE TABLE meetings_v2_new (
				id TEXT PRIMARY KEY,
				occurred_at TEXT NOT NULL,
				title TEXT NOT NULL,
				role_id TEXT REFERENCES roles(id),
				thread_id TEXT REFERENCES threads(id),
				contact_id TEXT REFERENCES contacts(id),
				path_md TEXT NOT NULL,
				created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
				updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
				CHECK(
					(role_id IS NOT NULL AND thread_id IS NULL    AND contact_id IS NULL) OR
					(role_id IS NULL    AND thread_id IS NOT NULL)                        OR
					(role_id IS NULL    AND thread_id IS NULL     AND contact_id IS NOT NULL)
				)
			);
			INSERT INTO meetings_v2_new
				SELECT id, occurred_at, title, role_id, thread_id, NULL AS contact_id, path_md, created_at, updated_at
				FROM meetings_v2;
			DROP TABLE meetings_v2;
			ALTER TABLE meetings_v2_new RENAME TO meetings_v2;
			CREATE INDEX idx_meetings_v2_role_occurred    ON meetings_v2(role_id,    occurred_at) WHERE role_id    IS NOT NULL;
			CREATE INDEX idx_meetings_v2_thread_occurred  ON meetings_v2(thread_id,  occurred_at) WHERE thread_id  IS NOT NULL;
			CREATE INDEX idx_meetings_v2_contact_occurred ON meetings_v2(contact_id, occurred_at) WHERE contact_id IS NOT NULL;

			-- Add code/slug/folder_path to contacts
			ALTER TABLE contacts ADD COLUMN code TEXT;
			ALTER TABLE contacts ADD COLUMN slug TEXT;
			ALTER TABLE contacts ADD COLUMN folder_path TEXT;
			CREATE UNIQUE INDEX idx_contacts_code ON contacts(code) WHERE code IS NOT NULL;
			CREATE UNIQUE INDEX idx_contacts_slug ON contacts(slug) WHERE slug IS NOT NULL;

			-- New contact_roles join table
			CREATE TABLE contact_roles (
				contact_id TEXT NOT NULL REFERENCES contacts(id),
				role_id    TEXT NOT NULL REFERENCES roles(id),
				created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
				PRIMARY KEY (contact_id, role_id)
			);
			CREATE INDEX idx_contact_roles_contact_id ON contact_roles(contact_id);
			CREATE INDEX idx_contact_roles_role_id    ON contact_roles(role_id);
		`,
	})
}
