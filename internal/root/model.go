// Package root provides the top-level Bubbletea model for the arcade.
// It owns the lobby↔game state machine for a single SSH session.
package root

import (
	"strings"

	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"

	"github.com/mahyarmirrashed/arcade/internal/game"
	"github.com/mahyarmirrashed/arcade/internal/leaderboard"
	"github.com/mahyarmirrashed/arcade/internal/lobby"
	"github.com/mahyarmirrashed/arcade/internal/ui"
)

type state int

const (
	stateLobby state = iota
	statePlaying
	stateName // choosing a leaderboard name, before the lobby
)

// Model is the root Bubbletea model. It holds either the lobby or an active
// game and delegates all messages to whichever is current.
type Model struct {
	state  state
	width  int
	height int
	lobby  lobby.Model
	active tea.Model // non-nil only while statePlaying
	styles ui.Styles
	lb     leaderboard.Session

	name    textinput.Model // the name prompt, while stateName
	nameErr string          // why the last name was refused
}

// New returns an initialised root Model. styles render for the session's
// terminal and are handed to the lobby and to every game started from it. lb
// is the player's leaderboard session; a player with a key but no name is
// asked for one before the lobby.
func New(width, height int, reg *game.Registry, styles ui.Styles, lb leaderboard.Session) Model {
	m := Model{
		state:  stateLobby,
		width:  width,
		height: height,
		lobby:  lobby.New(width, height, reg, styles, lb),
		styles: styles,
		lb:     lb,
	}
	if lb.NeedsName() {
		m.state = stateName
		m.name = newNameInput(styles)
	}
	return m
}

func newNameInput(styles ui.Styles) textinput.Model {
	ti := textinput.New()
	ti.Placeholder = "your name"
	ti.CharLimit = 12
	ti.PromptStyle = styles.Muted
	ti.TextStyle = styles.Accent
	ti.PlaceholderStyle = styles.Muted
	ti.Cursor.Style = styles.NewStyle()
	ti.Cursor.TextStyle = styles.NewStyle()
	ti.Cursor.SetMode(cursor.CursorStatic)
	ti.Focus()
	return ti
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

		// Bubble Tea redraws on resize but doesn't clear, so remnants of the
		// old layout could stay on screen.
		next, cmd := m.update(msg)
		return next, tea.Batch(tea.ClearScreen, cmd)
	}
	return m.update(msg)
}

// update forwards msg to whichever screen is active.
func (m Model) update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Results are recorded whatever the state: a game can finish in the same
	// instant the player quits, so its result may arrive in the lobby.
	if res, ok := msg.(game.ResultMsg); ok {
		return m, m.record(res)
	}

	switch m.state {
	case stateName:
		return m.updateName(msg)
	case stateLobby:
		return m.updateLobby(msg)
	case statePlaying:
		return m.updatePlaying(msg)
	}
	return m, nil
}

func (m Model) updateLobby(msg tea.Msg) (tea.Model, tea.Cmd) {
	// esc / ctrl+c in the lobby quits the program.
	if km, ok := msg.(tea.KeyMsg); ok {
		switch km.String() {
		case "esc", "ctrl+c":
			return m, tea.Quit
		}
	}

	// Intercept SelectMsg before forwarding to lobby so we can transition.
	if sel, ok := msg.(lobby.SelectMsg); ok {
		g := sel.Game.New(m.width, m.height, m.styles)
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
	case stateName:
		return m.viewName()
	case statePlaying:
		return m.active.View()
	default:
		return m.lobby.View()
	}
}

// State returns the current state (exported for tests).
func (m Model) State() state { return m.state }

// StateLobby, StatePlaying and StateName are the exported state constants
// for tests.
const (
	StateLobby   = stateLobby
	StatePlaying = statePlaying
	StateName    = stateName
)

// record saves a finished game's result and, if the player is still in the
// game, tells it where the result placed.
func (m Model) record(res game.ResultMsg) tea.Cmd {
	standing, err := m.lb.Record(res.Game, res.Ranking, res.Result)
	if err != nil {
		// The result is kept in memory and saved with the next change.
		log.Error("saving leaderboard", "error", err)
	}
	if m.state != statePlaying {
		return nil
	}
	return func() tea.Msg { return game.StandingMsg{Standing: standing} }
}

func (m Model) updateName(msg tea.Msg) (tea.Model, tea.Cmd) {
	if km, ok := msg.(tea.KeyMsg); ok {
		// Every other key, q included, is typing.
		switch km.Type {
		case tea.KeyCtrlC:
			return m, tea.Quit
		case tea.KeyEsc:
			// Skip for now: play unranked and be asked again next time.
			m.state = stateLobby
			return m, nil
		case tea.KeyEnter:
			if err := m.lb.SetName(strings.TrimSpace(m.name.Value())); err != nil {
				m.nameErr = err.Error()
				return m, nil
			}
			m.state = stateLobby
			return m, nil
		}
		m.nameErr = ""
	}

	var cmd tea.Cmd
	m.name, cmd = m.name.Update(msg)
	return m, cmd
}

func (m Model) viewName() string {
	hint := m.styles.Muted.Render("this is how you'll appear on the leaderboards")
	if m.nameErr != "" {
		hint = m.styles.Error.Render(" " + m.nameErr + " ")
	}
	content := lipgloss.JoinVertical(lipgloss.Left,
		m.styles.Title.Render("arcade"),
		"",
		m.styles.Subtitle.Render("choose a name"),
		"",
		m.name.View(),
		"",
		hint,
		"",
		m.styles.Muted.Render("enter to save · esc to skip"),
	)
	if w, h := lipgloss.Size(content); w > m.width || h > m.height {
		return m.styles.TooSmall(m.width, m.height, w, h)
	}
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
}
