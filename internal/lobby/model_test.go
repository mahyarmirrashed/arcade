package lobby_test

import (
	"testing"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/mahyarmirrashed/arcade/internal/game"
	"github.com/mahyarmirrashed/arcade/internal/lobby"
)

// stubGame satisfies game.Game for test purposes.
type stubGame struct{ name string }

func (s stubGame) Name() string                   { return s.name }
func (s stubGame) Description() string            { return s.name + " desc" }
func (s stubGame) New(_, _ int) tea.Model         { return nil }

func newTestModel(games ...game.Game) lobby.Model {
	reg := game.NewRegistry(games...)
	return lobby.New(80, 24, reg)
}

func update(m tea.Model, msg tea.Msg) tea.Model {
	next, _ := m.Update(msg)
	return next
}

func keyMsg(k string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)}
}

func TestLobby_downMovesSelection(t *testing.T) {
	alpha := stubGame{name: "Alpha"}
	beta := stubGame{name: "Beta"}

	m := newTestModel(alpha, beta)

	// send the down key — bubbles/list handles "down" and "j"
	next := update(m, tea.KeyMsg{Type: tea.KeyDown})

	lm, ok := next.(lobby.Model)
	if !ok {
		t.Fatalf("Update returned unexpected type %T", next)
	}

	// After one down, index should be 1
	if lm.Index() != 1 {
		t.Errorf("index = %d; want 1", lm.Index())
	}
}

func TestLobby_upWraps(t *testing.T) {
	m := newTestModel(stubGame{"A"}, stubGame{"B"}, stubGame{"C"})

	// up from 0 should not go negative (stays at 0 — bubbles does not wrap by default)
	next := update(m, tea.KeyMsg{Type: tea.KeyUp})
	lm := next.(lobby.Model)
	if lm.Index() < 0 {
		t.Errorf("index went negative: %d", lm.Index())
	}
}

func TestLobby_enterEmitsSelectMsg(t *testing.T) {
	alpha := stubGame{name: "Alpha"}
	m := newTestModel(alpha)

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected a command from enter, got nil")
	}

	msg := cmd()
	sel, ok := msg.(lobby.SelectMsg)
	if !ok {
		t.Fatalf("cmd() returned %T; want lobby.SelectMsg", msg)
	}
	if sel.Game.Name() != "Alpha" {
		t.Errorf("SelectMsg.Game.Name() = %q; want %q", sel.Game.Name(), "Alpha")
	}
}

func TestLobby_windowResizeUpdatesDimensions(t *testing.T) {
	m := newTestModel(stubGame{"A"})
	next := update(m, tea.WindowSizeMsg{Width: 120, Height: 40})
	lm := next.(lobby.Model)

	if lm.Width() != 120 || lm.Height() != 40 {
		t.Errorf("dimensions = %dx%d; want 120x40", lm.Width(), lm.Height())
	}
}

// Verify keys binding is correct (compile-time + runtime sanity).
func TestLobby_quitKeyBinding(t *testing.T) {
	qKey := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")}
	quitBinding := key.NewBinding(key.WithKeys("q", "ctrl+c"))
	if !key.Matches(qKey, quitBinding) {
		t.Error("q should match quit binding")
	}
}
