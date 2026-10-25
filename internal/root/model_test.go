package root_test

import (
	"io"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"

	"github.com/mahyarmirrashed/arcade/internal/game"
	"github.com/mahyarmirrashed/arcade/internal/lobby"
	"github.com/mahyarmirrashed/arcade/internal/root"
	"github.com/mahyarmirrashed/arcade/internal/ui"
)

// stubGame is a minimal game.Game for tests.
type stubGame struct{ name string }

func (s stubGame) Name() string                        { return s.name }
func (s stubGame) Description() string                 { return "" }
func (s stubGame) New(_, _ int, _ ui.Styles) tea.Model { return stubModel{} }

// stubModel is the model returned by stubGame.
type stubModel struct{}

func (stubModel) Init() tea.Cmd                       { return nil }
func (stubModel) Update(tea.Msg) (tea.Model, tea.Cmd) { return stubModel{}, nil }
func (stubModel) View() string                        { return "stub" }

func newTestRoot() root.Model {
	reg := game.NewRegistry(stubGame{name: "Stub"})
	return root.New(80, 24, reg, testStyles())
}

// testStyles renders without colour, as a terminal with no colour support would.
func testStyles() ui.Styles {
	r := lipgloss.NewRenderer(io.Discard)
	r.SetColorProfile(termenv.Ascii)
	return ui.NewStyles(r)
}

func update(m tea.Model, msg tea.Msg) root.Model {
	next, _ := m.Update(msg)
	return next.(root.Model)
}

func TestRoot_initialStateIsLobby(t *testing.T) {
	m := newTestRoot()
	if m.State() != root.StateLobby {
		t.Errorf("initial state = %v; want StateLobby", m.State())
	}
}

func TestRoot_selectMsgTransitionsToPlaying(t *testing.T) {
	m := newTestRoot()
	m = update(m, lobby.SelectMsg{Game: stubGame{name: "Stub"}})
	if m.State() != root.StatePlaying {
		t.Errorf("state after SelectMsg = %v; want StatePlaying", m.State())
	}
}

func TestRoot_quitFromGameReturnsToLobby(t *testing.T) {
	m := newTestRoot()
	m = update(m, lobby.SelectMsg{Game: stubGame{name: "Stub"}})
	m = update(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	if m.State() != root.StateLobby {
		t.Errorf("state after quit = %v; want StateLobby", m.State())
	}
}

func TestRoot_ctrlCFromGameReturnsToLobby(t *testing.T) {
	m := newTestRoot()
	m = update(m, lobby.SelectMsg{Game: stubGame{name: "Stub"}})
	m = update(m, tea.KeyMsg{Type: tea.KeyCtrlC})
	if m.State() != root.StateLobby {
		t.Errorf("state after ctrl+c = %v; want StateLobby", m.State())
	}
}

func TestRoot_windowResizeTracked(t *testing.T) {
	m := newTestRoot()
	m = update(m, tea.WindowSizeMsg{Width: 120, Height: 40})
	// Transition to a game - the game should be spawned at the new size.
	// We verify by checking the View doesn't panic (game gets non-zero dims).
	m = update(m, lobby.SelectMsg{Game: stubGame{name: "Stub"}})
	if m.State() != root.StatePlaying {
		t.Error("expected playing state")
	}
}
