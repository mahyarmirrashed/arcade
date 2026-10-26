package connect4

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/mahyarmirrashed/arcade/internal/game"
	"github.com/mahyarmirrashed/arcade/internal/ui"
)

// Compile-time check.
var _ game.Game = (*Model)(nil)

// modelState is the per-player UI phase.
type modelState int

const (
	stateWaiting modelState = iota
	stateJoining
	statePlaying
	stateGrace
	stateOver
)

type keyMap struct {
	left  key.Binding
	right key.Binding
	drop  key.Binding
}

var keys = keyMap{
	left:  key.NewBinding(key.WithKeys("left", "h"), key.WithHelp("←/h", "left")),
	right: key.NewBinding(key.WithKeys("right", "l"), key.WithHelp("→/l", "right")),
	drop:  key.NewBinding(key.WithKeys("enter", " "), key.WithHelp("enter", "drop")),
}

// Model is the per-player Bubbletea model.
type Model struct {
	seat      playerIndex
	room      *Room
	code      string
	state     modelState
	cursor    int
	joinInput textinput.Model
	notice    string
	snap      snapshot
	width     int
	height    int
	styles    ui.Styles
}

// snapshot is a safe copy of room state for rendering.
type snapshot struct {
	phase     roomPhase
	board     board
	turn      playerIndex
	won       bool
	draw      bool
	winner    playerIndex
	winCells  [][2]int
	cursors   [2]int
	graceEnds string
}

// - game.Game -

func (m *Model) Name() string        { return "Connect 4" }
func (m *Model) Description() string { return "drop pieces, connect four" }

func (m *Model) New(width, height int, styles ui.Styles) tea.Model {
	return newModel(width, height, styles)
}

func newModel(width, height int, styles ui.Styles) *Model {
	room := Broker.Create()
	ji := textinput.New()
	ji.Placeholder = "A2BF" // example room code format
	ji.CharLimit = 4
	ji.Width = 6
	return &Model{
		seat:      player0,
		room:      room,
		code:      room.Code,
		state:     stateWaiting,
		width:     width,
		height:    height,
		styles:    styles,
		joinInput: ji,
		snap:      takeSnapshot(room, player0),
	}
}

// - tea.Model -

func (m *Model) Init() tea.Cmd { return m.waitForUpdate() }

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case stateMsg:
		m.snap = takeSnapshot(msg.room, m.seat)
		m.syncState()
		return m, m.waitForUpdate()

	case joinResultMsg:
		if msg.room == nil {
			m.notice = "room not found"
			return m, nil
		}
		m.room = msg.room
		m.code = msg.room.Code
		m.seat = player1
		m.state = statePlaying
		m.snap = takeSnapshot(msg.room, player1)
		return m, m.waitForUpdate()

	case tea.KeyMsg:
		return m, m.handleKey(msg)
	}
	return m, nil
}

func (m *Model) syncState() {
	switch m.snap.phase {
	case phaseWaiting:
		m.state = stateWaiting
	case phasePlaying:
		m.state = statePlaying
	case phaseGrace:
		m.state = stateGrace
	case phaseOver:
		m.state = stateOver
	}
}

func (m *Model) waitForUpdate() tea.Cmd {
	ch := Broker.Channel(m.code, m.seat)
	if ch == nil {
		return nil
	}
	return func() tea.Msg { return <-ch }
}

func (m *Model) handleKey(msg tea.KeyMsg) tea.Cmd {
	m.notice = ""

	if msg.String() == "r" && m.state == stateOver {
		fresh := newModel(m.width, m.height, m.styles)
		return func() tea.Msg { return replaceMsg{fresh} }
	}

	switch m.state {
	case stateWaiting:
		if msg.String() == "j" {
			m.state = stateJoining
			m.joinInput.Focus()
			return textinput.Blink
		}

	case stateJoining:
		if msg.String() == "esc" {
			m.state = stateWaiting
			m.joinInput.Blur()
			return nil
		}
		if msg.Type == tea.KeyEnter {
			code := strings.ToUpper(strings.TrimSpace(m.joinInput.Value()))
			if len(code) != 4 {
				m.notice = "code must be 4 characters"
				return nil
			}
			return joinRoom(code)
		}
		var cmd tea.Cmd
		m.joinInput, cmd = m.joinInput.Update(msg)
		return cmd

	case statePlaying:
		if m.snap.turn != m.seat {
			return nil
		}
		switch {
		case key.Matches(msg, keys.left):
			if m.cursor > 0 {
				m.cursor--
				Broker.MoveCursor(m.code, m.seat, m.cursor)
			}
		case key.Matches(msg, keys.right):
			if m.cursor < cols-1 {
				m.cursor++
				Broker.MoveCursor(m.code, m.seat, m.cursor)
			}
		case key.Matches(msg, keys.drop):
			if !colFull(m.snap.board, m.cursor) {
				return dropPiece(m.code, m.seat, m.cursor)
			}
			m.notice = "column full"
		}
	}
	return nil
}

