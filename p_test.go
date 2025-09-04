package main

import (
	"database/sql"
	"os"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func TestValidatePromptName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"empty name", "", true},
		{"valid name", "test", false},
		{"max length name", string(make([]byte, 255)), false},
		{"too long name", string(make([]byte, 256)), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePromptName(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("validatePromptName() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

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
		{"malformed tags", "a,,b,,,c", "a,b,c"},
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

func TestEditorCancellation(t *testing.T) {
	// Test that validation catches empty content
	err := validatePromptContent("")
	if err == nil {
		t.Error("Expected error for empty content, got nil")
	}

	// Test that whitespace-only content is properly validated
	err = validatePromptContent("   \n\t  ")
	if err == nil {
		t.Error("Expected error for whitespace-only content, got nil")
	}
}

func setupMockEditor(t *testing.T) {
	mockScript := `#!/bin/bash
echo "test content" > "$1"
`
	mockFile, err := os.CreateTemp("", "mock_editor_*.sh")
	if err != nil {
		t.Fatal(err)
	}
	defer mockFile.Close()

	if _, err := mockFile.WriteString(mockScript); err != nil {
		t.Fatal(err)
	}

	if err := os.Chmod(mockFile.Name(), 0755); err != nil {
		t.Fatal(err)
	}

	os.Setenv("EDITOR", mockFile.Name())
	t.Cleanup(func() {
		os.Remove(mockFile.Name())
		os.Unsetenv("EDITOR")
	})
}

func TestAddPromptIntegration(t *testing.T) {
	store := setupTestDB(t)
	app := NewApp(store)
	setupMockEditor(t)

	// Test adding a prompt using external editor
	err := app.AddPrompt("testprompt", "test,cli", true)
	if err != nil {
		t.Errorf("AddPrompt failed: %v", err)
	}

	// Verify prompt was added
	prompt, err := store.GetPromptByName("testprompt")
	if err != nil {
		t.Errorf("Failed to retrieve added prompt: %v", err)
	}
	if prompt.Name != "testprompt" {
		t.Errorf("Expected prompt name 'testprompt', got '%s'", prompt.Name)
	}
}

func TestListPromptsWithTags(t *testing.T) {
	store := setupTestDB(t)
	app := NewApp(store)

	// Add test prompts
	err := store.AddPrompt("test1", "content1", "tag1,tag2")
	if err != nil {
		t.Fatal(err)
	}
	err = store.AddPrompt("test2", "content2", "tag2,tag3")
	if err != nil {
		t.Fatal(err)
	}

	// Test filtering by tags
	prompts, err := app.ListPrompts("tag2")
	if err != nil {
		t.Errorf("ListPrompts with tags failed: %v", err)
	}
	if len(prompts) != 2 {
		t.Errorf("Expected 2 prompts with tag2, got %d", len(prompts))
	}
}

func TestDeletePromptIntegration(t *testing.T) {
	store := setupTestDB(t)
	app := NewApp(store)

	// Add a test prompt
	err := store.AddPrompt("testdelete", "test content", "test")
	if err != nil {
		t.Fatal(err)
	}

	// Delete the prompt
	err = app.DeletePrompt("testdelete")
	if err != nil {
		t.Errorf("DeletePrompt failed: %v", err)
	}

	// Verify prompt was deleted
	_, err = store.GetPromptByName("testdelete")
	if err == nil {
		t.Error("Expected error when retrieving deleted prompt, got nil")
	}
}
