package minesweeper

import (
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/mahyarmirrashed/arcade/internal/game"
	"github.com/mahyarmirrashed/arcade/internal/ui"
)

// Compile-time check: Model must satisfy game.Game.
var _ game.Game = (*Model)(nil)

type gameState int

const (
	statePlaying gameState = iota
	stateWon
	stateLost
)

type keyMap struct {
	up    key.Binding
	down  key.Binding
	left  key.Binding
	right key.Binding
	reveal key.Binding
	flag   key.Binding
	quit   key.Binding
}

var keys = keyMap{
	up:     key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
	down:   key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
	left:   key.NewBinding(key.WithKeys("left", "h"), key.WithHelp("←/h", "left")),
	right:  key.NewBinding(key.WithKeys("right", "l"), key.WithHelp("→/l", "right")),
	reveal: key.NewBinding(key.WithKeys("enter", " "), key.WithHelp("enter/spc", "reveal")),
	flag:   key.NewBinding(key.WithKeys("f"), key.WithHelp("f", "flag")),
	quit:   key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
}

// Model is the Bubbletea model for Minesweeper.
type Model struct {
	board  board
	curR   int
	curC   int
	state  gameState
	width  int
	height int
	rng    *rand.Rand
	styles ui.Styles
}

// - game.Game implementation -

func (m *Model) Name() string        { return "Minesweeper" }
func (m *Model) Description() string { return "clear the minefield" }

func (m *Model) New(width, height int, styles ui.Styles) tea.Model {
	return newModel(width, height, styles)
}

func newModel(width, height int, styles ui.Styles) *Model {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	b := newBoard(defaultCols, defaultRows, defaultMines)
	return &Model{
		board:  b,
		curR:   defaultRows / 2,
		curC:   defaultCols / 2,
		state:  statePlaying,
		width:  width,
		height: height,
		rng:    rng,
		styles: styles,
	}
}

// - tea.Model implementation -

func (m *Model) Init() tea.Cmd { return nil }

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if rm, ok := msg.(restartMsg); ok {
		return rm.m, nil
	}
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyMsg:
		return m, m.handleKey(msg)
	}
	return m, nil
}

func (m *Model) handleKey(msg tea.KeyMsg) tea.Cmd {
	if msg.String() == "r" && m.state != statePlaying {
		fresh := newModel(m.width, m.height, m.styles)
		return func() tea.Msg { return restartMsg{fresh} }
	}

	if m.state != statePlaying {
		return nil
	}

	switch {
	case key.Matches(msg, keys.up):
		if m.curR > 0 {
			m.curR--
		}
	case key.Matches(msg, keys.down):
		if m.curR < m.board.rows-1 {
			m.curR++
		}
	case key.Matches(msg, keys.left):
		if m.curC > 0 {
			m.curC--
		}
	case key.Matches(msg, keys.right):
		if m.curC < m.board.cols-1 {
			m.curC++
		}
	case key.Matches(msg, keys.reveal):
		if m.board.reveal(m.curR, m.curC, m.rng) {
			m.board.revealAll()
			m.state = stateLost
		} else if m.board.hasWon() {
			m.state = stateWon
		}
	case key.Matches(msg, keys.flag):
		m.board.toggleFlag(m.curR, m.curC)
	}
	return nil
}

// restartMsg replaces the model with a fresh one.
// Update intercepts it before dispatching to handleKey.
type restartMsg struct{ m *Model }

// - View -

// neighbourColor maps neighbour counts to colours matching standard Minesweeper convention.
var neighbourColors = [9]lipgloss.Color{
	0: lipgloss.Color("0"),   // blank (unused)
	1: lipgloss.Color("21"),  // blue
	2: lipgloss.Color("34"),  // green
	3: lipgloss.Color("196"), // red
	4: lipgloss.Color("19"),  // dark blue
	5: lipgloss.Color("88"),  // dark red
	6: lipgloss.Color("30"),  // teal
	7: lipgloss.Color("0"),   // black
	8: lipgloss.Color("245"), // grey
}

func (m *Model) View() string {
	// Intercept restartMsg here too.
	help := m.renderHelp()
	boardStr := m.renderBoard()

	title := m.styles.Title.Render("minesweeper")
	boardWithTitle := lipgloss.JoinVertical(lipgloss.Left, title, "", boardStr)

	pad := m.styles.NewStyle().PaddingBottom(1)
	layout := lipgloss.JoinHorizontal(lipgloss.Bottom,
		pad.Render(help), "  ", boardWithTitle, "  ", pad.Render(m.renderSide()),
	)

	return lipgloss.Place(m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		layout,
	)
}

func (m *Model) renderBoard() string {
	var sb strings.Builder
	for r := 0; r < m.board.rows; r++ {
		for c := 0; c < m.board.cols; c++ {
			sb.WriteString(m.renderCell(r, c))
		}
		if r < m.board.rows-1 {
			sb.WriteByte('\n')
		}
	}

	inner := sb.String()

	switch m.state {
	case stateWon:
		w := lipgloss.Width(inner)
		h := lipgloss.Height(inner)
		inner = lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center,
			m.styles.Error.Render(" you win! "),
			lipgloss.WithWhitespaceChars(" "),
		)
	case stateLost:
		w := lipgloss.Width(inner)
		h := lipgloss.Height(inner)
		inner = lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center,
			m.styles.Error.Render(" boom! "),
			lipgloss.WithWhitespaceChars(" "),
		)
	}

	return m.styles.Border.Render(inner)
}

func (m *Model) renderCell(r, c int) string {
	cl := m.board.cells[r][c]
	cursor := r == m.curR && c == m.curC

	base := m.styles.NewStyle().Width(2).Align(lipgloss.Center)
	if cursor {
		base = base.Reverse(true)
	}

	switch cl.state {
	case cellFlagged:
		return base.Foreground(ui.Spectrum[0]).Render("F")
	case cellHidden:
		return base.Foreground(ui.ColorMuted).Render(".")
	case cellRevealed:
		if cl.mine {
			return base.Foreground(ui.Spectrum[0]).Render("*")
		}
		if cl.neighbours == 0 {
			return base.Foreground(ui.ColorMuted).Render(" ")
		}
		color := neighbourColors[cl.neighbours]
		return base.Foreground(color).Bold(true).Render(strconv.Itoa(cl.neighbours))
	}
	return base.Render(" ")
}

func (m *Model) renderSide() string {
	remaining := m.board.mines - m.board.flagged
	return lipgloss.JoinVertical(lipgloss.Left,
		m.styles.Subtitle.Render("mines"),
		m.styles.Accent.Render(strconv.Itoa(remaining)),
		"",
		m.styles.Subtitle.Render("flags"),
		m.styles.Accent.Render(strconv.Itoa(m.board.flagged)),
	)
}

func (m *Model) renderHelp() string {
	line := func(s string) string { return m.styles.Muted.Render(s) }
	return lipgloss.JoinVertical(lipgloss.Left,
		line("↑↓←→  move"),
		line("enter  reveal"),
		line("f      flag"),
		line("r      restart"),
		line("q      quit"),
	)
}
