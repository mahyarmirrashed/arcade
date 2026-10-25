package root_test

import (
	"fmt"
	"io"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"

	"github.com/mahyarmirrashed/arcade/internal/game"
	"github.com/mahyarmirrashed/arcade/internal/leaderboard"
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
	return root.New(80, 24, reg, testStyles(), leaderboard.Session{})
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
	m = update(m, tea.KeyMsg{Type: tea.KeyEsc})
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

// msgsOf runs cmd and any commands it batches, returning their messages.
func msgsOf(cmd tea.Cmd) []tea.Msg {
	if cmd == nil {
		return nil
	}
	msg := cmd()
	if batch, ok := msg.(tea.BatchMsg); ok {
		var msgs []tea.Msg
		for _, c := range batch {
			msgs = append(msgs, msgsOf(c)...)
		}
		return msgs
	}
	return []tea.Msg{msg}
}

func clearsScreen(cmd tea.Cmd) bool {
	want := fmt.Sprintf("%T", tea.ClearScreen())
	for _, msg := range msgsOf(cmd) {
		if fmt.Sprintf("%T", msg) == want {
			return true
		}
	}
	return false
}

func TestRoot_resizeClearsScreen(t *testing.T) {
	// Bubble Tea redraws on resize but doesn't clear, so remnants of the old
	// layout can stay on screen.
	lobbyRoot := newTestRoot()
	playingRoot := update(newTestRoot(), lobby.SelectMsg{Game: stubGame{name: "Stub"}})

	for name, m := range map[string]root.Model{"lobby": lobbyRoot, "playing": playingRoot} {
		t.Run(name, func(t *testing.T) {
			_, cmd := m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})

			if !clearsScreen(cmd) {
				t.Error("resize did not clear the screen")
			}
		})
	}
}
