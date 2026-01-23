package migrations

import "jobtracker/internal/infra/sqlite"

func init() {
	// Note: SQLite's ALTER TABLE doesn't support CHECK constraints.
	// Validation is enforced at the application layer in domain.ParseRoleStatus.
	sqlite.RegisterMigration(sqlite.Migration{
		Version: 5,
		Name:    "role_status",
		Up: `
			ALTER TABLE roles ADD COLUMN status TEXT NOT NULL DEFAULT 'recruiter_reached_out';
		`,
	})
}
