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

// LaunchExternalEditor
func LaunchExternalEditor(initialContent string) (string, error) {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		// Fallback to common editors if EDITOR is not set
		if _, err := exec.LookPath("vim"); err == nil {
			editor = "vim"
		} else if _, err := exec.LookPath("nano"); err == nil {
			editor = "nano"
		} else {
			return "", fmt.Errorf("EDITOR environment variable not set and no default editor (vim, nano) found")
		}
	}

	// Create a temporary file
	tmpfile, err := os.CreateTemp(os.TempDir(), "p-prompt-*.txt")
	if err != nil {
		return "", fmt.Errorf("could not create temporary file: %w", err)
	}
	defer os.Remove(tmpfile.Name()) // Clean up the temporary file

	// Write initial content to the temporary file
	if _, err := tmpfile.WriteString(initialContent); err != nil {
		return "", fmt.Errorf("could not write to temporary file: %w", err)
	}
	if err := tmpfile.Close(); err != nil {
		return "", fmt.Errorf("could not close temporary file: %w", err)
	}

	// Prepare the command to launch the editor
	cmd := exec.Command(editor, tmpfile.Name())
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Run the editor
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("editor command failed: %w", err)
	}

	// Read the edited content from the temporary file
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
	txtArea.CharLimit = 0 // No character limit
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
		case "ctrl+d": // End of transmission, typically used to signal EOF
			m.quitting = true
			return m, tea.Quit
		case "alt+enter": // Custom keybinding for submitting multi-line input
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

// RunTUIEditor launches a TUI-based multi-line editor.
func RunTUIEditor(initialContent string) (string, error) {
	p := tea.NewProgram(initialEditorModel(initialContent))

	m, err := p.Run()
	if err != nil {
		return "", fmt.Errorf("error running TUI editor: %w", err)
	}

	if m, ok := m.(editorModel); ok {
		return m.ta.Value(), nil
	}

	return initialContent, nil // This case should ideally not be reached if the editor exits cleanly after user input.
}
