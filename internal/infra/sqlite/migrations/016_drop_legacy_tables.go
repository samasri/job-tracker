package migrations

import "jobtracker/internal/infra/sqlite"

func init() {
	sqlite.RegisterMigration(sqlite.Migration{
		Version: 16,
		Name:    "drop_legacy_tables",
		Up: `
			DROP TABLE IF EXISTS meeting_threads;
			DROP TABLE IF EXISTS meetings;
			DROP TABLE IF EXISTS threads;
		`,
	})
}
