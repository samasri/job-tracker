package sqlite

import (
	"database/sql"
	"fmt"
	"sort"

	_ "github.com/mattn/go-sqlite3"
)

// DB wraps the sql.DB connection
type DB struct {
	*sql.DB
}

// New creates a new SQLite database connection
func New(dbPath string) (*DB, error) {
	db, err := sql.Open("sqlite3", dbPath+"?_foreign_keys=on")
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("pinging database: %w", err)
	}

	return &DB{db}, nil
}

// Migration represents a database migration
type Migration struct {
	Version int
	Name    string
	Up      string
}

// migrations holds all registered migrations
var migrations []Migration

// RegisterMigration adds a migration to the registry
func RegisterMigration(m Migration) {
	migrations = append(migrations, m)
}

// RunMigrations executes all pending migrations
func (db *DB) RunMigrations() error {
	// Create migrations table if not exists
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return fmt.Errorf("creating migrations table: %w", err)
	}

	// Sort migrations by version
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})

	// Run pending migrations
	for _, m := range migrations {
		var exists int
		err := db.QueryRow("SELECT 1 FROM schema_migrations WHERE version = ?", m.Version).Scan(&exists)
		if err == nil {
			continue // Migration already applied
		}
		if err != sql.ErrNoRows {
			return fmt.Errorf("checking migration %d: %w", m.Version, err)
		}

		// Apply migration
		if _, err := db.Exec(m.Up); err != nil {
			return fmt.Errorf("applying migration %d (%s): %w", m.Version, m.Name, err)
		}

		// Record migration
		if _, err := db.Exec("INSERT INTO schema_migrations (version, name) VALUES (?, ?)", m.Version, m.Name); err != nil {
			return fmt.Errorf("recording migration %d: %w", m.Version, err)
		}
	}

	return nil
}
