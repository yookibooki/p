package main

import (
	"fmt"
	"os"

	"github.com/ktr0731/go-fuzzyfinder"
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

// --- Content from cli.go starts here ---

var promptStore PromptStore

// SetDB sets the database connection for the core commands.
func SetDB(store PromptStore) {
	promptStore = store
}

// add
var AddCmd = &cobra.Command{
	Use:   "add [name]",
	Short: "Add a new prompt",
	Args:  cobra.ExactArgs(1), // Only name is a positional argument now
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

// search
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

// delete
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

// edit
var EditCmd = &cobra.Command{
	Use:   "edit [name]",
	Short: "Edit a prompt",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		useExternalEditor, _ := cmd.Flags().GetBool("external-editor")

		// Get existing prompt to pre-fill editor and preserve unchanged fields
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

		// Determine the final state of the prompt and tags
		finalPromptContent := editedPromptContent
		finalTags := existingPrompt.Tags // Default to existing tags

		if cmd.Flags().Changed("tags") {
			finalTags, _ = cmd.Flags().GetString("tags") // Only update if flag was explicitly used
		}

		if finalPromptContent == existingPrompt.Prompt && finalTags == existingPrompt.Tags {
			return fmt.Errorf("no changes detected for prompt or tags")
		}

		// Consistency check: prevent making prompt content empty on edit
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

// list
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

// printPrompt is a helper function to display prompt details.
func printPrompt(p Prompt) {
	fmt.Printf("Name: %s\nPrompt: %s\nTags: %s\n\n", p.Name, p.Prompt, p.Tags)
}

// --- End of content from cli.go ---

func init() {
	rootCmd.AddCommand(AddCmd)
	rootCmd.AddCommand(ListCmd)
	rootCmd.AddCommand(EditCmd)
	rootCmd.AddCommand(DeleteCmd)
	rootCmd.AddCommand(SearchCmd)
	rootCmd.AddCommand(versionCmd)

	// init() content from cli.go
	AddCmd.Flags().StringP("tags", "t", "", "Tags for the prompt (comma-separated)")
	AddCmd.Flags().BoolP("external-editor", "e", false, "Use external editor for prompt content")

	EditCmd.Flags().StringP("tags", "t", "", "New tags for the prompt (comma-separated)")
	EditCmd.Flags().BoolP("external-editor", "e", false, "Use external editor for prompt content")

	ListCmd.Flags().StringP("tags", "t", "", "Filter by tags (comma-separated)")
}

func main() {
	// Initialize the database and handle errors
	db, err := InitDB()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	// Pass the database connection to core commands
	promptStore := NewSQLitePromptStore(db)
	SetDB(promptStore)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
