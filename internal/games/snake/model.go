// Package snake implements the Snake game for the arcade.
package snake

import (
	"math/rand/v2"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/mahyarmirrashed/arcade/internal/game"
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
}

// newModel returns a game sized to the terminal, drawing food positions from rng.
func newModel(width, height int, rng *rand.Rand) *Model {
	m := &Model{rng: rng}
	m.resize(width, height)
	return m
}

// resize records the terminal size and starts a new game that fits it.
func (m *Model) resize(width, height int) {
	m.width, m.height = width, height
	m.reset()
}

// reset starts a new game, invalidating any ticks still in flight.
func (m *Model) reset() {
	m.gen++
	m.paused = false
	cols, rows, ok := gridSize(m.width, m.height)
	m.tooSmall = !ok
	if ok {
		m.board = newBoard(cols, rows, m.rng)
	}
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

// — game.Game implementation —

// Name implements game.Game.
func (m *Model) Name() string { return "Snake" }

// Description implements game.Game.
func (m *Model) Description() string { return "eat, grow, don't bite yourself" }

// New implements game.Game — returns a fresh model for a new session.
func (m *Model) New(width, height int) tea.Model {
	return newModel(width, height, rand.New(rand.NewPCG(rand.Uint64(), rand.Uint64())))
}

// — tea.Model implementation —

// Init implements tea.Model.
func (m *Model) Init() tea.Cmd { return m.tick() }

// Update implements tea.Model.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		if msg.Width == m.width && msg.Height == m.height {
			return m, nil
		}
		m.resize(msg.Width, msg.Height)
		return m, m.tick()
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
