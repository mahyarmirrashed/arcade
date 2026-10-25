// Package tetris implements the Tetris game for the arcade.
package tetris

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

// tickMsg is sent on every gravity tick. owner ties it to the game that
// scheduled it, so a tick left over from an earlier game is dropped instead
// of starting a second gravity loop.
type tickMsg struct{ owner *Model }

// tickCmd returns a command that fires after the appropriate delay for m's level.
func tickCmd(m *Model) tea.Cmd {
	delay := time.Duration(max(50, 1000-m.level*100)) * time.Millisecond
	return tea.Tick(delay, func(time.Time) tea.Msg { return tickMsg{owner: m} })
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// keys defines all Tetris keybindings.
type keyMap struct {
	left      key.Binding
	right     key.Binding
	rotateCW  key.Binding
	rotateCCW key.Binding
	softDrop  key.Binding
	hardDrop  key.Binding
	hold      key.Binding
	quit      key.Binding
}

var keys = keyMap{
	left:      key.NewBinding(key.WithKeys("left", "h"), key.WithHelp("←/h", "move left")),
	right:     key.NewBinding(key.WithKeys("right", "l"), key.WithHelp("→/l", "move right")),
	rotateCW:  key.NewBinding(key.WithKeys("up", "x"), key.WithHelp("↑/x", "rotate CW")),
	rotateCCW: key.NewBinding(key.WithKeys("z"), key.WithHelp("z", "rotate CCW")),
	softDrop:  key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "soft drop")),
	hardDrop:  key.NewBinding(key.WithKeys(" "), key.WithHelp("space", "hard drop")),
	hold:      key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "hold")),
	quit:      key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
}

// piece tracks the active falling piece.
type piece struct {
	pt      PieceType
	rot     int
	row     int // top-left row on board
	col     int // top-left col on board
}

func (p piece) shape() Shape {
	return Rotation(p.pt, p.rot)
}

// Model is the Bubbletea model for Tetris.
type Model struct {
	board     board
	current   piece
	next      PieceType
	hold      *PieceType // nil = no piece held yet
	canHold   bool       // reset to true each time a piece locks
	score     int
	level     int
	lines     int
	alive     bool
	started   bool
	width     int
	height    int
	rng       *rand.Rand
}

// — game.Game implementation —

// Name implements game.Game.
func (m *Model) Name() string { return "Tetris" }

// Description implements game.Game.
func (m *Model) Description() string { return "clear lines, survive gravity" }

// New implements game.Game — returns a fresh model for a new session.
func (m *Model) New(width, height int) tea.Model {
	return newModel(width, height)
}

// newModel constructs a ready-to-play Model.
func newModel(width, height int) *Model {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	m := &Model{
		width:   width,
		height:  height,
		level:   1,
		alive:   true,
		canHold: true,
		rng:     rng,
		next:    PieceType(rng.Intn(7)),
	}
	m.spawn()
	return m
}

// spawn places the next piece at the top of the board and draws a new next.
func (m *Model) spawn() {
	pt := m.next
	m.next = PieceType(m.rng.Intn(7))
	col := boardCols/2 - 2
	p := piece{pt: pt, rot: 0, row: 0, col: col}
	if collides(m.board, p.shape(), p.row, p.col) {
		m.alive = false
		return
	}
	m.current = p
}

// — tea.Model implementation —

// Init implements tea.Model.
func (m *Model) Init() tea.Cmd {
	m.started = true
	return tickCmd(m)
}

// Update implements tea.Model.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if !m.alive {
		if km, ok := msg.(tea.KeyMsg); ok {
			switch km.String() {
			case "r":
				fresh := newModel(m.width, m.height)
				return fresh, fresh.Init()
			}
		}
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tickMsg:
		if msg.owner != m {
			return m, nil
		}
		m.gravity()
		return m, tickCmd(m)

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keys.left):
			m.tryMove(0, -1)
		case key.Matches(msg, keys.right):
			m.tryMove(0, 1)
		case key.Matches(msg, keys.rotateCW):
			m.tryRotate(1)
		case key.Matches(msg, keys.rotateCCW):
			m.tryRotate(-1)
		case key.Matches(msg, keys.softDrop):
			m.gravity()
		case key.Matches(msg, keys.hardDrop):
			m.hardDrop()
		case key.Matches(msg, keys.hold):
			m.holdPiece()
		}
	}

	return m, nil
}

