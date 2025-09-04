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

type PromptStore interface {
	AddPrompt(name, prompt, tags string) error
	GetPromptByName(name string) (*Prompt, error)
	UpdatePrompt(name, newPrompt, newTags string) error
	DeletePrompt(name string) error
	ListPrompts(tagsFilter string) ([]Prompt, error)
}

type SQLitePromptStore struct {
	db *sql.DB
}

func NewSQLitePromptStore(db *sql.DB) *SQLitePromptStore {
	return &SQLitePromptStore{db: db}
}

func (s *SQLitePromptStore) AddPrompt(name, prompt, tags string) error {
	query := "INSERT INTO prompts (name, prompt, tags) VALUES (?, ?, ?)"
	_, err := s.db.Exec(query, name, prompt, tags)
	if err != nil {
		return fmt.Errorf("error adding prompt: %w", err)
	}
	return nil
}

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

func (s *SQLitePromptStore) UpdatePrompt(name, newPrompt, newTags string) error {
	query := "UPDATE prompts SET prompt = ?, tags = ? WHERE name = ?"
	result, err := s.db.Exec(query, newPrompt, newTags, name)
	if err != nil {
		return fmt.Errorf("error updating prompt: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		// This error is less critical but good to have for debugging
		return fmt.Errorf("error checking rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("prompt '%s' not found for update", name)
	}
	return nil
}

func (s *SQLitePromptStore) DeletePrompt(name string) error {
	query := "DELETE FROM prompts WHERE name = ?"
	_, err := s.db.Exec(query, name)
	if err != nil {
		return fmt.Errorf("error deleting prompt: %w", err)
	}
	return nil
}

func (s *SQLitePromptStore) ListPrompts(tagsFilter string) ([]Prompt, error) {
	query := "SELECT name, prompt, tags FROM prompts"
	var queryArgs []interface{}

	if tagsFilter != "" {
		tagList := strings.Split(tagsFilter, ",")
		query += " WHERE"
		for i, tag := range tagList {
			tag = strings.TrimSpace(tag)
			query += " tags LIKE ?"
			queryArgs = append(queryArgs, "%"+tag+"%")
			if i < len(tagList)-1 {
				query += " OR"
			}
		}
	}

	rows, err := s.db.Query(query, queryArgs...)
	if err != nil {
		return nil, fmt.Errorf("error listing prompts: %w", err)
	}
	defer rows.Close()

	var prompts []Prompt
	for rows.Next() {
		var p Prompt
		if err := rows.Scan(&p.Name, &p.Prompt, &p.Tags); err != nil {
			return nil, fmt.Errorf("error scanning row: %w", err)
		}
		prompts = append(prompts, p)
	}
	return prompts, nil
}
