package snake

import (
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/mahyarmirrashed/arcade/internal/ui"
)

// Layout. Each cell is drawn two characters wide so it looks square in a
// terminal, where characters are about twice as tall as they are wide.
const (
	cellWidth = 2
	borderW   = 2 // left and right border columns
	borderH   = 2 // top and bottom border rows
	titleH    = 2 // title and the blank line under it
	gapW      = 2 // space between the help panel, the board and the side panel
	sideW     = 6 // widest side panel line, "length"

	minCols, minRows = 10, 10
	// The board is capped so huge terminals don't make the game sluggish.
	maxCols, maxRows = 40, 20
)

// helpLines are the controls listed left of the board.
var helpLines = []string{
	"↑↓←→ move",
	"hjkl move",
	"p    pause",
	"r    restart",
	"q    quit",
}

// Cell glyphs, each cellWidth characters wide.
const (
	emptyCell = "  "
	snakeCell = "██"
	foodCell  = "()"
)

// spectrumIndex is the position in ui.Spectrum of body segment i, counting
// the neck as 1, so the colours run through the spectrum from neck to tail.
func spectrumIndex(i int) int { return (i - 1) % len(ui.Spectrum) }

// segmentColor is the colour of body segment i, counting the neck as 1.
func segmentColor(i int) lipgloss.Color { return ui.Spectrum[spectrumIndex(i)] }

// foodColor is the colour of the segment that eating the food will add, so
// the food shows the block the player is about to gain.
func foodColor(b *board) lipgloss.Color { return segmentColor(len(b.body)) }

// chromeWidth is the number of columns taken by everything beside the board's
// cells: the help panel, the side panel, the gaps and the border.
func chromeWidth() int {
	return lipgloss.Width(strings.Join(helpLines, "\n")) + gapW + borderW + gapW + sideW
}

// gridSize returns how many cells fit in a terminal of the given size, capped
// at maxCols x maxRows. ok is false when the terminal is too small to play.
func gridSize(termW, termH int) (cols, rows int, ok bool) {
	cols = min((termW-chromeWidth())/cellWidth, maxCols)
	rows = min(termH-titleH-borderH, maxRows)
	return cols, rows, cols >= minCols && rows >= minRows
}

// View implements tea.Model. It follows the arcade layout: help on the left,
// the title above the board in the middle and stats on the right, all
// bottom-aligned.
func (m *Model) View() string {
	if m.tooSmall {
		w, h := m.minSize()
		return m.styles.TooSmall(m.width, m.height, w, h)
	}

	// The border adds a row below the board's cells; padding the side panels
	// by one row lines their last line up with the board's last row of cells.
	pad := m.styles.NewStyle().PaddingBottom(1)
	gap := strings.Repeat(" ", gapW)
	boardWithTitle := lipgloss.JoinVertical(lipgloss.Left,
		m.styles.Title.Render("snake"),
		"",
		m.renderBoard(),
	)

	return m.place(lipgloss.JoinHorizontal(lipgloss.Bottom,
		pad.Render(m.renderHelp()), gap,
		boardWithTitle, gap,
		pad.Render(m.renderSide()),
	))
}

// place centres content in the terminal.
func (m *Model) place(content string) string {
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
}

func (m *Model) renderHelp() string {
	lines := make([]string, len(helpLines))
	for i, l := range helpLines {
		lines[i] = m.styles.Muted.Render(l)
	}
	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func (m *Model) renderSide() string {
	return lipgloss.JoinVertical(lipgloss.Left,
		m.styles.Subtitle.Render("score"),
		m.styles.Accent.Render(strconv.Itoa(m.board.score)),
		"",
		m.styles.Subtitle.Render("length"),
		m.styles.Accent.Render(strconv.Itoa(len(m.board.body))),
	)
}

// status is the message shown over the board, if any.
func (m *Model) status() string {
	switch {
	case m.board.won:
		return m.styles.Accent.Render("you win")
	case !m.board.alive:
		return m.styles.Error.Render(" game over ")
	case m.paused:
		return m.styles.Subtitle.Render("paused")
	}
	return ""
}

// renderBoard draws the grid inside the border, or the status message in its
// place when the game is paused or over.
func (m *Model) renderBoard() string {
	inner := m.renderGrid()
	if s := m.status(); s != "" {
		inner = lipgloss.Place(m.board.cols*cellWidth, m.board.rows,
			lipgloss.Center, lipgloss.Center,
			s,
			lipgloss.WithWhitespaceChars(" "),
		)
	}
	return m.styles.Border.Render(inner)
}

func (m *Model) renderGrid() string {
	b := &m.board
	head := m.styles.NewStyle().Foreground(ui.ColorText).Render(snakeCell)
	food := m.styles.NewStyle().Foreground(foodColor(b)).Render(foodCell)
	bodies := make([]string, len(ui.Spectrum))
	for k, c := range ui.Spectrum {
		bodies[k] = m.styles.NewStyle().Foreground(c).Render(snakeCell)
	}

	segment := make(map[point]int, len(b.body))
	for i, p := range b.body[1:] {
		segment[p] = i + 1
	}

	lines := make([]string, b.rows)
	for y := range b.rows {
		var sb strings.Builder
		for x := range b.cols {
			p := point{x, y}
			i, inBody := segment[p]
			switch {
			case p == b.body[0]:
				sb.WriteString(head)
			case inBody:
				sb.WriteString(bodies[spectrumIndex(i)])
			case p == b.food && !b.won:
				sb.WriteString(food)
			default:
				sb.WriteString(emptyCell)
			}
		}
		lines[y] = sb.String()
	}
	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

// minSize is the smallest terminal that can show the current game, or the
// smallest board if there is no game yet.
func (m *Model) minSize() (width, height int) {
	cols, rows := minCols, minRows
	if m.hasGame() {
		cols, rows = m.board.cols, m.board.rows
	}
	return cols*cellWidth + chromeWidth(), rows + titleH + borderH
}
