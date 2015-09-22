package common

import "golang.org/x/crypto/ssh/terminal"

// Terminal wraps an ssh.Terminal in order to supply an interface for easy mocking
type Terminal interface {
	GetPrompt() string
	ResetPrompt()
	SetPrompt(p string)
}

// NewTerminal creates a new terminal wrapper
func NewTerminal(term *terminal.Terminal, prompt string) Terminal {
	return &basicTerminal{term, prompt, prompt}
}

type basicTerminal struct {
	term          *terminal.Terminal
	defaultPrompt string
	currentPrompt string
}

func (t *basicTerminal) GetPrompt() string {
	return t.currentPrompt
}

func (t *basicTerminal) ResetPrompt() {
	t.currentPrompt = t.defaultPrompt
	t.term.SetPrompt(t.currentPrompt)
}

func (t *basicTerminal) SetPrompt(p string) {
	t.currentPrompt = p
	t.term.SetPrompt(p)
}