// gravity advances the current piece one row. If it can't move, it locks.
func (m *Model) gravity() {
	if !collides(m.board, m.current.shape(), m.current.row+1, m.current.col) {
		m.current.row++
		return
	}
	m.lock()
}

// hardDrop drops the piece to the lowest valid position immediately.
func (m *Model) hardDrop() {
	for !collides(m.board, m.current.shape(), m.current.row+1, m.current.col) {
		m.current.row++
	}
	m.lock()
}

// lock stamps the current piece onto the board, clears lines, and spawns next.
func (m *Model) lock() {
	m.board = place(m.board, m.current.shape(), m.current.row, m.current.col, m.current.pt)
	var cleared int
	m.board, cleared = clearLines(m.board)
	if cleared > 0 {
		m.score += score(cleared, m.level)
		m.lines += cleared
		m.level = m.lines/10 + 1
	}
	m.canHold = true // allow hold again after locking
	m.spawn()
}

// holdPiece swaps the current piece with the held piece.
// If nothing is held yet, the current piece is stashed and the next spawns.
// Can only be used once per piece — resets when a piece locks.
func (m *Model) holdPiece() {
	if !m.canHold {
		return
	}
	m.canHold = false

	if m.hold == nil {
		// No held piece yet — stash current and spawn next.
		pt := m.current.pt
		m.hold = &pt
		m.spawn()
	} else {
		// Swap current with held. Piece always re-enters at rotation 0.
		pt := m.current.pt
		m.current = piece{
			pt:  *m.hold,
			rot: 0,
			row: 0,
			col: boardCols/2 - 2,
		}
		m.hold = &pt
		// If the swap would collide (board too full), game over.
		if collides(m.board, m.current.shape(), m.current.row, m.current.col) {
			m.alive = false
		}
	}
}

// tryMove attempts to shift the current piece by (dRow, dCol).
func (m *Model) tryMove(dRow, dCol int) {
	nr := m.current.row + dRow
	nc := m.current.col + dCol
	if !collides(m.board, m.current.shape(), nr, nc) {
		m.current.row = nr
		m.current.col = nc
	}
}

// tryRotate attempts a CW (dir=1) or CCW (dir=-1) rotation with SRS wall-kicks.
func (m *Model) tryRotate(dir int) {
	fromRot := m.current.rot
	toRot := (fromRot + dir + 4) % 4
	newShape := Rotation(m.current.pt, toRot)

	offsets := WallKickOffsets(m.current.pt, fromRot, toRot)
	if len(offsets) == 0 {
		// O piece or no table entry — attempt plain rotation.
		offsets = [][2]int{{0, 0}}
	}

	for _, off := range offsets {
		nc := m.current.col + off[0]
		nr := m.current.row + off[1]
		if !collides(m.board, newShape, nr, nc) {
			m.current.rot = toRot
			m.current.row = nr
			m.current.col = nc
			return
		}
	}
}

// ghostRow returns the row where the current piece would land if dropped.
func (m *Model) ghostRow() int {
	row := m.current.row
	for !collides(m.board, m.current.shape(), row+1, m.current.col) {
		row++
	}
	return row
}

// — View —

const (
	cellFull  = "██"
	cellEmpty = "  "
	cellGhost = "░░"
)

// pieceColor maps each piece type to a colour.
var pieceColor = [7]lipgloss.Color{
	PieceI: lipgloss.Color("51"),  // cyan
	PieceO: lipgloss.Color("226"), // yellow
	PieceT: lipgloss.Color("129"), // purple
	PieceS: lipgloss.Color("46"),  // green
	PieceZ: lipgloss.Color("196"), // red
	PieceJ: lipgloss.Color("21"),  // blue
	PieceL: lipgloss.Color("208"), // orange
}

// View implements tea.Model.
func (m *Model) View() string {
	// The board is wrapped in a border (+1 row top, +1 row bottom).
	// Pad help and side by 1 at the bottom so their content aligns with the
	// board's inner content rather than the border's bottom edge.
	pad := lipgloss.NewStyle().PaddingBottom(1)

	help  := pad.Render(m.renderHelp())
	board := m.renderBoard()
	side  := pad.Render(m.renderSide())

	title := ui.Title.Render("tetris")
	boardWithTitle := lipgloss.JoinVertical(lipgloss.Left, title, "", board)

	game := lipgloss.JoinHorizontal(lipgloss.Bottom, help, "  ", boardWithTitle, "  ", side)

	return lipgloss.Place(m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		game,
	)
}

