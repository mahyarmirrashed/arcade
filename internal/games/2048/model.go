package _2048

import (
	"math/rand"
	"strconv"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/mahyarmirrashed/arcade/internal/game"
	"github.com/mahyarmirrashed/arcade/internal/ui"
)

// Compile-time check: Model must satisfy game.Game.
var _ game.Game = (*Model)(nil)

// keyMap defines the 2048 keybindings.
type keyMap struct {
	up    key.Binding
	down  key.Binding
	left  key.Binding
	right key.Binding
	quit  key.Binding
}

var keys = keyMap{
	up:    key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
	down:  key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
	left:  key.NewBinding(key.WithKeys("left", "h"), key.WithHelp("←/h", "left")),
	right: key.NewBinding(key.WithKeys("right", "l"), key.WithHelp("→/l", "right")),
	quit:  key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
}

// state tracks game phase.
type state int

const (
	statePlaying state = iota
	stateWon
	stateLost
)

// Model is the Bubbletea model for 2048.
type Model struct {
	g      grid
	score  int
	best   int
	state  state
	width  int
	height int
	rng    *rand.Rand
	styles ui.Styles
}

// - game.Game implementation -

func (m *Model) Name() string        { return "2048" }
func (m *Model) Description() string { return "slide tiles, reach 2048" }

func (m *Model) New(width, height int, styles ui.Styles) tea.Model {
	return newModel(width, height, styles)
}

func newModel(width, height int, styles ui.Styles) *Model {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	var g grid
	g, _ = spawn(g, rng)
	g, _ = spawn(g, rng)
	return &Model{
		g:      g,
		width:  width,
		height: height,
		rng:    rng,
		state:  statePlaying,
		styles: styles,
	}
}

// - tea.Model implementation -

func (m *Model) Init() tea.Cmd { return nil }

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if km, ok := msg.(tea.KeyMsg); ok {
		// r restarts from any state.
		if km.String() == "r" {
			fresh := newModel(m.width, m.height, m.styles)
			fresh.best = m.best
			return fresh, nil
		}
	}

	if ws, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = ws.Width
		m.height = ws.Height
		return m, nil
	}

	if m.state != statePlaying {
		return m, nil
	}

	km, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	// Hold the board while the player can't see it.
	if !m.fits() {
		return m, nil
	}

	var d direction
	switch {
	case key.Matches(km, keys.up):
		d = dirUp
	case key.Matches(km, keys.down):
		d = dirDown
	case key.Matches(km, keys.left):
		d = dirLeft
	case key.Matches(km, keys.right):
		d = dirRight
	default:
		return m, nil
	}

	newG, points, moved := slide(m.g, d)
	if !moved {
		return m, nil
	}

	m.g = newG
	m.score += points
	if m.score > m.best {
		m.best = m.score
	}

	if hasWon(m.g) {
		m.state = stateWon
		return m, nil
	}

	m.g, _ = spawn(m.g, m.rng)

	if !hasMovesLeft(m.g) {
		m.state = stateLost
	}

	return m, nil
}

// - View -

// tileStyle returns a Lip Gloss style for the given tile value.
// Higher tiles get brighter colours from ui.Spectrum.
func (m *Model) tileStyle(value int) lipgloss.Style {
	base := m.styles.NewStyle().
		Width(6).
		Align(lipgloss.Center).
		Bold(true)

	if value == 0 {
		return base.Foreground(ui.ColorMuted)
	}

	// Map powers of 2 (2=0, 4=1, ... 2048=10) to spectrum colours.
	idx := 0
	v := value
	for v > 2 {
		v >>= 1
		idx++
	}
	if idx >= len(ui.Spectrum) {
		idx = len(ui.Spectrum) - 1
	}
	return base.Foreground(ui.Spectrum[idx])
}

func (m *Model) View() string {
	layout := m.layout()
	if w, h := lipgloss.Size(layout); w > m.width || h > m.height {
		return m.styles.TooSmall(m.width, m.height, w, h)
	}
	return lipgloss.Place(m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		layout,
	)
}

// fits reports whether the whole layout fits in the terminal.
func (m *Model) fits() bool {
	w, h := lipgloss.Size(m.layout())
	return w <= m.width && h <= m.height
}

// layout renders the help beside the titled board.
func (m *Model) layout() string {
	help := m.renderHelp()
	board := m.renderBoard()

	title := m.styles.Title.Render("2048")
	boardWithTitle := lipgloss.JoinVertical(lipgloss.Left, title, "", board)

	pad := m.styles.NewStyle().PaddingBottom(1)
	layout := lipgloss.JoinHorizontal(lipgloss.Bottom,
		pad.Render(help), "  ", boardWithTitle,
	)

	return layout
}

func (m *Model) renderBoard() string {
	var rows [size]string
	for r := 0; r < size; r++ {
		var cells [size]string
		for c := 0; c < size; c++ {
			v := m.g[r][c]
			s := m.tileStyle(v)
			if v == 0 {
				cells[c] = s.Render("·")
			} else {
				cells[c] = s.Render(strconv.Itoa(v))
			}
		}
		rows[r] = lipgloss.JoinHorizontal(lipgloss.Center,
			cells[0], cells[1], cells[2], cells[3],
		)
	}

	inner := lipgloss.JoinVertical(lipgloss.Left,
		rows[0], rows[1], rows[2], rows[3],
	)

	// Overlay for game-over / win states.
	if m.state != statePlaying {
		var msg string
		if m.state == stateWon {
			msg = " you win! "
		} else {
			msg = " game over "
		}
		w := lipgloss.Width(inner)
		h := lipgloss.Height(inner)
		inner = lipgloss.Place(w, h,
			lipgloss.Center, lipgloss.Center,
			m.styles.Error.Render(msg),
			lipgloss.WithWhitespaceChars(" "),
		)
	}

	stats := lipgloss.JoinHorizontal(lipgloss.Top,
		lipgloss.JoinVertical(lipgloss.Left,
			m.styles.Subtitle.Render("score"),
			strconv.Itoa(m.score),
		),
		"   ",
		lipgloss.JoinVertical(lipgloss.Left,
			m.styles.Subtitle.Render("best"),
			strconv.Itoa(m.best),
		),
	)

	return m.styles.Border.Render(
		lipgloss.JoinVertical(lipgloss.Left,
			stats,
			"",
			inner,
		),
	)
}

func (m *Model) renderHelp() string {
	line := func(s string) string { return m.styles.Muted.Render(s) }
	return lipgloss.JoinVertical(lipgloss.Left,
		line("↑/k  up"),
		line("↓/j  down"),
		line("←/h  left"),
		line("→/l  right"),
		line("r    restart"),
		line("q    quit"),
	)
}
