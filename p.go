package main

import (
	"fmt"
	"os"

	"github.com/ktr0731/go-fuzzyfinder"
	"github.com/spf13/cobra"
)

// App struct holds the core application logic and its dependencies.
type App struct {
	promptStore    PromptStore
	externalEditor Editor
	tuiEditor      Editor
}

// NewApp creates a new App instance.
func NewApp(store PromptStore, extEditor, tuiEditor Editor) *App {
	return &App{
		promptStore:    store,
		externalEditor: extEditor,
		tuiEditor:      tuiEditor,
	}
}

// AddPrompt handles the logic for adding a new prompt.
func (a *App) AddPrompt(name, tags string, useExternalEditor bool) error {
	var promptContent string
	var err error

	if useExternalEditor {
		fmt.Println("Launching external editor...")
		promptContent, err = a.externalEditor.Edit("")
	} else {
		promptContent, err = a.tuiEditor.Edit("")
	}

	if err != nil {
		return fmt.Errorf("error getting prompt content: %w", err)
	}

	if promptContent == "" {
		return fmt.Errorf("prompt content cannot be empty")
	}

	err = a.promptStore.AddPrompt(name, promptContent, tags)
	if err != nil {
		return fmt.Errorf("error adding prompt: %w", err)
	}
	return nil
}

// DeletePrompt handles the logic for deleting a prompt.
func (a *App) DeletePrompt(name string) error {
	err := a.promptStore.DeletePrompt(name)
	if err != nil {
		return fmt.Errorf("error deleting prompt: %w", err)
	}
	return nil
}

// EditPrompt handles the logic for editing a prompt.
func (a *App) EditPrompt(name, tags string, useExternalEditor bool) error {
	// Get existing prompt to pre-fill editor and preserve unchanged fields
	existingPrompt, err := a.promptStore.GetPromptByName(name)
	if err != nil {
		return fmt.Errorf("error retrieving existing prompt: %w", err)
	}

	var editedPromptContent string
	if useExternalEditor {
		fmt.Println("Launching external editor...")
		editedPromptContent, err = a.externalEditor.Edit(existingPrompt.Prompt)
	} else {
		editedPromptContent, err = a.tuiEditor.Edit(existingPrompt.Prompt)
	}
	if err != nil {
		return fmt.Errorf("error getting edited prompt content: %w", err)
	}

	// Determine the final state of the prompt and tags
	finalPromptContent := editedPromptContent
	finalTags := tags // Use the 'tags' parameter directly

	if finalPromptContent == existingPrompt.Prompt && finalTags == existingPrompt.Tags {
		return fmt.Errorf("no changes detected for prompt or tags")
	}

	// Consistency check: prevent making prompt content empty on edit
	if finalPromptContent == "" {
		return fmt.Errorf("prompt content cannot be empty")
	}

	err = a.promptStore.UpdatePrompt(name, finalPromptContent, finalTags)
	if err != nil {
		return fmt.Errorf("error editing prompt: %w", err)
	}
	return nil
}

// ListPrompts handles the logic for listing prompts.
func (a *App) ListPrompts(tagsFilter string) ([]Prompt, error) {
	prompts, err := a.promptStore.ListPrompts(tagsFilter)
	if err != nil {
		return nil, fmt.Errorf("error listing prompts: %w", err)
	}
	return prompts, nil
}

func printPrompt(p Prompt) {
	fmt.Printf("Name: %s\n", p.Name)
	fmt.Printf("Prompt: %s\n", p.Prompt)
	fmt.Printf("Tags: %s\n", p.Tags)
	fmt.Println("---")
}

func main() {
	db, err := InitDB()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing database: %v\n", err)
		os.Exit(1)
	}

	defer db.Close()

	app := NewApp(NewSQLitePromptStore(db), &ExternalEditor{}, &TUIEditor{})

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

	var addCmd = &cobra.Command{
		Use:   "add [name]",
		Short: "Add a new prompt",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			tags, _ := cmd.Flags().GetString("tags")
			useExternalEditor, _ := cmd.Flags().GetBool("external-editor")

			err := app.AddPrompt(name, tags, useExternalEditor)
			if err != nil {
				return err
			}
			fmt.Println("Prompt added successfully!")
			return nil
		},
	}
	addCmd.Flags().StringP("tags", "t", "", "Tags for the prompt (comma-separated)")
	addCmd.Flags().BoolP("external-editor", "e", false, "Use external editor for prompt content")

	var searchCmd = &cobra.Command{
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

	var deleteCmd = &cobra.Command{
		Use:   "delete [name]",
		Short: "Delete a prompt",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			err := app.DeletePrompt(name)
			if err != nil {
				return err
			}
			fmt.Println("Prompt deleted successfully!")
			return nil
		},
	}

	var editCmd = &cobra.Command{
		Use:   "edit [name]",
		Short: "Edit a prompt",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			useExternalEditor, _ := cmd.Flags().GetBool("external-editor")
			var finalTags string

			if cmd.Flags().Changed("tags") {
				finalTags, _ = cmd.Flags().GetString("tags")
			} else {
				existingPrompt, err := app.promptStore.GetPromptByName(name)
				if err != nil {
					return fmt.Errorf("error retrieving existing prompt for tags: %w", err)
				}
				finalTags = existingPrompt.Tags
			}

			err := app.EditPrompt(name, finalTags, useExternalEditor)
			if err != nil {
				return err
			}
			fmt.Println("Prompt edited successfully!")
			return nil
		},
	}
	editCmd.Flags().StringP("tags", "t", "", "New tags for the prompt (comma-separated)")
	editCmd.Flags().BoolP("external-editor", "e", false, "Use external editor for prompt content")

	var listCmd = &cobra.Command{
		Use:   "list",
		Short: "List all prompts",
		RunE: func(cmd *cobra.Command, args []string) error {
			tags, _ := cmd.Flags().GetString("tags")
			prompts, err := app.ListPrompts(tags)
			if err != nil {
				return err
			}

			for _, p := range prompts {
				printPrompt(p)
			}
			return nil
		},
	}
	listCmd.Flags().StringP("tags", "t", "", "Filter by tags (comma-separated)")

	rootCmd.AddCommand(addCmd, searchCmd, deleteCmd, editCmd, listCmd, versionCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
