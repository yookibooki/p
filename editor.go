package main

// Editor defines the interface for editing content.
type Editor interface {
	Edit(initialContent string) (string, error)
}

// TUIEditor is an editor that uses a terminal user interface.
type TUIEditor struct{}

// Edit launches the TUI editor.
func (t *TUIEditor) Edit(initialContent string) (string, error) {
	return RunTUIEditor(initialContent)
}

// ExternalEditor is an editor that uses an external command-line editor.
type ExternalEditor struct{}

// Edit launches the external editor.
func (e *ExternalEditor) Edit(initialContent string) (string, error) {
	return LaunchExternalEditor(initialContent)
}