func (m *Model) renderBoard() string {
	// Build a grid of strings to render.
	grid := [boardRows][boardCols]string{}

	// Filled board cells.
	for r := 0; r < boardRows; r++ {
		for c := 0; c < boardCols; c++ {
			if m.board[r][c].filled {
				color := pieceColor[m.board[r][c].pieceType]
				grid[r][c] = lipgloss.NewStyle().Foreground(color).Render(cellFull)
			} else {
				grid[r][c] = ui.Muted.Render(cellEmpty)
			}
		}
	}

	// Ghost piece.
	ghostR := m.ghostRow()
	ghostStyle := lipgloss.NewStyle().Foreground(ui.ColorMuted).Faint(true)
	for _, pos := range Cells(m.current.shape()) {
		r := ghostR + pos[0]
		c := m.current.col + pos[1]
		if r >= 0 && r < boardRows && c >= 0 && c < boardCols && !m.board[r][c].filled {
			grid[r][c] = ghostStyle.Render(cellGhost)
		}
	}

	// Active piece (drawn on top of ghost).
	activeColor := pieceColor[m.current.pt]
	activeStyle := lipgloss.NewStyle().Foreground(activeColor)
	for _, pos := range Cells(m.current.shape()) {
		r := m.current.row + pos[0]
		c := m.current.col + pos[1]
		if r >= 0 && r < boardRows && c >= 0 && c < boardCols {
			grid[r][c] = activeStyle.Render(cellFull)
		}
	}

	// Render rows to string.
	var sb strings.Builder
	for r := 0; r < boardRows; r++ {
		sb.WriteString(strings.Join(grid[r][:], ""))
		if r < boardRows-1 {
			sb.WriteByte('\n')
		}
	}

	inner := sb.String()

	// Game-over overlay: place centered text on top of the board content,
	// then wrap in the border.
	if !m.alive {
		// Each cell is 2 chars wide; board inner width = boardCols*2.
		innerW := boardCols * 2
		overlay := ui.Error.Render(" game over ")
		inner = lipgloss.Place(innerW, boardRows,
			lipgloss.Center, lipgloss.Center,
			overlay,
			lipgloss.WithWhitespaceChars(" "),
		)
	}

	return ui.Border.Render(inner)
}

func (m *Model) renderHelp() string {
	line := func(s string) string { return ui.Muted.Render(s) }
	return lipgloss.JoinVertical(lipgloss.Left,
		line("←→   move"),
		line("↑/x  rotate cw"),
		line("z    rotate ccw"),
		line("↓    soft drop"),
		line("spc  hard drop"),
		line("c    hold"),
		line("r    restart"),
		line("q    quit"),
	)
}

func (m *Model) renderSide() string {
	nextLabel := ui.Subtitle.Render("next")
	nextPiece := m.renderMiniPiece(m.next)

	holdLabel := ui.Subtitle.Render("hold")
	var holdPiece string
	if m.hold != nil {
		holdPiece = m.renderMiniPiece(*m.hold)
	} else {
		holdPiece = ui.Muted.Render("─")
	}

	stats := lipgloss.JoinVertical(lipgloss.Left,
		ui.Subtitle.Render("score"),
		strconv.Itoa(m.score),
		"",
		ui.Subtitle.Render("level"),
		strconv.Itoa(m.level),
		"",
		ui.Subtitle.Render("lines"),
		strconv.Itoa(m.lines),
	)

	return lipgloss.JoinVertical(lipgloss.Left,
		nextLabel,
		nextPiece,
		"",
		holdLabel,
		holdPiece,
		"",
		stats,
	)
}

func (m *Model) renderMiniPiece(pt PieceType) string {
	s := Rotation(pt, 0)
	color := pieceColor[pt]
	style := lipgloss.NewStyle().Foreground(color)

	var sb strings.Builder
	for r := 0; r < 4; r++ {
		for c := 0; c < 4; c++ {
			if s[r][c] {
				sb.WriteString(style.Render(cellFull))
			} else {
				sb.WriteString(cellEmpty)
			}
		}
		if r < 3 {
			sb.WriteByte('\n')
		}
	}
	return sb.String()
}