type joinResultMsg struct{ room *Room }
type replaceMsg struct{ m *Model }

func joinRoom(code string) tea.Cmd {
	return func() tea.Msg {
		r, ok := Broker.Join(code)
		if !ok {
			return joinResultMsg{nil}
		}
		return joinResultMsg{r}
	}
}

func dropPiece(code string, seat playerIndex, col int) tea.Cmd {
	return func() tea.Msg {
		Broker.Drop(code, seat, col)
		return nil
	}
}

// - View -

// piece colours: player 0 = red, player 1 = yellow.
var pieceColor = [2]lipgloss.Color{
	ui.Spectrum[0], // red
	ui.Spectrum[2], // yellow
}

func (m *Model) View() string {
	var content string
	switch m.state {
	case stateWaiting:
		content = m.viewWaiting()
	case stateJoining:
		content = m.viewJoining()
	case statePlaying, stateGrace, stateOver:
		content = m.viewGame()
	default:
		content = m.viewWaiting()
	}
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
}

func (m *Model) viewWaiting() string {
	return lipgloss.JoinVertical(lipgloss.Left,
		m.styles.Title.Render("connect 4"),
		"",
		m.styles.Subtitle.Render("your room code"),
		"",
		m.styles.NewStyle().Bold(true).Foreground(ui.ColorText).
			Width(8).Align(lipgloss.Center).Render(m.code),
		"",
		m.styles.Muted.Render("share this code with your opponent"),
		"",
		m.styles.Muted.Render("waiting for opponent to join..."),
		"",
		m.styles.Muted.Render("j to join a room instead · esc to quit"),
	)
}

func (m *Model) viewJoining() string {
	notice := ""
	if m.notice != "" {
		notice = m.styles.Error.Render(" " + m.notice + " ")
	}
	return lipgloss.JoinVertical(lipgloss.Left,
		m.styles.Title.Render("connect 4"),
		"",
		m.styles.Subtitle.Render("enter room code"),
		"",
		m.joinInput.View(),
		"",
		notice,
		"",
		m.styles.Muted.Render("enter to join · esc to go back"),
	)
}

func (m *Model) viewGame() string {
	help := m.renderHelp()
	board := m.renderBoard()
	side := m.renderSide()

	title := m.styles.Title.Render("connect 4")
	boardWithTitle := lipgloss.JoinVertical(lipgloss.Left, title, "", board)

	pad := m.styles.NewStyle().PaddingBottom(1)
	layout := lipgloss.JoinHorizontal(lipgloss.Bottom,
		pad.Render(help), "  ", boardWithTitle, "  ", pad.Render(side),
	)
	return layout
}

