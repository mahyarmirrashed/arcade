package main

import (
	"fmt"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// recorder is a stub game that remembers the last message it was given.
type recorder struct{ last tea.Msg }

func (r *recorder) Init() tea.Cmd                           { return nil }
func (r *recorder) Update(msg tea.Msg) (tea.Model, tea.Cmd) { r.last = msg; return r, nil }
func (r *recorder) View() string                            { return "" }

func TestStandalone_quitKeysEndSession(t *testing.T) {
	keys := []tea.KeyMsg{
		{Type: tea.KeyRunes, Runes: []rune("q")},
		{Type: tea.KeyCtrlC},
	}
	for _, key := range keys {
		t.Run(key.String(), func(t *testing.T) {
			game := &recorder{}

			_, cmd := standalone{game}.Update(key)

			if cmd == nil {
				t.Fatal("cmd = nil; want quit")
			}
			if _, ok := cmd().(tea.QuitMsg); !ok {
				t.Errorf("cmd() = %T; want tea.QuitMsg", cmd())
			}
			if game.last != nil {
				t.Errorf("game received %v; quit keys should not reach it", game.last)
			}
		})
	}
}

func TestStandalone_otherMessagesReachGame(t *testing.T) {
	game := &recorder{}
	key := tea.KeyMsg{Type: tea.KeyUp}

	next, cmd := standalone{game}.Update(key)

	if got, ok := game.last.(tea.KeyMsg); !ok || got.Type != tea.KeyUp {
		t.Errorf("game received %v; want the up key", game.last)
	}
	if cmd != nil {
		t.Error("cmd != nil; want the game's nil command")
	}
	if _, ok := next.(standalone); !ok {
		t.Errorf("Update returned %T; want standalone so quit keys keep working", next)
	}
}

func TestStandalone_resizeClearsScreenAndReachesGame(t *testing.T) {
	game := &recorder{}

	_, cmd := standalone{game}.Update(tea.WindowSizeMsg{Width: 100, Height: 40})

	if cmd == nil || fmt.Sprintf("%T", cmd()) != fmt.Sprintf("%T", tea.ClearScreen()) {
		t.Error("resize did not clear the screen")
	}
	if _, ok := game.last.(tea.WindowSizeMsg); !ok {
		t.Errorf("game received %v; want the resize", game.last)
	}
}
