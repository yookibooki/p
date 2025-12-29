package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/ktr0731/go-fuzzyfinder"
	"github.com/spf13/cobra"
)

var Version = "dev"

const (
	MaxPromptNameLen    = 255
	MaxPromptContentLen = 10000
)

// toTagSet splits, trims, and deduplicates tags into a set.
func toTagSet(tags string) map[string]struct{} {
	tagSet := make(map[string]struct{})
	for _, tag := range strings.Split(tags, ",") {
		if trimmed := strings.TrimSpace(tag); trimmed != "" {
			tagSet[trimmed] = struct{}{}
		}
	}
	return tagSet
}

// normalizeTags deduplicates and sorts comma-separated tags, trimming whitespace.
func normalizeTags(tags string) string {
	if tags == "" {
		return ""
	}
	tagSet := toTagSet(tags)
	var result []string
	for tag := range tagSet {
		result = append(result, tag)
	}
	sort.Strings(result)
	return strings.Join(result, ",")
}

// addExternalEditorFlag adds the external-editor flag to a command.
func addExternalEditorFlag(cmd *cobra.Command) {
	cmd.Flags().BoolP("external-editor", "e", false, "Use external editor for prompt content")
}

// validatePromptName checks if prompt name meets basic requirements.
func validatePromptName(name string) error {
	if name == "" {
		return fmt.Errorf("prompt name cannot be empty")
	}
	if len(name) > MaxPromptNameLen {
		return fmt.Errorf("prompt name too long (%d chars), maximum %d characters", len(name), MaxPromptNameLen)
	}
	return nil
}

// validatePromptContent checks if prompt content meets basic requirements.
func validatePromptContent(content string) error {
	trimmed := strings.TrimSpace(content)
	if len(trimmed) == 0 {
		return fmt.Errorf("prompt content cannot be empty")
	}
	if len(trimmed) > MaxPromptContentLen {
		return fmt.Errorf("prompt content too long (%d chars), maximum %d characters", len(trimmed), MaxPromptContentLen)
	}
	return nil
}

// App manages prompt-related operations using a SQLite store.
type App struct {
	promptStore *SQLitePromptStore
	dbPath      string
}

// NewApp creates a new App instance with the given prompt store and database path.
func NewApp(store *SQLitePromptStore, dbPath string) *App {
	return &App{promptStore: store, dbPath: dbPath}
}

// AddPrompt creates a new prompt using either external editor or TUI editor.
func (a *App) AddPrompt(name, tags string, useExternalEditor bool) error {
	if err := validatePromptName(name); err != nil {
		return err
	}

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
		fmt.Println("Operation cancelled. No prompt added.")
		return nil
	}

	if err := validatePromptContent(promptContent); err != nil {
		return err
	}

	normalizedTags := normalizeTags(tags)
	err = a.promptStore.AddPrompt(name, promptContent, normalizedTags)
	if err != nil {
		return err
	}
	return nil
}

// DeletePrompt removes a prompt by name from the store.
func (a *App) DeletePrompt(name string) error {
	err := a.promptStore.DeletePrompt(name)
	if err != nil {
		return fmt.Errorf("error deleting prompt: %w", err)
	}
	return nil
}

// EditPrompt updates an existing prompt's content and tags.
func (a *App) EditPrompt(existingPrompt *Prompt, newPrompt, newTags string) error {
	if err := validatePromptName(existingPrompt.Name); err != nil {
		return err
	}

	normalizedTags := normalizeTags(newTags)
	// The logic to check for changes can also be simplified or moved here
	if newPrompt == existingPrompt.Prompt && normalizedTags == existingPrompt.Tags {
		fmt.Println("No changes detected for prompt or tags.")
		return nil
	}

	if err := validatePromptContent(newPrompt); err != nil {
		return err
	}

	err := a.promptStore.UpdatePrompt(existingPrompt.Name, newPrompt, normalizedTags)
	if err != nil {
		return fmt.Errorf("error editing prompt: %w", err)
	}
	return nil
}

// ListPrompts retrieves all prompts, optionally filtered by tags.
// Supports AND/OR logic: "tag1,tag2" (OR) or "AND:tag1,tag2" (AND)
func (a *App) ListPrompts(tagsFilter string) ([]Prompt, error) {
	if tagsFilter != "" {
		return a.promptStore.ListPromptsByTags(tagsFilter)
	}
	return a.promptStore.ListPrompts()
}

// printPrompt formats and prints a Prompt struct to stdout.
func printPrompt(p Prompt) {
	fmt.Printf("Name: %s\n", p.Name)
	fmt.Printf("Prompt: %s\n", p.Prompt)
	fmt.Printf("Tags: %s\n", p.Tags)
	fmt.Println("---")
}

