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

// Board layout. Each cell is drawn two characters wide so it looks square in
// a terminal, where characters are about twice as tall as they are wide.
const (
	cellWidth = 2
	borderW   = 2 // left and right border columns
	borderH   = 2 // top and bottom border rows
	headerH   = 1
	footerH   = 1

	minCols, minRows = 10, 10
	// The board is capped so huge terminals don't make the game sluggish.
	maxCols, maxRows = 40, 20
)

// gridSize returns how many cells fit in a terminal of the given size, capped
// at maxCols x maxRows. ok is false when the terminal is too small to play.
func gridSize(termW, termH int) (cols, rows int, ok bool) {
	cols = min((termW-borderW)/cellWidth, maxCols)
	rows = min(termH-borderH-headerH-footerH, maxRows)
	return cols, rows, cols >= minCols && rows >= minRows
}

// tickMsg advances the game one step. gen ties it to the game that scheduled
// it, so ticks left over from before a reset are dropped.
type tickMsg struct{ gen int }

// Model is the Bubbletea model for Snake.
type Model struct {
	width    int
	height   int
	board    board
	rng      *rand.Rand
	gen      int
	tooSmall bool
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
	if !m.playing() {
		return nil
	}
	gen := m.gen
	return tea.Tick(interval(m.board.score), func(time.Time) tea.Msg {
		return tickMsg{gen: gen}
	})
}

// quit ends the session. It is the one place to change when Snake should
// return to the lobby instead.
func quit() tea.Cmd { return tea.Quit }

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
		if msg.gen != m.gen || !m.playing() {
			return m, nil
		}
		m.board.step(m.rng)
		return m, m.tick()
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, quit()
		case "up", "k":
			m.board.turn(up)
		case "down", "j":
			m.board.turn(down)
		case "left", "h":
			m.board.turn(left)
		case "right", "l":
			m.board.turn(right)
		}
	}
	return m, nil
}
