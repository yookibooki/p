package main

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

const (
	appName    = "p"
	dbFileName = "prompts.db"
)

func InitDB() (*sql.DB, error) {
	db, _, err := InitDBWithPath()
	return db, err
}

func InitDBWithPath() (*sql.DB, string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return nil, "", fmt.Errorf("error finding user config directory: %w", err)
	}

	appConfigDir := filepath.Join(configDir, appName)
	// Uses 0700 permissions for the config directory as prompts may contain sensitive data.
	if err := os.MkdirAll(appConfigDir, 0o700); err != nil {
		return nil, "", fmt.Errorf("error creating application config directory: %w", err)
	}

	dbPath := filepath.Join(appConfigDir, dbFileName)

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, "", fmt.Errorf("error opening database: %w", err)
	}

	// Run database migrations
	if err := runMigrations(db); err != nil {
		db.Close()
		return nil, "", fmt.Errorf("error running migrations: %w", err)
	}

	return db, dbPath, nil
}

// runMigrations applies database schema migrations
func runMigrations(db *sql.DB) error {
	// Create migrations table if it doesn't exist
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
	`)
	if err != nil {
		return fmt.Errorf("error creating migrations table: %w", err)
	}

	// Migration 1: Create prompts table
	if err := applyMigration(db, 1, `
		CREATE TABLE IF NOT EXISTS prompts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL UNIQUE,
			prompt TEXT NOT NULL,
			tags TEXT
		);
	`); err != nil {
		return fmt.Errorf("error applying migration 1: %w", err)
	}

	// Future migrations can be added here
	// Migration 2: Add created_at and updated_at columns
	// if err := applyMigration(db, 2, `
	//     ALTER TABLE prompts ADD COLUMN created_at DATETIME DEFAULT CURRENT_TIMESTAMP;
	//     ALTER TABLE prompts ADD COLUMN updated_at DATETIME DEFAULT CURRENT_TIMESTAMP;
	// `); err != nil {
	//     return fmt.Errorf("error applying migration 2: %w", err)
	// }

	return nil
}

// applyMigration applies a single migration if it hasn't been applied yet
func applyMigration(db *sql.DB, version int, sql string) error {
	// Check if migration has already been applied
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM schema_migrations WHERE version = ?", version).Scan(&count)
	if err != nil {
		return fmt.Errorf("error checking migration status: %w", err)
	}

	if count > 0 {
		// Migration already applied
		return nil
	}

	// Apply the migration
	_, err = db.Exec(sql)
	if err != nil {
		return fmt.Errorf("error executing migration SQL: %w", err)
	}

	// Record the migration as applied
	_, err = db.Exec("INSERT INTO schema_migrations (version) VALUES (?)", version)
	if err != nil {
		return fmt.Errorf("error recording migration: %w", err)
	}

	return nil
}
