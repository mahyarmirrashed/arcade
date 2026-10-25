// Package lobby provides the arcade lobby screen where players choose a game.
package lobby

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/mahyarmirrashed/arcade/internal/ui"
)

// SelectMsg is sent when the player chooses a game.
// The root model listens for this and transitions to the selected game.
// NOTE: wired up fully in Stage 2B — placeholder for now.
type SelectMsg struct{}

// Model is the Bubbletea model for the lobby screen.
type Model struct {
	width  int
	height int
}

// New returns an initialised lobby Model sized to the given terminal dimensions.
func New(width, height int) Model {
	return Model{width: width, height: height}
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
func (m Model) View() string {
	content := ui.Title.Render("arcade") + "\n\n" +
		ui.Muted.Render("games coming soon — check back shortly") + "\n\n" +
		ui.Muted.Render("q  quit")

	return lipgloss.Place(
		m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		ui.Border.Render(content),
	)
}
