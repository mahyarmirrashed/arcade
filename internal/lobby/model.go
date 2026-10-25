// Package lobby provides the arcade lobby screen where players choose a game.
package lobby

import (
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/mahyarmirrashed/arcade/internal/game"
	"github.com/mahyarmirrashed/arcade/internal/leaderboard"
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
func newDelegate(styles ui.Styles) list.DefaultDelegate {
	d := list.NewDefaultDelegate()

	d.ShowDescription = false
	d.SetHeight(1)
	d.SetSpacing(0)

	d.Styles.SelectedTitle = styles.NewStyle().
		Border(lipgloss.NormalBorder(), false, false, false, true).
		BorderForeground(ui.ColorText).
		Foreground(ui.ColorText).
		Bold(true).
		Padding(0, 0, 0, 1)
	d.Styles.NormalTitle = styles.NewStyle().
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
	styles ui.Styles
	lb     leaderboard.Session
}

// New returns an initialised lobby Model sized to the given terminal
// dimensions, rendering with the session's styles. lb is the player's
// leaderboard session.
func New(width, height int, reg *game.Registry, styles ui.Styles, lb leaderboard.Session) Model {
	games := reg.All()
	items := make([]list.Item, len(games))
	for i, g := range games {
		items[i] = item{g: g}
	}

	lh := listHeight(height)
	l := list.New(items, newDelegate(styles), listWidth, lh)

	// No title - we render it ourselves in View.
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.SetShowHelp(false)
	l.DisableQuitKeybindings()
	l.KeyMap.PrevPage.SetKeys("pgup")
	l.KeyMap.NextPage.SetKeys("pgdown")

	return Model{list: l, games: games, width: width, height: height, styles: styles, lb: lb}
}

// Index returns the currently selected list index (used in tests).
func (m Model) Index() int { return m.list.Index() }

// Width returns the current terminal width.
func (m Model) Width() int { return m.width }

// Height returns the current terminal height.
func (m Model) Height() int { return m.height }

// selectedGame returns the currently highlighted game, or nil.
func (m Model) selectedGame() game.Game {
	i := m.list.Index()
	if i >= 0 && i < len(m.games) {
		return m.games[i]
	}
	return nil
}

// selectedDescription returns the description of the currently highlighted game.
func (m Model) selectedDescription() string {
	g := m.selectedGame()
	if g == nil {
		return ""
	}
	return g.Description()
}

// sizeWarning returns a warning string if the current terminal is too small
// for the selected game, or an empty string otherwise.
func (m Model) sizeWarning() string {
	g := m.selectedGame()
	if g == nil {
		return ""
	}
	sc, ok := g.(game.SizeConstraint)
	if !ok {
		return ""
	}
	minW, minH := sc.MinSize()
	if m.width >= minW && m.height >= minH {
		return ""
	}
	return fmt.Sprintf("terminal too small - resize to at least %dx%d", minW, minH)
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
	title   := m.styles.Title.Render("arcade")
	tagline := m.styles.Muted.Render(appTagline)

	var desc string
	if w := m.sizeWarning(); w != "" {
		desc = m.styles.Error.Render(" " + w + " ")
	} else {
		desc = m.styles.Subtitle.Render(m.selectedDescription())
	}

	items := m.list.View()
	help  := m.styles.Muted.Render("↑/↓  navigate   enter  select   q  quit")

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

	if panel := m.leaderboard(); panel != "" {
		content = lipgloss.JoinHorizontal(lipgloss.Top, content, "    ", panel)
	}

	// Show a notice rather than letting Bubble Tea clip the lobby.
	if w, h := lipgloss.Size(content); w > m.width || h > m.height {
		return m.styles.TooSmall(m.width, m.height, w, h)
	}

	return lipgloss.Place(
		m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		content,
	)
}

// topN is how many players the lobby lists for each game.
const topN = 10

// leaderboard renders the highlighted game's best players and where this
// player stands, or "" if there is no leaderboard or the game isn't ranked.
func (m Model) leaderboard() string {
	g, ok := m.selectedGame().(game.Ranked)
	if !ok || !m.lb.Enabled() {
		return ""
	}
	id, r := game.ID(g), g.Ranking()

	lines := []string{m.styles.Subtitle.Render(leaderboard.Label(r)), ""}
	top := m.lb.Top(id, r, topN)
	if len(top) == 0 {
		lines = append(lines, m.styles.Muted.Render("no scores yet"))
	}
	for _, e := range top {
		row := fmt.Sprintf("%2d  %-12s %6s", e.Rank, e.Name, e.Value)
		if e.You {
			row = m.styles.Accent.Render(row)
		} else {
			row = m.styles.Subtitle.Render(row)
		}
		lines = append(lines, row)
	}
	lines = append(lines, "", m.standing(id, r))
	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

// standing says where the player stands on a game's board, or how to get on
// it.
func (m Model) standing(id string, r game.Ranking) string {
	switch {
	case m.lb.NeedsName():
		return m.styles.Muted.Render("reconnect to pick a name")
	case m.lb.Name() == "":
		return m.styles.Muted.Render("connect with an ssh key\nto be ranked")
	}
	rank, of := m.lb.Rank(id, r)
	if rank == 0 {
		return m.styles.Muted.Render("you: not ranked yet")
	}
	return m.styles.Accent.Render(fmt.Sprintf("you: #%d of %d", rank, of))
}
