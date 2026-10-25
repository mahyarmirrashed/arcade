// Package lobby provides the arcade lobby screen where players choose a game.
package lobby

import (
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/mahyarmirrashed/arcade/internal/game"
	"github.com/mahyarmirrashed/arcade/internal/ui"
)

// SelectMsg is sent when the player chooses a game.
// The root model listens for this and transitions to the selected game.
type SelectMsg struct {
	Game game.Game
}

const (
	listWidth   = 30
	appTagline  = "a collection of games, playable over ssh"
)

// item wraps a game.Game so it satisfies list.Item.
type item struct {
	g game.Game
}

func (i item) Title() string       { return i.g.Name() }
func (i item) Description() string { return i.g.Description() }
func (i item) FilterValue() string { return i.g.Name() }

// newDelegate returns a single-line delegate styled monochromatically.
func newDelegate() list.DefaultDelegate {
	d := list.NewDefaultDelegate()

	d.ShowDescription = false
	d.SetHeight(1)
	d.SetSpacing(0)

	d.Styles.SelectedTitle = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), false, false, false, true).
		BorderForeground(ui.ColorText).
		Foreground(ui.ColorText).
		Bold(true).
		Padding(0, 0, 0, 1)
	d.Styles.NormalTitle = lipgloss.NewStyle().
		Foreground(ui.ColorSubtle).
		Padding(0, 0, 0, 2)

	// Emit SelectMsg on enter.
	d.UpdateFunc = func(msg tea.Msg, m *list.Model) tea.Cmd {
		km, ok := msg.(tea.KeyMsg)
		if !ok {
			return nil
		}
		if km.String() == "enter" {
			if selected, ok := m.SelectedItem().(item); ok {
				return func() tea.Msg { return SelectMsg{Game: selected.g} }
			}
		}
		return nil
	}

	return d
}

// listHeight returns a sensible list height given the terminal height.
// Just enough to show all items without extra empty space.
func listHeight(termHeight int) int {
	h := termHeight / 3
	if h < 6 {
		h = 6
	}
	return h
}

// Model is the Bubbletea model for the lobby screen.
type Model struct {
	list   list.Model
	games  []game.Game
	width  int
	height int
}

// New returns an initialised lobby Model sized to the given terminal dimensions.
func New(width, height int, reg *game.Registry) Model {
	games := reg.All()
	items := make([]list.Item, len(games))
	for i, g := range games {
		items[i] = item{g: g}
	}

	lh := listHeight(height)
	l := list.New(items, newDelegate(), listWidth, lh)

	// No title — we render it ourselves in View.
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.SetShowHelp(false)
	l.DisableQuitKeybindings()
	l.KeyMap.PrevPage.SetKeys("pgup")
	l.KeyMap.NextPage.SetKeys("pgdown")

	return Model{list: l, games: games, width: width, height: height}
}

// Index returns the currently selected list index (used in tests).
func (m Model) Index() int { return m.list.Index() }

// Width returns the current terminal width.
func (m Model) Width() int { return m.width }

// Height returns the current terminal height.
func (m Model) Height() int { return m.height }

// selectedDescription returns the description of the currently highlighted game.
func (m Model) selectedDescription() string {
	i := m.list.Index()
	if i >= 0 && i < len(m.games) {
		return m.games[i].Description()
	}
	return ""
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if ws, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = ws.Width
		m.height = ws.Height
		m.list.SetSize(listWidth, listHeight(ws.Height))
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

// View implements tea.Model.
func (m Model) View() string {
	title   := ui.Title.Render("arcade")
	tagline := ui.Muted.Render(appTagline)
	desc    := ui.Subtitle.Render(m.selectedDescription())
	items   := m.list.View()
	help    := ui.Muted.Render("↑/↓  navigate   enter  select   q  quit")

	content := lipgloss.JoinVertical(lipgloss.Left,
		title,
		"",
		tagline,
		"",
		desc,
		"",
		items,
		"",
		help,
	)

	return lipgloss.Place(
		m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		content,
	)
}
