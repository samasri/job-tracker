package migrations

import "jobtracker/internal/infra/sqlite"

func init() {
	sqlite.RegisterMigration(sqlite.Migration{
		Version: 7,
		Name:    "thread_code_slug",
		Up: `
			-- Add code, slug, folder_path columns to threads
			ALTER TABLE threads ADD COLUMN code TEXT;
			ALTER TABLE threads ADD COLUMN slug TEXT;
			ALTER TABLE threads ADD COLUMN folder_path TEXT;

			-- Create unique indexes (will be enforced after backfill)
			CREATE UNIQUE INDEX idx_threads_code ON threads(code) WHERE code IS NOT NULL;
			CREATE UNIQUE INDEX idx_threads_slug ON threads(slug) WHERE slug IS NOT NULL;
		`,
	})
}
