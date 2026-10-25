package snake

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/mahyarmirrashed/arcade/internal/ui"
)

// Cell glyphs, each cellWidth characters wide.
const (
	emptyCell = "  "
	snakeCell = "██"
	deadCell  = "XX"
	foodCell  = "()"
)

var (
	headStyle = ui.Accent
	bodyStyle = lipgloss.NewStyle().Foreground(ui.ColorSubtle)
	foodStyle = lipgloss.NewStyle().Foreground(ui.ColorText)
)

// View implements tea.Model.
func (m *Model) View() string {
	if m.tooSmall {
		return m.place(m.tooSmallView())
	}
	width := m.board.cols*cellWidth + borderW
	return m.place(lipgloss.JoinVertical(lipgloss.Left,
		m.header(width),
		ui.Border.Render(strings.Join(m.gridLines(), "\n")),
		m.footer(width),
	))
}

// place centres content in the terminal.
func (m *Model) place(content string) string {
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
}

// header puts the title on the left and the score on the right across width.
func (m *Model) header(width int) string {
	title := ui.Title.Render("snake")
	score := ui.Muted.Render("score ") + ui.Accent.Render(strconv.Itoa(m.board.score))
	gap := max(width-lipgloss.Width(title)-lipgloss.Width(score), 1)
	return title + strings.Repeat(" ", gap) + score
}

// footer lists the keys that do something right now. Hints are dropped from
// the front until the rest fit within width, so small boards don't wrap.
func (m *Model) footer(width int) string {
	var hints []string
	switch {
	case !m.playing():
		hints = []string{"r  restart", "q  quit"}
	case m.paused:
		hints = []string{"p  resume", "q  quit"}
	default:
		hints = []string{"arrows/hjkl  move", "p  pause", "q  quit"}
	}
	line := strings.Join(hints, "    ")
	for len(hints) > 1 && lipgloss.Width(line) > width {
		hints = hints[1:]
		line = strings.Join(hints, "    ")
	}
	return ui.Muted.Render(line)
}

// status is the message shown across the middle of the board, if any.
func (m *Model) status() string {
	switch {
	case m.board.won:
		return ui.Accent.Render("you win!")
	case !m.board.alive:
		return ui.Error.Render("game over")
	case m.paused:
		return ui.Subtitle.Render("paused")
	}
	return ""
}

// gridLines renders the board one row per line, with any status message
// written over the middle row.
func (m *Model) gridLines() []string {
	b := &m.board
	head := headStyle.Render(snakeCell)
	if !b.alive {
		head = ui.Error.Render(deadCell)
	}
	body := bodyStyle.Render(snakeCell)
	food := foodStyle.Render(foodCell)

	occupied := make(map[point]bool, len(b.body))
	for _, p := range b.body[1:] {
		occupied[p] = true
	}

	lines := make([]string, b.rows)
	for y := range b.rows {
		var sb strings.Builder
		for x := range b.cols {
			switch p := (point{x, y}); {
			case p == b.body[0]:
				sb.WriteString(head)
			case occupied[p]:
				sb.WriteString(body)
			case p == b.food && !b.won:
				sb.WriteString(food)
			default:
				sb.WriteString(emptyCell)
			}
		}
		lines[y] = sb.String()
	}

	if s := m.status(); s != "" {
		lines[b.rows/2] = lipgloss.PlaceHorizontal(b.cols*cellWidth, lipgloss.Center, s)
	}
	return lines
}

func (m *Model) tooSmallView() string {
	minW := minCols*cellWidth + borderW
	minH := minRows + borderH + headerH + footerH
	return ui.Border.Padding(0, 1).Render(
		ui.Error.Render("terminal too small") + "\n\n" +
			ui.Muted.Render(fmt.Sprintf("resize to at least %dx%d", minW, minH)) + "\n\n" +
			ui.Muted.Render("q  quit"),
	)
}
