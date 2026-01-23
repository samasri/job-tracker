package migrations

import "jobtracker/internal/infra/sqlite"

func init() {
	sqlite.RegisterMigration(sqlite.Migration{
		Version: 2,
		Name:    "contacts_threads_meetings",
		Up: `
			CREATE TABLE contacts (
				id TEXT PRIMARY KEY,
				name TEXT NOT NULL,
				org TEXT DEFAULT '',
				linkedin_url TEXT DEFAULT '',
				email TEXT DEFAULT '',
				created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
				updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
			);

			CREATE TABLE threads (
				id TEXT PRIMARY KEY,
				title TEXT NOT NULL,
				contact_id TEXT REFERENCES contacts(id),
				created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
				updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
			);

			CREATE INDEX idx_threads_contact_id ON threads(contact_id);

			CREATE TABLE meetings (
				id TEXT PRIMARY KEY,
				occurred_at TEXT NOT NULL,
				title TEXT NOT NULL,
				company_id TEXT NOT NULL REFERENCES companies(id),
				path_md TEXT NOT NULL,
				created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
				updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
			);

			CREATE INDEX idx_meetings_company_id ON meetings(company_id);
			CREATE INDEX idx_meetings_occurred_at ON meetings(occurred_at);

			CREATE TABLE meeting_threads (
				meeting_id TEXT NOT NULL REFERENCES meetings(id),
				thread_id TEXT NOT NULL REFERENCES threads(id),
				created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
				PRIMARY KEY (meeting_id, thread_id)
			);
		`,
	})
}
