// Package sqlite provides SQLite-backed repository implementations.
// It uses modernc.org/sqlite (pure Go, no CGO) for cross-platform builds.
package sqlite

import (
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"

	_ "modernc.org/sqlite" // register "sqlite" driver
)

//go:embed migrations/*.sql
var sqlMigrations embed.FS

// Open opens (or creates) the SQLite database at path, runs pending
// migrations, and configures sensible pragmas. Returns a ready *sql.DB.
func Open(path string) (*sql.DB, error) {
	// Ensure parent directory exists.
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("sqlite: create data dir: %w", err)
	}

	dsn := fmt.Sprintf("file:%s?_journal=WAL&_timeout=5000&_fk=true", path)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("sqlite: open: %w", err)
	}

	// Single writer recommended for SQLite; readers can pool.
	db.SetMaxOpenConns(1)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("sqlite: ping: %w", err)
	}

	if err := runMigrations(db); err != nil {
		return nil, fmt.Errorf("sqlite: migrate: %w", err)
	}

	return db, nil
}

// runMigrations applies all *.sql migration files in order, skipping ones
// already recorded in the schema_migrations table.
func runMigrations(db *sql.DB) error {
	// Ensure the migrations tracking table exists.
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
		version    TEXT PRIMARY KEY,
		applied_at TEXT NOT NULL
	)`)
	if err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	entries, err := fs.ReadDir(sqlMigrations, "migrations")
	if err != nil {
		return fmt.Errorf("read migrations dir: %w", err)
	}

	files := make([]string, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			files = append(files, e.Name())
		}
	}
	sort.Strings(files)

	for _, name := range files {
		version := strings.TrimSuffix(name, ".sql")

		var applied string
		row := db.QueryRow(`SELECT version FROM schema_migrations WHERE version = ?`, version)
		if scanErr := row.Scan(&applied); scanErr == nil {
			continue // already applied
		}

		data, readErr := sqlMigrations.ReadFile("migrations/" + name)
		if readErr != nil {
			return fmt.Errorf("read migration %s: %w", name, readErr)
		}

		if _, execErr := db.Exec(string(data)); execErr != nil {
			return fmt.Errorf("execute migration %s: %w", name, execErr)
		}

		if _, insErr := db.Exec(
			`INSERT INTO schema_migrations (version, applied_at) VALUES (?, datetime('now'))`,
			version,
		); insErr != nil {
			return fmt.Errorf("record migration %s: %w", name, insErr)
		}

		slog.Info("migration applied", "version", version)
	}

	return nil
}
