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
	configDir, err := os.UserConfigDir()
	if err != nil {
		return nil, fmt.Errorf("error finding user config directory: %w", err)
	}

	appConfigDir := filepath.Join(configDir, appName)
	// Uses 0700 permissions for the config directory as prompts may contain sensitive data.
	if err := os.MkdirAll(appConfigDir, 0700); err != nil {
		return nil, fmt.Errorf("error creating application config directory: %w", err)
	}

	dbPath := filepath.Join(appConfigDir, dbFileName)

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("error opening database: %w", err)
	}

	sqlStmt := `
		CREATE TABLE IF NOT EXISTS prompts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL UNIQUE,
			prompt TEXT NOT NULL,
			tags TEXT
		);
	`

	_, err = db.Exec(sqlStmt)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("error creating table: %w", err)
	}

	return db, nil
}
