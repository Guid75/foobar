package db

import (
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"sort"
	"strconv"
	"strings"

	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

func Open(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}

	if err := runMigrations(db); err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}

func runMigrations(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY
		)
	`)
	if err != nil {
		return err
	}

	entries, err := fs.ReadDir(migrationsFS, "migrations")
	if err != nil {
		return err
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		version, err := migrationVersion(entry.Name())
		if err != nil {
			return err
		}

		applied, err := migrationApplied(db, version)
		if err != nil {
			return err
		}

		if applied {
			continue
		}

		if err := runMigration(db, entry.Name(), version); err != nil {
			return err
		}
	}

	return nil
}

func migrationVersion(filename string) (int, error) {
	parts := strings.SplitN(filename, "_", 2)
	if len(parts) != 2 {
		return 0, fmt.Errorf("invalid migration filename: %s", filename)
	}

	version, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, fmt.Errorf("invalid migration version: %s", filename)
	}

	return version, nil
}

func migrationApplied(db *sql.DB, version int) (bool, error) {
	var count int

	err := db.QueryRow(`
		SELECT COUNT(*)
		FROM schema_migrations
		WHERE version = ?
	`, version).Scan(&count)

	return count > 0, err
}

func runMigration(db *sql.DB, filename string, version int) error {
	content, err := migrationsFS.ReadFile("migrations/" + filename)
	if err != nil {
		return err
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(string(content)); err != nil {
		return fmt.Errorf("migration %s failed: %w", filename, err)
	}

	if _, err := tx.Exec(`
		INSERT INTO schema_migrations (version)
		VALUES (?)
	`, version); err != nil {
		return err
	}

	return tx.Commit()
}
