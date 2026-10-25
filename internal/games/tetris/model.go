// Package tetris implements the Tetris game for the arcade.
package tetris

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/mahyarmirrashed/arcade/internal/game"
	"github.com/mahyarmirrashed/arcade/internal/ui"
)

// Compile-time check: Model must satisfy game.Game.
var _ game.Game = (*Model)(nil)

// Model is the Bubbletea model for Tetris.
// Full implementation lives in Stage 2B.
type Model struct {
	width  int
	height int
}

// — game.Game implementation —

// Name implements game.Game.
func (m *Model) Name() string { return "Tetris" }

// Description implements game.Game.
func (m *Model) Description() string { return "clear lines, survive gravity" }

// New implements game.Game — returns a fresh model for a new session.
func (m *Model) New(width, height int) tea.Model {
	return &Model{width: width, height: height}
}

// — tea.Model implementation —

// Init implements tea.Model.
func (m *Model) Init() tea.Cmd { return nil }

// Update implements tea.Model.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		}
	}
	return m, nil
}

// View implements tea.Model.
func (m *Model) View() string {
	content := ui.Title.Render("tetris") + "\n\n" +
		ui.Muted.Render("coming soon") + "\n\n" +
		ui.Muted.Render("q  quit")

	return lipgloss.Place(
		m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		ui.Border.Render(content),
	)
}
