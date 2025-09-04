# Directory Structure
```
database/
  init.go
p/
  cli.go
  db.go
  tui-editor.go
main.go
Makefile
```

# Files

## File: database/init.go
```go
package database

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
```

## File: p/db.go
```go
package p

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
	SearchPrompts() ([]Prompt, error)
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

func (s *SQLitePromptStore) SearchPrompts() ([]Prompt, error) {
	rows, err := s.db.Query("SELECT name, prompt, tags FROM prompts")
	if err != nil {
		return nil, fmt.Errorf("error listing prompts for search: %w", err)
	}
	defer rows.Close()

	var prompts []Prompt
	for rows.Next() {
		var p Prompt
		if err := rows.Scan(&p.Name, &p.Prompt, &p.Tags); err != nil {
			return nil, fmt.Errorf("error scanning row for search: %w", err)
		}
		prompts = append(prompts, p)
	}
	return prompts, nil
}
```

## File: Makefile
```
.PHONY: build clean

build: ~/.local/bin
	go build -o ~/.local/bin/p ./main.go
	mkdir -p ~/.local/share/bash-completion/completions
	p completion bash > ~/.local/share/bash-completion/completions/p

clean:
	rm -f ~/.local/bin/p
```

## File: p/cli.go
```go
package p

import (
	"fmt"

	"github.com/ktr0731/go-fuzzyfinder"
	"github.com/spf13/cobra"
)

var promptStore PromptStore


func SetDB(store PromptStore) {
	promptStore = store
}



var AddCmd = &cobra.Command{
	Use:   "add [name]",
	Short: "Add a new prompt",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		tags, _ := cmd.Flags().GetString("tags")
		useExternalEditor, _ := cmd.Flags().GetBool("external-editor")

		var promptContent string
		var err error

		if useExternalEditor {
			fmt.Println("Launching external editor...")
			promptContent, err = LaunchExternalEditor("")
		} else {
			promptContent, err = RunTUIEditor("")
		}

		if err != nil {
			return fmt.Errorf("error getting prompt content: %w", err)
		}

		if promptContent == "" {
			return fmt.Errorf("prompt content cannot be empty")
		}

		err = promptStore.AddPrompt(name, promptContent, tags)
		if err != nil {
			return fmt.Errorf("error adding prompt: %w", err)
		}

		fmt.Println("Prompt added successfully!")
		return nil
	},
}

func init() {
	AddCmd.Flags().StringP("tags", "t", "", "Tags for the prompt (comma-separated)")
	AddCmd.Flags().BoolP("external-editor", "e", false, "Use external editor for prompt content")
}


var SearchCmd = &cobra.Command{
	Use:   "search",
	Short: "Search for prompts using a fuzzy finder",
	RunE: func(cmd *cobra.Command, args []string) error {

		prompts, err := promptStore.SearchPrompts()
		if err != nil {
			return fmt.Errorf("error listing prompts: %w", err)
		}

		idx, err := fuzzyfinder.Find(
			prompts,
			func(i int) string {
				return prompts[i].Name
			},
			fuzzyfinder.WithPreviewWindow(func(i, w, h int) string {
				if i == -1 {
					return ""
				}
				return fmt.Sprintf("Name: %s\nPrompt: %s\nTags: %s", prompts[i].Name, prompts[i].Prompt, prompts[i].Tags)
			}),
		)
		if err != nil {
			return fmt.Errorf("error finding prompt: %w", err)
		}
		printPrompt(prompts[idx])
		return nil
	},
}


var DeleteCmd = &cobra.Command{
	Use:   "delete [name]",
	Short: "Delete a prompt",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		err := promptStore.DeletePrompt(name)
		if err != nil {
			return fmt.Errorf("error deleting prompt: %w", err)
		}

		fmt.Println("Prompt deleted successfully!")
		return nil
	},
}


var EditCmd = &cobra.Command{
	Use:   "edit [name]",
	Short: "Edit a prompt",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		useExternalEditor, _ := cmd.Flags().GetBool("external-editor")


		existingPrompt, err := promptStore.GetPromptByName(name)
		if err != nil {
			return fmt.Errorf("error retrieving existing prompt: %w", err)
		}

		var editedPromptContent string
		if useExternalEditor {
			fmt.Println("Launching external editor...")
			editedPromptContent, err = LaunchExternalEditor(existingPrompt.Prompt)
		} else {
			editedPromptContent, err = RunTUIEditor(existingPrompt.Prompt)
		}
		if err != nil {
			return fmt.Errorf("error getting edited prompt content: %w", err)
		}


		finalPromptContent := editedPromptContent
		finalTags := existingPrompt.Tags

		if cmd.Flags().Changed("tags") {
			finalTags, _ = cmd.Flags().GetString("tags")
		}

		if finalPromptContent == existingPrompt.Prompt && finalTags == existingPrompt.Tags {
			return fmt.Errorf("no changes detected for prompt or tags")
		}


		if finalPromptContent == "" {
			return fmt.Errorf("prompt content cannot be empty")
		}

		err = promptStore.UpdatePrompt(name, finalPromptContent, finalTags)
		if err != nil {
			return fmt.Errorf("error editing prompt: %w", err)
		}

		fmt.Println("Prompt edited successfully!")
		return nil
	},
}

func init() {
	EditCmd.Flags().StringP("tags", "t", "", "New tags for the prompt (comma-separated)")
	EditCmd.Flags().BoolP("external-editor", "e", false, "Use external editor for prompt content")
}


var ListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all prompts",
	RunE: func(cmd *cobra.Command, args []string) error {
		tags, _ := cmd.Flags().GetString("tags")
		prompts, err := promptStore.ListPrompts(tags)
		if err != nil {
			return fmt.Errorf("error listing prompts: %w", err)
		}

		for _, p := range prompts {
			printPrompt(p)
		}
		return nil
	},
}


func printPrompt(p Prompt) {
	fmt.Printf("Name: %s\nPrompt: %s\nTags: %s\n\n", p.Name, p.Prompt, p.Tags)
}

func init() {
	ListCmd.Flags().StringP("tags", "t", "", "Filter by tags (comma-separated)")
}
```

