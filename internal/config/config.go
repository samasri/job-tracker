package config

import (
	"os"
	"path/filepath"
)

// Config holds application configuration
type Config struct {
	// RepoRoot is the root directory for the data repository
	RepoRoot string
	// DBPath is the path to the SQLite database
	DBPath string
	// Addr is the server bind address
	Addr string
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	cwd, _ := os.Getwd()
	return &Config{
		RepoRoot: cwd,
		DBPath:   filepath.Join(cwd, "db", "index.sqlite"),
		Addr:     "127.0.0.1:8080",
	}
}

// LoadConfig loads configuration from environment variables
func LoadConfig() *Config {
	cfg := DefaultConfig()

	if v := os.Getenv("JOBTRACKER_REPO_ROOT"); v != "" {
		cfg.RepoRoot = v
	}

	if v := os.Getenv("JOBTRACKER_DB_PATH"); v != "" {
		cfg.DBPath = v
	}

	if v := os.Getenv("JOBTRACKER_ADDR"); v != "" {
		cfg.Addr = v
	}

	return cfg
}
