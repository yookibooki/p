package main

import (
	"fmt"
	"os"
	"p/database"
	"p/p"

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
	// Initialize the database and handle errors
	db, err := database.InitDB()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	// Pass the database connection to core commands
	promptStore := p.NewSQLitePromptStore(db)
	p.SetDB(promptStore)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
