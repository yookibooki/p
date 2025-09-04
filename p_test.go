package main

import (
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
	"os"
	"testing"
)

func TestValidatePromptContent(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantErr bool
	}{
		{"empty content", "", true},
		{"whitespace only", "   \n\t  ", true},
		{"valid content", "test prompt", false},
		{"max length content", string(make([]byte, 10000)), false},
		{"too long content", string(make([]byte, 10001)), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePromptContent(tt.content)
			if (err != nil) != tt.wantErr {
				t.Errorf("validatePromptContent() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNormalizeTags(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"empty tags", "", ""},
		{"single tag", "test", "test"},
		{"multiple tags", "b,a,c", "a,b,c"},
		{"duplicate tags", "test,test,other", "other,test"},
		{"whitespace tags", " a , b , c ", "a,b,c"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeTags(tt.input)
			if result != tt.expected {
				t.Errorf("normalizeTags() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func setupTestDB(t *testing.T) *SQLitePromptStore {
	tmpFile, err := os.CreateTemp("", "test_*.db")
	if err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()
	t.Cleanup(func() { os.Remove(tmpFile.Name()) })

	db, err := sql.Open("sqlite3", tmpFile.Name())
	if err != nil {
		t.Fatal(err)
	}

	sqlStmt := `CREATE TABLE IF NOT EXISTS prompts (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL UNIQUE,
		prompt TEXT NOT NULL,
		tags TEXT
	);`

	if _, err = db.Exec(sqlStmt); err != nil {
		t.Fatal(err)
	}

	store := NewSQLitePromptStore(db)
	t.Cleanup(func() { db.Close() })
	return store
}

func TestAppWithEmptyDB(t *testing.T) {
	store := setupTestDB(t)
	app := NewApp(store)

	// Test listing empty DB
	prompts, err := app.ListPrompts("")
	if err != nil {
		t.Errorf("ListPrompts() on empty DB failed: %v", err)
	}
	if len(prompts) != 0 {
		t.Errorf("Expected 0 prompts, got %d", len(prompts))
	}
}

func TestDuplicateNames(t *testing.T) {
	store := setupTestDB(t)

	// Add first prompt
	err := store.AddPrompt("test", "content1", "tag1")
	if err != nil {
		t.Fatal(err)
	}

	// Try to add duplicate name - should fail due to UNIQUE constraint
	err = store.AddPrompt("test", "content2", "tag2")
	if err == nil {
		t.Error("Expected error for duplicate name, got nil")
	}
}
