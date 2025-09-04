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

func LaunchExternalEditor(initialContent string) (string, error) {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		defaultEditors := []string{"vim", "nano"} // Easy to add more, like "emacs"
		for _, e := range defaultEditors {
			if path, err := exec.LookPath(e); err == nil {
				editor = path
				break
			}
		}
	}
	if editor == "" {
		return "", fmt.Errorf("EDITOR environment variable not set and no default editor (vim, nano) found")
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
