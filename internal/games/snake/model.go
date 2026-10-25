// Package snake implements the Snake game for the arcade.
package snake

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/mahyarmirrashed/arcade/internal/game"
	"github.com/mahyarmirrashed/arcade/internal/ui"
)

// Compile-time check: Model must satisfy game.Game.
var _ game.Game = (*Model)(nil)

// Model is the Bubbletea model for Snake.
// Full implementation lives in Stage 2A.
type Model struct {
	width  int
	height int
}

// — game.Game implementation —

// Name implements game.Game.
func (m *Model) Name() string { return "Snake" }

// Description implements game.Game.
func (m *Model) Description() string { return "eat, grow, don't bite yourself" }

// New implements game.Game — returns a fresh model for a new session.
func (m *Model) New(width, height int) tea.Model {
	return &Model{width: width, height: height}
}

// — tea.Model implementation —

// Init implements tea.Model.
func (m *Model) Init() tea.Cmd { return nil }

// Update implements tea.Model.
// q and ctrl+c are intentionally not handled here — the root model intercepts
// them and returns the player to the lobby.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if ws, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = ws.Width
		m.height = ws.Height
	}
	return m, nil
}

// View implements tea.Model.
func (m *Model) View() string {
	content := lipgloss.JoinVertical(lipgloss.Left,
		ui.Title.Render("snake"),
		"",
		ui.Muted.Render("coming soon"),
	)

	return lipgloss.Place(
		m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		content,
	)
}
