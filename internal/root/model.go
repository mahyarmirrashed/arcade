// Package root provides the top-level Bubbletea model for the arcade.
// It owns the lobby↔game state machine for a single SSH session.
package root

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/mahyarmirrashed/arcade/internal/game"
	"github.com/mahyarmirrashed/arcade/internal/lobby"
)

type state int

const (
	stateLobby   state = iota
	statePlaying state = iota
)

// Model is the root Bubbletea model. It holds either the lobby or an active
// game and delegates all messages to whichever is current.
type Model struct {
	state  state
	width  int
	height int
	lobby  lobby.Model
	active tea.Model // non-nil only while statePlaying
}

// New returns an initialised root Model.
func New(width, height int, reg *game.Registry) Model {
	return Model{
		state:  stateLobby,
		width:  width,
		height: height,
		lobby:  lobby.New(width, height, reg),
	}
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return m.lobby.Init()
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Always track terminal size so we can pass it to freshly spawned games.
	if ws, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = ws.Width
		m.height = ws.Height
	}

	switch m.state {
	case stateLobby:
		return m.updateLobby(msg)
	case statePlaying:
		return m.updatePlaying(msg)
	}
	return m, nil
}

func (m Model) updateLobby(msg tea.Msg) (tea.Model, tea.Cmd) {
	// q / ctrl+c in the lobby quits the program.
	if km, ok := msg.(tea.KeyMsg); ok {
		switch km.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		}
	}

	// Intercept SelectMsg before forwarding to lobby so we can transition.
	if sel, ok := msg.(lobby.SelectMsg); ok {
		g := sel.Game.New(m.width, m.height)
		m.active = g
		m.state = statePlaying
		return m, g.Init()
	}

	next, cmd := m.lobby.Update(msg)
	m.lobby = next.(lobby.Model)
	return m, cmd
}

func (m Model) updatePlaying(msg tea.Msg) (tea.Model, tea.Cmd) {
	// q or ctrl+c while in a game returns to the lobby.
	if km, ok := msg.(tea.KeyMsg); ok {
		switch km.String() {
		case "q", "ctrl+c":
			m.state = stateLobby
			m.active = nil
			// Re-send the window size so the lobby re-layouts.
			return m, func() tea.Msg {
				return tea.WindowSizeMsg{Width: m.width, Height: m.height}
			}
		}
	}

	next, cmd := m.active.Update(msg)
	m.active = next
	return m, cmd
}

// View implements tea.Model.
func (m Model) View() string {
	switch m.state {
	case statePlaying:
		return m.active.View()
	default:
		return m.lobby.View()
	}
}

// State returns the current state (exported for tests).
func (m Model) State() state { return m.state }

// StateLobby and StatePlaying are the exported state constants for tests.
const (
	StateLobby   = stateLobby
	StatePlaying = statePlaying
)
