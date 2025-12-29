package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	ta "github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// LaunchExternalEditor opens an external editor to capture prompt content.
func LaunchExternalEditor(initialContent string) (string, error) {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		defaultEditors := []string{"vim", "nano", "vi"} // Easy to add more, like "emacs"
		for _, e := range defaultEditors {
			if path, err := exec.LookPath(e); err == nil {
				editor = path
				break
			}
		}
	}
	if editor == "" {
		return "", fmt.Errorf("EDITOR environment variable not set and no default editor (vim, nano, vi) found")
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

// editorModel represents the TUI editor state with a textarea and quit flag.
type editorModel struct {
	ta       ta.Model
	quitting bool
}

// initialEditorModel creates a new editor model with the given initial content.
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

// Init initializes the editor model and returns the blink command for the textarea.
func (m editorModel) Init() tea.Cmd {
	return ta.Blink
}

// Update handles key events and updates the editor model.
func (m editorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case msg.Type == tea.KeyEsc || msg.Type == tea.KeyCtrlC:
			m.quitting = true
			return m, tea.Quit
		case msg.Type == tea.KeyCtrlD:
			m.quitting = true
			return m, tea.Quit
		case msg.Type == tea.KeyEnter && msg.Alt:
			m.quitting = true
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.ta, cmd = m.ta.Update(msg)
	return m, cmd
}

// View renders the editor interface with instructions and textarea.
func (m editorModel) View() string {
	return lipgloss.JoinVertical(lipgloss.Left,
		"\n"+
			"  "+lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Render("Enter your prompt. Press Alt+Enter or Ctrl+D to save, Esc or Ctrl+C to cancel."),
		"\n"+
			m.ta.View(),
	)
}

// RunTUIEditor launches the Bubble Tea TUI for editing prompt content.
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
