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
	grid := ui.Border.Render(m.renderGrid())
	return m.place(lipgloss.JoinVertical(lipgloss.Left,
		m.header(lipgloss.Width(grid)),
		grid,
		m.footer(),
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

func (m *Model) footer() string {
	return ui.Muted.Render("arrows/hjkl  move    q  quit")
}

func (m *Model) renderGrid() string {
	b := &m.board
	head := headStyle.Render(snakeCell)
	body := bodyStyle.Render(snakeCell)
	food := foodStyle.Render(foodCell)

	occupied := make(map[point]bool, len(b.body))
	for _, p := range b.body[1:] {
		occupied[p] = true
	}

	var sb strings.Builder
	for y := range b.rows {
		if y > 0 {
			sb.WriteByte('\n')
		}
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
	}
	return sb.String()
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