func (m *Model) renderBoard() string {
	s := m.snap

	// Build win cell set for highlighting.
	winSet := make(map[[2]int]bool, len(s.winCells))
	for _, c := range s.winCells {
		winSet[[2]int{c[0], c[1]}] = true
	}

	// Cursor row: show where pieces would drop.
	cursorRow := m.renderCursorRow(s)

	// Grid rows.
	gridLines := make([]string, rows)
	for r := 0; r < rows; r++ {
		var sb strings.Builder
		sb.WriteString(m.styles.NewStyle().Foreground(ui.ColorSubtle).Render("│"))
		for c := 0; c < cols; c++ {
			cl := s.board[r][c]
			var glyph string
			switch cl {
			case p0:
				st := m.styles.NewStyle().Foreground(pieceColor[0])
				if winSet[[2]int{r, c}] {
					st = st.Reverse(true)
				}
				glyph = st.Render("●")
			case p1:
				st := m.styles.NewStyle().Foreground(pieceColor[1])
				if winSet[[2]int{r, c}] {
					st = st.Reverse(true)
				}
				glyph = st.Render("●")
			default:
				glyph = m.styles.NewStyle().Foreground(ui.ColorMuted).Render("·")
			}
			sb.WriteString(" " + glyph + " ")
			sb.WriteString(m.styles.NewStyle().Foreground(ui.ColorSubtle).Render("│"))
		}
		gridLines[r] = sb.String()
	}

	sep := m.styles.NewStyle().Foreground(ui.ColorSubtle).Render(
		"├" + strings.Repeat("───┼", cols-1) + "───┤",
	)
	top := m.styles.NewStyle().Foreground(ui.ColorSubtle).Render(
		"┌" + strings.Repeat("───┬", cols-1) + "───┐",
	)
	bot := m.styles.NewStyle().Foreground(ui.ColorSubtle).Render(
		"└" + strings.Repeat("───┴", cols-1) + "───┘",
	)

	parts := []string{cursorRow, top}
	for r, line := range gridLines {
		parts = append(parts, line)
		if r < rows-1 {
			parts = append(parts, sep)
		}
	}
	parts = append(parts, bot)

	inner := lipgloss.JoinVertical(lipgloss.Left, parts...)

	// Overlay for game over.
	if s.phase == phaseOver {
		var msg string
		if s.draw {
			msg = " draw! "
		} else if s.winner == m.seat {
			msg = " you win! "
		} else {
			msg = " you lose "
		}
		w := lipgloss.Width(inner)
		h := lipgloss.Height(inner)
		inner = lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center,
			m.styles.Error.Render(msg),
			lipgloss.WithWhitespaceChars(" "),
		)
	}

	if s.phase == phaseGrace {
		w := lipgloss.Width(inner)
		h := lipgloss.Height(inner)
		overlay := lipgloss.JoinVertical(lipgloss.Center,
			m.styles.Error.Render(" opponent disconnected "),
			m.styles.Muted.Render("waiting "+s.graceEnds+"..."),
		)
		inner = lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center,
			overlay, lipgloss.WithWhitespaceChars(" "))
	}

	return inner
}

// renderCursorRow shows a ▼ above the active column for each player.
func (m *Model) renderCursorRow(s snapshot) string {
	cells := make([]string, cols)
	for c := 0; c < cols; c++ {
		w := m.styles.NewStyle().Width(4).Align(lipgloss.Center)
		// My cursor.
		if s.turn == m.seat && c == m.cursor {
			cells[c] = w.Foreground(pieceColor[m.seat]).Render("▼")
		} else if s.turn == m.seat.opponent() && c == s.cursors[m.seat.opponent()] {
			// Opponent's cursor (only shown on their turn).
			cells[c] = w.Foreground(pieceColor[m.seat.opponent()]).Render("▼")
		} else {
			cells[c] = w.Render(" ")
		}
	}
	return strings.Join(cells, "")
}

func (m *Model) renderSide() string {
	s := m.snap
	myColor := m.styles.NewStyle().Foreground(pieceColor[m.seat]).Render("●")
	oppColor := m.styles.NewStyle().Foreground(pieceColor[m.seat.opponent()]).Render("●")

	turnLine := ""
	if s.phase == phasePlaying {
		if s.turn == m.seat {
			turnLine = m.styles.Accent.Render("your turn")
		} else {
			turnLine = m.styles.Muted.Render("their turn")
		}
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		m.styles.Subtitle.Render("you"),
		myColor,
		"",
		m.styles.Subtitle.Render("opponent"),
		oppColor,
		"",
		turnLine,
		"",
		m.styles.Subtitle.Render("code"),
		m.styles.Accent.Render(m.code),
		"",
		m.styles.Muted.Render("r to play again"),
	)
}

func (m *Model) renderHelp() string {
	line := func(s string) string { return m.styles.Muted.Render(s) }
	return lipgloss.JoinVertical(lipgloss.Left,
		line("←→   move"),
		line("enter  drop"),
		line("esc   quit"),
	)
}

func takeSnapshot(r *Room, seat playerIndex) snapshot {
	s := snapshot{
		phase:    r.Phase,
		board:    r.Game.Board,
		turn:     r.Game.Turn,
		won:      r.Game.Won,
		draw:     r.Game.Draw,
		winner:   r.Game.Winner,
		winCells: r.Game.WinCells,
		cursors:  r.Cursor,
	}
	if r.Phase == phaseGrace {
		remaining := r.GraceEnds.Sub(r.now())
		if remaining < 0 {
			remaining = 0
		}
		s.graceEnds = fmt.Sprintf("%ds", int(remaining.Seconds()))
	}
	return s
}