// getPromptNames returns all prompt names for shell completion.
func getPromptNames(app *App) []string {
	prompts, err := app.ListPrompts("")
	if err != nil {
		return []string{}
	}
	names := make([]string, len(prompts))
	for i, p := range prompts {
		names[i] = p.Name
	}
	return names
}

// getAllTags returns all unique tags for shell completion.
func getAllTags(app *App) []string {
	prompts, err := app.ListPrompts("")
	if err != nil {
		return []string{}
	}
	tagSet := make(map[string]struct{})
	for _, p := range prompts {
		if p.Tags != "" {
			for tag := range toTagSet(p.Tags) {
				tagSet[tag] = struct{}{}
			}
		}
	}
	tags := make([]string, 0, len(tagSet))
	for tag := range tagSet {
		tags = append(tags, tag)
	}
	sort.Strings(tags)
	return tags
}

func main() {
	db, dbPath, err := InitDBWithPath()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	app := NewApp(NewSQLitePromptStore(db), dbPath)

	rootCmd := &cobra.Command{
		Use:   "p",
		Short: "p is a CLI for managing LLM prompts",
	}

	rootCmd.AddCommand(
		newAddCmd(app),
		newSearchCmd(app),
		newDeleteCmd(app),
		newEditCmd(app),
		newListCmd(app),
		newExportCmd(app),
		newImportCmd(app),
		newBackupCmd(app),
		newRestoreCmd(app),
		newVersionCmd(),
	)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func newAddCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add [name]",
		Short: "Add a new prompt",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			tags, err := cmd.Flags().GetString("tags")
			if err != nil {
				return fmt.Errorf("could not parse tags flag: %w", err)
			}
			useExternalEditor, err := cmd.Flags().GetBool("external-editor")
			if err != nil {
				return fmt.Errorf("could not parse external-editor flag: %w", err)
			}

			if err := app.AddPrompt(name, tags, useExternalEditor); err != nil {
				return err
			}
			fmt.Println("Prompt added successfully!")
			return nil
		},
	}
	cmd.Flags().StringP("tags", "t", "", "Tags for the prompt (comma-separated)")
	addExternalEditorFlag(cmd)

	// Add completion for tags flag
	_ = cmd.RegisterFlagCompletionFunc("tags", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return getAllTags(app), cobra.ShellCompDirectiveNoFileComp
	})

	return cmd
}

// Uses go-fuzzyfinder for enhanced UX in interactive prompt search; stdlib filtering could suffice for simpler needs.
func newSearchCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "search",
		Short: "Search for prompts using a fuzzy finder",
		RunE: func(cmd *cobra.Command, args []string) error {
			prompts, err := app.ListPrompts("")
			if err != nil {
				return err
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
}

func newDeleteCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete [name]",
		Short: "Delete a prompt",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			if err := app.DeletePrompt(name); err != nil {
				return err
			}
			fmt.Println("Prompt deleted successfully!")
			return nil
		},
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) == 0 {
				return getPromptNames(app), cobra.ShellCompDirectiveNoFileComp
			}
			return nil, cobra.ShellCompDirectiveNoFileComp
		},
	}
	return cmd
}

func newEditCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "edit [name]",
		Short: "Edit a prompt",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			useExternalEditor, err := cmd.Flags().GetBool("external-editor")
			if err != nil {
				return fmt.Errorf("could not parse external-editor flag: %w", err)
			}

			existingPrompt, err := app.promptStore.GetPromptByName(name)
			if err != nil {
				return err
			}

			finalTags := existingPrompt.Tags
			if cmd.Flags().Changed("tags") {
				finalTags, err = cmd.Flags().GetString("tags")
				if err != nil {
					return fmt.Errorf("could not parse tags flag: %w", err)
				}
			}

			var editedPromptContent string
			if useExternalEditor {
				fmt.Println("Launching external editor...")
				editedPromptContent, err = LaunchExternalEditor(existingPrompt.Prompt)
			} else {
				editedPromptContent, err = RunTUIEditor(existingPrompt.Prompt)
			}
			if err != nil {
				return err
			}

			if err := app.EditPrompt(existingPrompt, editedPromptContent, finalTags); err != nil {
				return err
			}
			fmt.Println("Prompt edited successfully!")
			return nil
		},
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) == 0 {
				return getPromptNames(app), cobra.ShellCompDirectiveNoFileComp
			}
			return nil, cobra.ShellCompDirectiveNoFileComp
		},
	}
	cmd.Flags().StringP("tags", "t", "", "New tags for the prompt (comma-separated)")
	addExternalEditorFlag(cmd)

	// Add completion for tags flag
	_ = cmd.RegisterFlagCompletionFunc("tags", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return getAllTags(app), cobra.ShellCompDirectiveNoFileComp
	})

	return cmd
}

func newListCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all prompts",
		RunE: func(cmd *cobra.Command, args []string) error {
			tags, _ := cmd.Flags().GetString("tags")
			if tags == "" {
				fmt.Println("No tags specified, listing all prompts")
			}
			prompts, err := app.ListPrompts(tags)
			if err != nil {
				return err
			}

			if len(prompts) == 0 && tags != "" {
				fmt.Printf("No prompts found for tags: %s\n", tags)
				return nil
			}

			for _, p := range prompts {
				printPrompt(p)
			}
			return nil
		},
	}
	cmd.Flags().StringP("tags", "t", "", "Filter by tags (comma-separated). Use AND:tag1,tag2 for AND logic, tag1,tag2 for OR logic")

	// Add completion for tags flag
	_ = cmd.RegisterFlagCompletionFunc("tags", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return getAllTags(app), cobra.ShellCompDirectiveNoFileComp
	})

	return cmd
}

func newExportCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "export [file]",
		Short: "Export all prompts to a JSON file",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			filename := args[0]

			prompts, err := app.ListPrompts("")
			if err != nil {
				return fmt.Errorf("error listing prompts: %w", err)
			}

			if len(prompts) == 0 {
				fmt.Println("No prompts to export")
				return nil
			}

			// Create JSON export
			data, err := json.MarshalIndent(prompts, "", "  ")
			if err != nil {
				return fmt.Errorf("error marshaling prompts: %w", err)
			}

			err = os.WriteFile(filename, data, 0o644)
			if err != nil {
				return fmt.Errorf("error writing file: %w", err)
			}

			fmt.Printf("Exported %d prompts to %s\n", len(prompts), filename)
			return nil
		},
	}
}

func newImportCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "import [file]",
		Short: "Import prompts from a JSON file",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			filename := args[0]

			data, err := os.ReadFile(filename)
			if err != nil {
				return fmt.Errorf("error reading file: %w", err)
			}

			var prompts []Prompt
			err = json.Unmarshal(data, &prompts)
			if err != nil {
				return fmt.Errorf("error unmarshaling prompts: %w", err)
			}

			imported := 0
			skipped := 0
			for _, prompt := range prompts {
				// Skip prompts with empty names or content
				if prompt.Name == "" || prompt.Prompt == "" {
					skipped++
					continue
				}

				err := app.promptStore.AddPrompt(prompt.Name, prompt.Prompt, prompt.Tags)
				if err != nil {
					// If prompt already exists, try to update it
					if strings.Contains(err.Error(), "already exists") {
						err = app.promptStore.UpdatePrompt(prompt.Name, prompt.Prompt, prompt.Tags)
						if err != nil {
							fmt.Printf("Warning: failed to update prompt '%s': %v\n", prompt.Name, err)
							skipped++
							continue
						}
					} else {
						fmt.Printf("Warning: failed to import prompt '%s': %v\n", prompt.Name, err)
						skipped++
						continue
					}
				}
				imported++
			}

			fmt.Printf("Imported %d prompts, skipped %d\n", imported, skipped)
			return nil
		},
	}
}

func newBackupCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "backup [file]",
		Short: "Backup the database to a file",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			backupPath := args[0]

			// Copy the database file
			sourceFile, err := os.Open(app.dbPath)
			if err != nil {
				return fmt.Errorf("error opening database file: %w", err)
			}
			defer sourceFile.Close()

			destFile, err := os.Create(backupPath)
			if err != nil {
				return fmt.Errorf("error creating backup file: %w", err)
			}
			defer destFile.Close()

			_, err = destFile.ReadFrom(sourceFile)
			if err != nil {
				return fmt.Errorf("error copying database: %w", err)
			}

			fmt.Printf("Database backed up to %s\n", backupPath)
			return nil
		},
	}
}

func newRestoreCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "restore [file]",
		Short: "Restore the database from a backup file",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			backupPath := args[0]

			// Check if backup file exists
			if _, err := os.Stat(backupPath); os.IsNotExist(err) {
				return fmt.Errorf("backup file does not exist: %s", backupPath)
			}

			// Close the current database connection
			// Note: This is a limitation - we can't close the DB from within the app
			// In a real implementation, you'd want to handle this differently

			// Copy the backup file to the database location
			sourceFile, err := os.Open(backupPath)
			if err != nil {
				return fmt.Errorf("error opening backup file: %w", err)
			}
			defer sourceFile.Close()

			destFile, err := os.Create(app.dbPath)
			if err != nil {
				return fmt.Errorf("error creating database file: %w", err)
			}
			defer destFile.Close()

			_, err = destFile.ReadFrom(sourceFile)
			if err != nil {
				return fmt.Errorf("error restoring database: %w", err)
			}

			fmt.Printf("Database restored from %s\n", backupPath)
			fmt.Println("Note: You may need to restart the application for changes to take effect")
			return nil
		},
	}
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the version number of p",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("p version %s\n", Version)
		},
	}
}
