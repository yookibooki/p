package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/ktr0731/go-fuzzyfinder"
	"github.com/spf13/cobra"
)

var Version = "dev"

type App struct {
	promptStore *SQLitePromptStore
}

func NewApp(store *SQLitePromptStore) *App {
	return &App{promptStore: store}
}

func (a *App) AddPrompt(name, tags string, useExternalEditor bool) error {
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

	err = a.promptStore.AddPrompt(name, promptContent, tags)
	if err != nil {
		return fmt.Errorf("error adding prompt: %w", err)
	}
	return nil
}

func (a *App) DeletePrompt(name string) error {
	err := a.promptStore.DeletePrompt(name)
	if err != nil {
		return fmt.Errorf("error deleting prompt: %w", err)
	}
	return nil
}

func (a *App) EditPrompt(name, newPrompt, newTags string) error {
	existingPrompt, err := a.promptStore.GetPromptByName(name)
	if err != nil {
		return fmt.Errorf("error retrieving existing prompt: %w", err)
	}

	// The logic to check for changes can also be simplified or moved here
	if newPrompt == existingPrompt.Prompt && newTags == existingPrompt.Tags {
		fmt.Println("No changes detected for prompt or tags.")
		return nil
	}

	if newPrompt == "" {
		return fmt.Errorf("prompt content cannot be empty")
	}

	err = a.promptStore.UpdatePrompt(name, newPrompt, newTags)
	if err != nil {
		return fmt.Errorf("error editing prompt: %w", err)
	}
	return nil
}

func (a *App) ListPrompts(tagsFilter string) ([]Prompt, error) {
	prompts, err := a.promptStore.ListPrompts()
	if err != nil {
		return nil, fmt.Errorf("error listing prompts: %w", err)
	}

	if tagsFilter == "" {
		return prompts, nil
	}

	filterTags := map[string]struct{}{}
	for _, t := range strings.Split(tagsFilter, ",") {
		filterTags[strings.TrimSpace(t)] = struct{}{}
	}

	var filteredPrompts []Prompt
	for _, p := range prompts {
		promptTags := strings.Split(p.Tags, ",")
		for _, pt := range promptTags {
			if _, ok := filterTags[strings.TrimSpace(pt)]; ok {
				filteredPrompts = append(filteredPrompts, p)
				break
			}
		}
	}

	return filteredPrompts, nil
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

	app := NewApp(NewSQLitePromptStore(db))

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
			useExternalEditor, _ := cmd.Flags().GetBool("external-editor")

			if err := app.AddPrompt(name, tags, useExternalEditor); err != nil {
				return err
			}
			fmt.Println("Prompt added successfully!")
			return nil
		},
	}
	cmd.Flags().StringP("tags", "t", "", "Tags for the prompt (comma-separated)")
	cmd.Flags().BoolP("external-editor", "e", false, "Use external editor for prompt content")
	return cmd
}

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
	return &cobra.Command{
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
	}
}

func newEditCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "edit [name]",
		Short: "Edit a prompt",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			useExternalEditor, _ := cmd.Flags().GetBool("external-editor")

			existingPrompt, err := app.promptStore.GetPromptByName(name)
			if err != nil {
				return err
			}

			finalTags := existingPrompt.Tags
			if cmd.Flags().Changed("tags") {
				finalTags, _ = cmd.Flags().GetString("tags")
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

			if err := app.EditPrompt(name, editedPromptContent, finalTags); err != nil {
				return err
			}
			fmt.Println("Prompt edited successfully!")
			return nil
		},
	}
	cmd.Flags().StringP("tags", "t", "", "New tags for the prompt (comma-separated)")
	cmd.Flags().BoolP("external-editor", "e", false, "Use external editor for prompt content")
	return cmd
}

func newListCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
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
	cmd.Flags().StringP("tags", "t", "", "Filter by tags (comma-separated)")
	return cmd
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
