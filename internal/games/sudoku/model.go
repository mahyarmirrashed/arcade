package sudoku

import (
	"math/rand"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/mahyarmirrashed/arcade/internal/game"
	"github.com/mahyarmirrashed/arcade/internal/ui"
)

// Compile-time checks.
var _ game.Game   = (*Model)(nil)
var _ game.Ranked = (*Model)(nil)

// cluesTarget is how many given cells to leave in the puzzle (medium difficulty).
const cluesTarget = 35

type gameState int

const (
	statePlaying gameState = iota
	stateWon
)

type keyMap struct {
	up    key.Binding
	down  key.Binding
	left  key.Binding
	right key.Binding
	clear key.Binding
	quit  key.Binding
}

var keys = keyMap{
	up:    key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
	down:  key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
	left:  key.NewBinding(key.WithKeys("left", "h"), key.WithHelp("←/h", "left")),
	right: key.NewBinding(key.WithKeys("right", "l"), key.WithHelp("→/l", "right")),
	clear: key.NewBinding(key.WithKeys("0", "backspace"), key.WithHelp("0/⌫", "clear")),
	quit:  key.NewBinding(key.WithKeys("esc", "ctrl+c"), key.WithHelp("esc", "quit")),
}

// Model is the Bubbletea model for Sudoku.
type Model struct {
	puzzle   grid // original puzzle clues (locked cells are non-zero)
	solution grid // the unique solution
	current  grid // player's working grid

	curR    int
	curC    int
	state   gameState
	started time.Time      // set on first digit placed
	now     func() time.Time // clock; replaced in tests

	width  int
	height int
	styles ui.Styles
}

// - game.Game implementation -

func (m *Model) Name() string        { return "Sudoku" }
func (m *Model) Description() string { return "fill the grid, no repeats" }

func (m *Model) New(width, height int, styles ui.Styles) tea.Model {
	return newModel(width, height, styles)
}

// Ranking implements game.Ranked.
func (m *Model) Ranking() game.Ranking { return game.ByTime }

func newModel(width, height int, styles ui.Styles) *Model {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	puzzle, solution := newPuzzle(rng, cluesTarget)
	return &Model{
		puzzle:   puzzle,
		solution: solution,
		current:  puzzle,
		width:    width,
		height:   height,
		styles:   styles,
		state:    statePlaying,
		now:      time.Now,
	}
}

// locked reports whether (r, c) is a given clue cell.
func (m *Model) locked(r, c int) bool { return m.puzzle[r][c] != 0 }

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

type restartMsg struct{ m *Model }

func (m *Model) handleKey(msg tea.KeyMsg) tea.Cmd {
	if msg.String() == "r" && m.state != statePlaying {
		fresh := newModel(m.width, m.height, m.styles)
		fresh.now = m.now
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
		if m.curR < size-1 {
			m.curR++
		}
	case key.Matches(msg, keys.left):
		if m.curC > 0 {
			m.curC--
		}
	case key.Matches(msg, keys.right):
		if m.curC < size-1 {
			m.curC++
		}
	case key.Matches(msg, keys.clear):
		return m.place(0)
	default:
		s := msg.String()
		if len(s) == 1 && s[0] >= '1' && s[0] <= '9' {
			return m.place(int(s[0] - '0'))
		}
	}
	return nil
}

// place sets the value at the cursor, validates, and checks for win.
func (m *Model) place(num int) tea.Cmd {
	if m.locked(m.curR, m.curC) {
		return nil
	}
	// Start the clock on the first digit placed.
	if m.started.IsZero() && num != 0 {
		m.started = m.now()
	}
	m.current[m.curR][m.curC] = num
	if num != 0 && isSolved(m.current) {
		m.state = stateWon
		return game.Finished(m, game.Result{Won: true, Duration: m.now().Sub(m.started)})
	}
	return nil
}

// - View -

// Box-drawing characters for the grid.
// All lines use the same weight; box boundaries are distinguished by colour.
const (
	segH = "───" // horizontal segment (3 chars wide to match cell width)
	segV = "│"   // vertical segment

	// Corners.
	cTL = "┌"
	cTR = "┐"
	cBL = "└"
	cBR = "┘"

	// T-junctions.
	jTop   = "┬"
	jBot   = "┴"
	jLeft  = "├"
	jRight = "┤"

	// Cross.
	jCross = "┼"
)

func (m *Model) View() string {
	help := m.renderHelp()
	board := m.renderBoard()

	title := m.styles.Title.Render("sudoku")
	boardWithTitle := lipgloss.JoinVertical(lipgloss.Left, title, "", board)

	pad := m.styles.NewStyle().PaddingBottom(1)
	layout := lipgloss.JoinHorizontal(lipgloss.Bottom,
		pad.Render(help), "  ", boardWithTitle,
	)

	if w, h := lipgloss.Size(layout); w > m.width || h > m.height {
		return m.styles.TooSmall(m.width, m.height, w, h)
	}

	return lipgloss.Place(m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		layout,
	)
}

