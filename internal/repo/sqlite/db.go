// Package sqlite implements the repository interfaces using SQLite.
//
// Driver choice: modernc.org/sqlite (pure Go).
// - No cgo / C compiler needed → trivial cross-compilation for ARM/Linux/etc.
// - Single binary distribution is the #1 priority for this project.
// - ~15% slower than mattn/go-sqlite3 on write-heavy benchmarks, which is
//   negligible for a personal file cloud with occasional uploads.
//
// Connection strategy: MaxOpenConns(1).
// SQLite only supports one writer at a time. A single connection eliminates
// "database is locked" errors and ensures PRAGMAs (foreign_keys=ON) persist
// for the lifetime of the process. WAL mode still allows the OS to serve
// concurrent reads at the filesystem level.
package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"log/slog"
	"sort"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

// DB wraps a SQLite connection and provides repository constructors.
type DB struct {
	sql    *sql.DB
	logger *slog.Logger
}

// Open creates a new SQLite database connection with production-safe defaults.
func Open(dbPath string, logger *slog.Logger) (*DB, error) {
	sqlDB, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("opening sqlite %s: %w", dbPath, err)
	}

	// Single connection — see package doc for rationale.
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetConnMaxLifetime(0)

	// WAL: better read concurrency.
	// foreign_keys: enforce referential integrity (off by default in SQLite!).
	// busy_timeout: retry on contention instead of failing immediately.
	pragmas := []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA foreign_keys=ON",
		"PRAGMA busy_timeout=5000",
	}
	for _, p := range pragmas {
		if _, err := sqlDB.Exec(p); err != nil {
			sqlDB.Close()
			return nil, fmt.Errorf("setting %s: %w", p, err)
		}
	}

	// Verify the connection is live.
	if err := sqlDB.Ping(); err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("pinging sqlite: %w", err)
	}

	return &DB{sql: sqlDB, logger: logger}, nil
}

// Close closes the underlying database connection.
func (d *DB) Close() error {
	return d.sql.Close()
}

// HealthCheck returns nil if the database is reachable, or an error with context.
func (d *DB) HealthCheck() error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	return d.sql.PingContext(ctx)
}

// Migrate runs all pending .sql migrations from the provided filesystem.
// Each migration is applied in its own transaction. Applied migrations are
// tracked in a _migrations table and never re-run.
//
// This is a lightweight alternative to goose/golang-migrate — fewer deps,
// no magic, ~60 lines. Sufficient for a project with a stable schema that
// evolves through numbered SQL files.
func (d *DB) Migrate(migrationsFS fs.FS) error {
	_, err := d.sql.Exec(`CREATE TABLE IF NOT EXISTS _migrations (
		id         INTEGER PRIMARY KEY AUTOINCREMENT,
		name       TEXT NOT NULL UNIQUE,
		applied_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
	)`)
	if err != nil {
		return fmt.Errorf("creating _migrations table: %w", err)
	}

	entries, err := fs.ReadDir(migrationsFS, ".")
	if err != nil {
		return fmt.Errorf("reading migrations: %w", err)
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}

		name := entry.Name()

		var count int
		if err := d.sql.QueryRow("SELECT COUNT(*) FROM _migrations WHERE name = ?", name).Scan(&count); err != nil {
			return fmt.Errorf("checking migration %s: %w", name, err)
		}
		if count > 0 {
			continue // already applied
		}

		data, err := fs.ReadFile(migrationsFS, name)
		if err != nil {
			return fmt.Errorf("reading migration %s: %w", name, err)
		}

		tx, err := d.sql.Begin()
		if err != nil {
			return fmt.Errorf("begin tx for %s: %w", name, err)
		}

		if _, err := tx.Exec(string(data)); err != nil {
			tx.Rollback()
			return fmt.Errorf("applying migration %s: %w", name, err)
		}

		if _, err := tx.Exec("INSERT INTO _migrations (name) VALUES (?)", name); err != nil {
			tx.Rollback()
			return fmt.Errorf("recording migration %s: %w", name, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("committing migration %s: %w", name, err)
		}

		d.logger.Info("applied migration", "name", name)
	}

	return nil
}

// UserRepo returns the SQLite-backed user repository.
func (d *DB) UserRepo() *UserRepo {
	return &UserRepo{db: d.sql}
}

// FileRepo returns the SQLite-backed file repository.
func (d *DB) FileRepo() *FileRepo {
	return &FileRepo{db: d.sql}
}

// DirectoryRepo returns the SQLite-backed directory repository.
func (d *DB) DirectoryRepo() *DirectoryRepo {
	return &DirectoryRepo{db: d.sql}
}

// SessionRepo returns the SQLite-backed session repository.
func (d *DB) SessionRepo() *SessionRepo {
	return &SessionRepo{db: d.sql}
}
