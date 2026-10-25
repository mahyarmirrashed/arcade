// Package snake implements the Snake game for the arcade.
package snake

import (
	"math/rand/v2"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/mahyarmirrashed/arcade/internal/game"
	"github.com/mahyarmirrashed/arcade/internal/ui"
)

// Compile-time check: Model must satisfy game.Game.
var _ game.Game = (*Model)(nil)

// tickMsg advances the game one step. owner and gen tie it to the game that
// scheduled it, so ticks left over from an earlier session or from before a
// reset are dropped instead of running a second tick chain.
type tickMsg struct {
	owner *Model
	gen   int
}

// Model is the Bubbletea model for Snake.
type Model struct {
	width    int
	height   int
	board    board
	rng      *rand.Rand
	gen      int
	tooSmall bool
	paused   bool
	styles   ui.Styles
}

// newModel returns a game sized to the terminal, drawing food positions from
// rng and rendering with styles.
func newModel(width, height int, rng *rand.Rand, styles ui.Styles) *Model {
	m := &Model{rng: rng, styles: styles}
	m.resize(width, height)
	return m
}

// resize records the terminal size. A game in progress keeps its board; if
// the board no longer fits, the game pauses until the terminal is big enough
// again. With no game yet, one starts as soon as the terminal fits.
func (m *Model) resize(width, height int) tea.Cmd {
	m.width, m.height = width, height
	if !m.hasGame() {
		m.reset()
		return m.tick()
	}
	m.tooSmall = !m.boardFits()
	if m.tooSmall && m.board.alive && !m.board.won && !m.paused {
		// Pause so the snake doesn't move while the player can't see it.
		m.paused = true
		m.gen++
	}
	return nil
}

// reset starts a new game sized to the terminal, invalidating any ticks still
// in flight. If the terminal is too small there is no game until it grows.
func (m *Model) reset() {
	m.gen++
	m.paused = false
	cols, rows, ok := gridSize(m.width, m.height)
	m.tooSmall = !ok
	m.board = board{}
	if ok {
		m.board = newBoard(cols, rows, m.rng)
	}
}

func (m *Model) hasGame() bool { return len(m.board.body) > 0 }

// boardFits reports whether the current board, with the panels around it,
// fits in the terminal.
func (m *Model) boardFits() bool {
	return m.width >= m.board.cols*cellWidth+chromeWidth() &&
		m.height >= m.board.rows+titleH+borderH
}

func (m *Model) playing() bool {
	return !m.tooSmall && m.board.alive && !m.board.won
}

// tick schedules the next step for the current game.
func (m *Model) tick() tea.Cmd {
	if !m.playing() || m.paused {
		return nil
	}
	msg := tickMsg{owner: m, gen: m.gen}
	return tea.Tick(interval(m.board.score), func(time.Time) tea.Msg {
		return msg
	})
}

// - game.Game implementation -

// Name implements game.Game.
func (m *Model) Name() string { return "Snake" }

// Description implements game.Game.
func (m *Model) Description() string { return "eat, grow, don't bite yourself" }

// MinSize implements game.SizeConstraint.
func (m *Model) MinSize() (int, int) {
	minW := minCols*cellWidth + chromeWidth()
	minH := minRows + titleH + borderH
	return minW, minH
}

// New implements game.Game - returns a fresh model for a new session.
func (m *Model) New(width, height int, styles ui.Styles) tea.Model {
	return newModel(width, height, rand.New(rand.NewPCG(rand.Uint64(), rand.Uint64())), styles)
}

// - tea.Model implementation -

// Init implements tea.Model.
func (m *Model) Init() tea.Cmd { return m.tick() }

// Update implements tea.Model.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		if msg.Width == m.width && msg.Height == m.height {
			return m, nil
		}
		return m, m.resize(msg.Width, msg.Height)
	case tickMsg:
		if msg.owner != m || msg.gen != m.gen || !m.playing() || m.paused {
			return m, nil
		}
		m.board.step(m.rng)
		return m, m.tick()
	case tea.KeyMsg:
		return m, m.handleKey(msg)
	}
	return m, nil
}

// handleKey applies a key press. q and ctrl+c are intentionally not handled:
// the root model intercepts them and returns the player to the lobby.
func (m *Model) handleKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "r":
		if !m.tooSmall && !m.playing() {
			m.reset()
			return m.tick()
		}
	case "p", " ":
		if m.playing() {
			// Pausing or resuming invalidates the tick in flight, so a quick
			// pause and resume can't leave two tick chains running.
			m.paused = !m.paused
			m.gen++
			return m.tick()
		}
	case "up", "k":
		m.steer(up)
	case "down", "j":
		m.steer(down)
	case "left", "h":
		m.steer(left)
	case "right", "l":
		m.steer(right)
	}
	return nil
}

func (m *Model) steer(d direction) {
	if !m.paused {
		m.board.turn(d)
	}
}
