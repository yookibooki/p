package main

import (
	"database/sql"
	"fmt"
	"strings"
)

type Prompt struct {
	ID     int
	Name   string
	Prompt string
	Tags   string
}

// SQLitePromptStore manages prompts using SQLite database.
type SQLitePromptStore struct {
	db *sql.DB
}

func NewSQLitePromptStore(db *sql.DB) *SQLitePromptStore {
	return &SQLitePromptStore{db: db}
}

// AddPrompt inserts a new prompt into the database.
func (s *SQLitePromptStore) AddPrompt(name, prompt, tags string) error {
	query := "INSERT INTO prompts (name, prompt, tags) VALUES (?, ?, ?)"
	_, err := s.db.Exec(query, name, prompt, tags)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint") {
			return fmt.Errorf("prompt name '%s' already exists", name)
		}
		return fmt.Errorf("error adding prompt: %w", err)
	}
	return nil
}

// GetPromptByName retrieves a prompt by its name from the database.
func (s *SQLitePromptStore) GetPromptByName(name string) (*Prompt, error) {
	query := "SELECT id, name, prompt, tags FROM prompts WHERE name = ?"
	row := s.db.QueryRow(query, name)
	var p Prompt
	if err := row.Scan(&p.ID, &p.Name, &p.Prompt, &p.Tags); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("prompt '%s' not found", name)
		}
		return nil, fmt.Errorf("error scanning prompt: %w", err)
	}
	return &p, nil
}

// UpdatePrompt modifies an existing prompt's content and tags in the database.
func (s *SQLitePromptStore) UpdatePrompt(name, newPrompt, newTags string) error {
	query := "UPDATE prompts SET prompt = ?, tags = ? WHERE name = ?"
	result, err := s.db.Exec(query, newPrompt, newTags, name)
	if err != nil {
		return fmt.Errorf("error updating prompt: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("error checking rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("prompt '%s' not found for update", name)
	}
	return nil
}

// DeletePrompt removes a prompt from the database by name.
func (s *SQLitePromptStore) DeletePrompt(name string) error {
	query := "DELETE FROM prompts WHERE name = ?"
	result, err := s.db.Exec(query, name)
	if err != nil {
		return fmt.Errorf("error deleting prompt: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("error checking rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("prompt '%s' not found for deletion", name)
	}
	return nil
}

// ListPrompts retrieves all prompts from the database.
func (s *SQLitePromptStore) ListPrompts() ([]Prompt, error) {
	query := "SELECT id, name, prompt, tags FROM prompts ORDER BY name"
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("error listing prompts: %w", err)
	}
	defer rows.Close()

	var prompts []Prompt
	for rows.Next() {
		var p Prompt
		if err := rows.Scan(&p.ID, &p.Name, &p.Prompt, &p.Tags); err != nil {
			return nil, fmt.Errorf("error scanning row: %w", err)
		}
		prompts = append(prompts, p)
	}
	return prompts, nil
}

// ListPromptsByTags retrieves prompts filtered by tags using proper set-based filtering.
// Supports AND/OR logic: "tag1,tag2" (OR) or "AND:tag1,tag2" (AND)
func (s *SQLitePromptStore) ListPromptsByTags(tagsFilter string) ([]Prompt, error) {
	// Get all prompts and filter in memory for proper tag matching
	allPrompts, err := s.ListPrompts()
	if err != nil {
		return nil, fmt.Errorf("error listing all prompts: %w", err)
	}
	if tagsFilter == "" {
		return allPrompts, nil
	}

	// Parse filter logic and tags
	var filterTags []string
	useAndLogic := false
	if strings.HasPrefix(tagsFilter, "AND:") {
		useAndLogic = true
		tagsFilter = strings.TrimPrefix(tagsFilter, "AND:")
	}

	for _, tag := range strings.Split(tagsFilter, ",") {
		if trimmed := strings.TrimSpace(tag); trimmed != "" {
			filterTags = append(filterTags, trimmed)
		}
	}

	if len(filterTags) == 0 {
		return allPrompts, nil
	}

	var filteredPrompts []Prompt
	for _, prompt := range allPrompts {
		if prompt.Tags == "" {
			continue
		}

		// Split prompt tags into a set
		promptTagSet := make(map[string]struct{})
		for _, tag := range strings.Split(prompt.Tags, ",") {
			if trimmed := strings.TrimSpace(tag); trimmed != "" {
				promptTagSet[trimmed] = struct{}{}
			}
		}

		if useAndLogic {
			// AND logic: prompt must have ALL filter tags
			hasAllTags := true
			for _, filterTag := range filterTags {
				if _, exists := promptTagSet[filterTag]; !exists {
					hasAllTags = false
					break
				}
			}
			if hasAllTags {
				filteredPrompts = append(filteredPrompts, prompt)
			}
		} else {
			// OR logic: prompt must have ANY filter tag
			hasAnyTag := false
			for _, filterTag := range filterTags {
				if _, exists := promptTagSet[filterTag]; exists {
					hasAnyTag = true
					break
				}
			}
			if hasAnyTag {
				filteredPrompts = append(filteredPrompts, prompt)
			}
		}
	}

	return filteredPrompts, nil
}