func (m *Model) renderBoard() string {
	var sb strings.Builder

	sb.WriteString(m.topBorder())
	sb.WriteByte('\n')

	for r := 0; r < size; r++ {
		sb.WriteString(m.dataRow(r))
		sb.WriteByte('\n')

		if r == size-1 {
			sb.WriteString(m.bottomBorder())
		} else if (r+1)%3 == 0 {
			// Between box rows: thick separator.
			sb.WriteString(m.thickMidBorder())
			sb.WriteByte('\n')
		} else {
			// Within box: thin separator.
			sb.WriteString(m.thinMidBorder())
			sb.WriteByte('\n')
		}
	}

	inner := sb.String()

	if m.state == stateWon {
		w := lipgloss.Width(inner)
		h := lipgloss.Height(inner)
		inner = lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center,
			m.styles.Error.Render(" solved! "),
			lipgloss.WithWhitespaceChars(" "),
		)
	}

	return inner
}

// cellStyle returns the style for a cell at (r, c).
func (m *Model) cellStyle(r, c int) lipgloss.Style {
	base := m.styles.NewStyle().Width(3).Align(lipgloss.Center)

	isCursor := r == m.curR && c == m.curC
	isRelated := r == m.curR || c == m.curC || sameBox(r, c, m.curR, m.curC)
	cursorVal := m.current[m.curR][m.curC]
	isSameDigit := cursorVal != 0 && m.current[r][c] == cursorVal
	isConflict := m.current[r][c] != 0 && !m.locked(r, c) &&
		!isValidPlacement(zeroAt(m.current, r, c), r, c, m.current[r][c])

	switch {
	case isCursor:
		return base.Reverse(true).Bold(true)
	case isConflict:
		return base.Foreground(ui.Spectrum[0]) // red
	case isSameDigit:
		return base.Foreground(ui.ColorText).Bold(true)
	case isRelated:
		return base.Foreground(ui.ColorSubtle)
	default:
		return base.Foreground(ui.ColorMuted)
	}
}

// boxSt is the style for outer edges and box-boundary lines: bright white.
func (m *Model) boxSt() lipgloss.Style {
	return m.styles.NewStyle().Foreground(ui.ColorText)
}

// innerSt is the style for inner column/row separators within a 3x3 box.
func (m *Model) innerSt() lipgloss.Style {
	return m.styles.NewStyle().Foreground(ui.ColorSubtle)
}

func (m *Model) dataRow(r int) string {
	var sb strings.Builder
	for c := 0; c < size; c++ {
		if c%3 == 0 {
			sb.WriteString(m.boxSt().Render(segV))
		} else {
			sb.WriteString(m.innerSt().Render(segV))
		}

		val := m.current[r][c]
		st := m.cellStyle(r, c)
		if val == 0 {
			sb.WriteString(st.Render(" "))
		} else if m.locked(r, c) {
			sb.WriteString(st.Bold(true).Foreground(ui.ColorText).Render(string(rune('0' + val))))
		} else {
			sb.WriteString(st.Render(string(rune('0' + val))))
		}
	}
	sb.WriteString(m.boxSt().Render(segV))
	return sb.String()
}

// hBorder renders a full horizontal border line.
// isBox=true means this is an outer edge or box-boundary row (white);
// false means it is an inner row within a box (muted).
func (m *Model) hBorder(left, right, boxMid, innerMid string, isBox bool) string {
	var sb strings.Builder
	lineSt := m.boxSt()
	if !isBox {
		lineSt = m.innerSt()
	}
	sb.WriteString(m.boxSt().Render(left))
	for c := 0; c < size; c++ {
		sb.WriteString(lineSt.Render(segH))
		if c == size-1 {
			sb.WriteString(m.boxSt().Render(right))
		} else if (c+1)%3 == 0 {
			sb.WriteString(m.boxSt().Render(boxMid))
		} else {
			sb.WriteString(lineSt.Render(innerMid))
		}
	}
	return sb.String()
}

func (m *Model) topBorder() string {
	return m.hBorder(cTL, cTR, jTop, jTop, true)
}

func (m *Model) bottomBorder() string {
	return m.hBorder(cBL, cBR, jBot, jBot, true)
}

func (m *Model) thickMidBorder() string {
	return m.hBorder(jLeft, jRight, jCross, jCross, true)
}

func (m *Model) thinMidBorder() string {
	return m.hBorder(jLeft, jRight, jCross, jCross, false)
}

func (m *Model) renderHelp() string {
	line := func(s string) string { return m.styles.Muted.Render(s) }
	return lipgloss.JoinVertical(lipgloss.Left,
		line("↑↓←→  move"),
		line("1-9   place"),
		line("0/⌫   clear"),
		line("r     new game"),
		line("esc   quit"),
	)
}