## File: p/tui-editor.go
```go
package p

import (
	"fmt"

	"os"
	"os/exec"
	"strings"

	ta "github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)


func LaunchExternalEditor(initialContent string) (string, error) {
	editor := os.Getenv("EDITOR")
	if editor == "" {

		if _, err := exec.LookPath("vim"); err == nil {
			editor = "vim"
		} else if _, err := exec.LookPath("nano"); err == nil {
			editor = "nano"
		} else {
			return "", fmt.Errorf("EDITOR environment variable not set and no default editor (vim, nano) found")
		}
	}


	tmpfile, err := os.CreateTemp(os.TempDir(), "p-prompt-*.txt")
	if err != nil {
		return "", fmt.Errorf("could not create temporary file: %w", err)
	}
	defer os.Remove(tmpfile.Name())


	if _, err := tmpfile.WriteString(initialContent); err != nil {
		return "", fmt.Errorf("could not write to temporary file: %w", err)
	}
	if err := tmpfile.Close(); err != nil {
		return "", fmt.Errorf("could not close temporary file: %w", err)
	}


	cmd := exec.Command(editor, tmpfile.Name())
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr


	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("editor command failed: %w", err)
	}


	editedContentBytes, err := os.ReadFile(tmpfile.Name())
	if err != nil {
		return "", fmt.Errorf("could not read edited content from temporary file: %w", err)
	}

	return strings.TrimSpace(string(editedContentBytes)), nil
}

type editorModel struct {
	ta       ta.Model
	quitting bool
}

func initialEditorModel(initialContent string) editorModel {
	txtArea := ta.New()
	txtArea.Placeholder = "Enter your prompt..."
	txtArea.Focus()
	txtArea.CharLimit = 0
	txtArea.SetWidth(80)
	txtArea.Prompt = ""
	txtArea.SetValue(initialContent)

	return editorModel{
		ta: txtArea,
	}
}

func (m editorModel) Init() tea.Cmd {
	return ta.Blink
}

func (m editorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.quitting = true
			return m, tea.Quit
		case "ctrl+d":
			m.quitting = true
			return m, tea.Quit
		case "alt+enter":
			m.quitting = true
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.ta, cmd = m.ta.Update(msg)
	return m, cmd
}

func (m editorModel) View() string {
	return lipgloss.JoinVertical(lipgloss.Left,
		"\n"+
			"  "+lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Render("Enter your prompt. Press Alt+Enter or Ctrl+D to save and exit, or Ctrl+C to cancel."),
		"\n"+
			m.ta.View(),
	)
}


func RunTUIEditor(initialContent string) (string, error) {
	p := tea.NewProgram(initialEditorModel(initialContent))

	m, err := p.Run()
	if err != nil {
		return "", fmt.Errorf("error running TUI editor: %w", err)
	}

	if m, ok := m.(editorModel); ok {
		return m.ta.Value(), nil
	}

	return initialContent, nil
}
```

## File: main.go
```go
package main

import (
	"fmt"
	"os"

	"github.com/yookibooki/p/database"
	"github.com/yookibooki/p/p"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "p",
	Short: "p is a CLI for managing LLM prompts",
	Long:  `p is a CLI for managing LLM prompts`,
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of p",
	Long:  `All software has versions. This is p's.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("p version 0.1.0")
	},
}

func init() {
	rootCmd.AddCommand(p.AddCmd)
	rootCmd.AddCommand(p.ListCmd)
	rootCmd.AddCommand(p.EditCmd)
	rootCmd.AddCommand(p.DeleteCmd)
	rootCmd.AddCommand(p.SearchCmd)
	rootCmd.AddCommand(versionCmd)
}

func main() {

	db, err := database.InitDB()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()


	promptStore := p.NewSQLitePromptStore(db)
	p.SetDB(promptStore)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
```
