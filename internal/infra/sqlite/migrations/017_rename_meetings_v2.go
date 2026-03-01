package migrations

import "jobtracker/internal/infra/sqlite"

func init() {
	sqlite.RegisterMigration(sqlite.Migration{
		Version: 17,
		Name:    "rename_meetings_v2",
		Up:      `ALTER TABLE meetings_v2 RENAME TO meetings;`,
	})
}
