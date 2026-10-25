package solitaire

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

// Compile-time check: Model must satisfy game.Game.
var _ game.Game = (*Model)(nil)

// cursor positions:
//   0-6: tableau columns 0-6
//   7-10: foundations 0-3
//   11: stock
//   12: waste

const (
	cursorTableauStart    = 0
	cursorTableauEnd      = 6
	cursorFoundationStart = 7
	cursorFoundationEnd   = 10
	cursorStock           = 11
	cursorWaste           = 12
	cursorMax             = 12
)

type keyMap struct {
	left   key.Binding
	right  key.Binding
	up     key.Binding
	down   key.Binding
	pick   key.Binding // select / place
	auto   key.Binding // auto-move to foundation
	draw   key.Binding // draw from stock
	quit   key.Binding
}

var keys = keyMap{
	left:  key.NewBinding(key.WithKeys("left", "h"), key.WithHelp("←/h", "left")),
	right: key.NewBinding(key.WithKeys("right", "l"), key.WithHelp("→/l", "right")),
	up:    key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
	down:  key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
	pick:  key.NewBinding(key.WithKeys("enter", " "), key.WithHelp("enter", "pick/place")),
	auto:  key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "auto")),
	draw:  key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "draw")),
	quit:  key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
}

// selection holds a grabbed sequence of cards and where they came from.
type selection struct {
	fromCursor int
	fromIdx    int // index into the source pile's cards slice
}

// gameState is the phase of the game.
type gameState int

const (
	statePlaying gameState = iota
	stateWon
)

// Model is the Bubbletea model for Klondike solitaire.
type Model struct {
	tableau     [7]pile
	foundations [4]pile
	stock       pile
	waste       pile

	cursor int
	sel    *selection // nil = nothing selected

	state  gameState
	width  int
	height int
	styles ui.Styles
}

// - game.Game implementation -

func (m *Model) Name() string        { return "Solitaire" }
func (m *Model) Description() string { return "klondike - clear the tableau" }

func (m *Model) New(width, height int, styles ui.Styles) tea.Model {
	return newModel(width, height, styles)
}

func newModel(width, height int, styles ui.Styles) *Model {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	m := &Model{
		width:  width,
		height: height,
		styles: styles,
		cursor: 0,
	}
	m.deal(rng)
	return m
}

// deal distributes cards for a new Klondike game.
func (m *Model) deal(rng *rand.Rand) {
	deck := newDeck(rng)

	// Tableau: column i gets i+1 cards; only the top card is face-up.
	idx := 0
	for i := 0; i < 7; i++ {
		m.tableau[i] = pile{kind: kindTableau}
		for j := 0; j <= i; j++ {
			c := deck[idx]
			c.FaceUp = (j == i)
			m.tableau[i].cards = append(m.tableau[i].cards, c)
			idx++
		}
	}

	// Stock: remaining cards face-down.
	m.stock = pile{kind: kindStock}
	for ; idx < 52; idx++ {
		c := deck[idx]
		c.FaceUp = false
		m.stock.cards = append(m.stock.cards, c)
	}

	// Foundations and waste start empty.
	for i := range m.foundations {
		m.foundations[i] = pile{kind: kindFoundation}
	}
	m.waste = pile{kind: kindWaste}
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

type restartMsg struct{ m *Model }

func (m *Model) handleKey(msg tea.KeyMsg) tea.Cmd {
	if msg.String() == "r" && m.state == stateWon {
		fresh := newModel(m.width, m.height, m.styles)
		return func() tea.Msg { return restartMsg{fresh} }
	}

	if m.state != statePlaying {
		return nil
	}

	switch {
	case key.Matches(msg, keys.left):
		if m.cursor > 0 {
			m.cursor--
		}
	case key.Matches(msg, keys.right):
		if m.cursor < cursorMax {
			m.cursor++
		}
	case key.Matches(msg, keys.up):
		m.moveCursorUp()
	case key.Matches(msg, keys.down):
		m.moveCursorDown()
	case key.Matches(msg, keys.pick):
		m.pick()
	case key.Matches(msg, keys.auto):
		m.autoMove()
	case key.Matches(msg, keys.draw):
		m.draw()
	}
	return nil
}

// moveCursorUp moves the cursor from tableau row to the top row (foundations/stock/waste).
func (m *Model) moveCursorUp() {
	if m.cursor <= cursorTableauEnd {
		// Move to the foundation/stock row above.
		if m.cursor < 4 {
			m.cursor = cursorFoundationStart + m.cursor
		} else if m.cursor == 4 {
			m.cursor = cursorStock
		} else {
			m.cursor = cursorWaste
		}
	}
}

// moveCursorDown moves cursor from top row back to tableau.
func (m *Model) moveCursorDown() {
	switch {
	case m.cursor >= cursorFoundationStart && m.cursor <= cursorFoundationEnd:
		m.cursor = m.cursor - cursorFoundationStart
	case m.cursor == cursorStock:
		m.cursor = 4
	case m.cursor == cursorWaste:
		m.cursor = 5
	}
}

// currentPile returns a pointer to the pile at the current cursor position.
func (m *Model) currentPile() *pile {
	switch {
	case m.cursor <= cursorTableauEnd:
		return &m.tableau[m.cursor]
	case m.cursor <= cursorFoundationEnd:
		return &m.foundations[m.cursor-cursorFoundationStart]
	case m.cursor == cursorStock:
		return &m.stock
	default:
		return &m.waste
	}
}

// pick handles the select/place action at the current cursor.
func (m *Model) pick() {
	p := m.currentPile()

	if m.sel == nil {
		// Nothing selected yet - try to select something.
		if m.cursor == cursorStock {
			m.draw()
			return
		}
		if len(p.cards) == 0 {
			return
		}
		top := p.top()
		if !top.FaceUp {
			return
		}
		// For waste/foundation, only the top card can be selected.
		// For tableau, select the topmost face-up run.
		fromIdx := len(p.cards) - 1
		if m.cursor <= cursorTableauEnd {
			for fromIdx > 0 && p.cards[fromIdx-1].FaceUp {
				fromIdx--
			}
		}
		m.sel = &selection{fromCursor: m.cursor, fromIdx: fromIdx}
		return
	}

	// Something is selected - try to place it.
	src := m.pileAt(m.sel.fromCursor)
	cards := src.cards[m.sel.fromIdx:]

	placed := false
	switch {
	case m.cursor <= cursorTableauEnd:
		if canPlaceSequence(p, src.cards, m.sel.fromIdx) {
			p.cards = append(p.cards, cards...)
			src.cards = src.cards[:m.sel.fromIdx]
			m.flipTop(src)
			placed = true
		}
	case m.cursor <= cursorFoundationEnd && len(cards) == 1:
		if canPlaceOnFoundation(p, cards[0]) {
			p.cards = append(p.cards, cards[0])
			src.cards = src.cards[:m.sel.fromIdx]
			m.flipTop(src)
			placed = true
		}
	}

	m.sel = nil
	if placed && isComplete(m.foundations) {
		m.state = stateWon
	}
}

// autoMove tries to move the top card at the cursor to a foundation.
func (m *Model) autoMove() {
	p := m.currentPile()
	if i := autoFoundation(p, &m.foundations); i >= 0 {
		m.flipTop(p)
		if isComplete(m.foundations) {
			m.state = stateWon
		}
	}
}

// draw turns one card from stock to waste. If stock is empty, recycles waste.
func (m *Model) draw() {
	if len(m.stock.cards) == 0 {
		// Recycle waste back to stock face-down.
		for i := len(m.waste.cards) - 1; i >= 0; i-- {
			c := m.waste.cards[i]
			c.FaceUp = false
			m.stock.cards = append(m.stock.cards, c)
		}
		m.waste.cards = m.waste.cards[:0]
		return
	}
	c := m.stock.cards[len(m.stock.cards)-1]
	c.FaceUp = true
	m.waste.cards = append(m.waste.cards, c)
	m.stock.cards = m.stock.cards[:len(m.stock.cards)-1]
}

// flipTop flips the new top card of p face-up if it is face-down.
func (m *Model) flipTop(p *pile) {
	if len(p.cards) > 0 && !p.cards[len(p.cards)-1].FaceUp {
		p.cards[len(p.cards)-1].FaceUp = true
	}
}

// pileAt returns a pointer to the pile at cursor position c.
func (m *Model) pileAt(c int) *pile {
	switch {
	case c <= cursorTableauEnd:
		return &m.tableau[c]
	case c <= cursorFoundationEnd:
		return &m.foundations[c-cursorFoundationStart]
	case c == cursorStock:
		return &m.stock
	default:
		return &m.waste
	}
}

// - View -

func (m *Model) View() string {
	help := m.renderHelp()
	board := m.renderBoard()

	title := m.styles.Title.Render("solitaire")
	boardWithTitle := lipgloss.JoinVertical(lipgloss.Left, title, "", board)

	pad := m.styles.NewStyle().PaddingBottom(1)
	layout := lipgloss.JoinHorizontal(lipgloss.Bottom,
		pad.Render(help), "  ", boardWithTitle,
	)

	return lipgloss.Place(m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		layout,
	)
}

func (m *Model) renderBoard() string {
	topRow := m.renderTopRow()
	tableauRow := m.renderTableau()

	board := lipgloss.JoinVertical(lipgloss.Left, topRow, "", tableauRow)

	if m.state == stateWon {
		w := lipgloss.Width(board)
		h := lipgloss.Height(board)
		board = lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center,
			m.styles.Error.Render(" you win! "),
			lipgloss.WithWhitespaceChars(" "),
		)
	}

	return m.styles.Border.Render(board)
}

// renderTopRow renders: foundations (4) + gap + stock + waste.
func (m *Model) renderTopRow() string {
	parts := make([]string, 0, 10)

	for i := 0; i < 4; i++ {
		cursor := m.cursor == cursorFoundationStart+i
		sel := m.sel != nil && m.sel.fromCursor == cursorFoundationStart+i
		f := &m.foundations[i]
		var card string
		if len(f.cards) == 0 {
			card = renderEmpty(m.styles)
		} else {
			card = renderCard(*f.top(), cursor || sel, m.styles)
		}
		if cursor {
			card = m.styles.NewStyle().BorderStyle(lipgloss.RoundedBorder()).
				BorderForeground(ui.ColorText).Render(card)
		}
		parts = append(parts, card)
		if i < 3 {
			parts = append(parts, " ")
		}
	}

	parts = append(parts, "   ") // gap between foundations and stock

	// Stock
	stockCursor := m.cursor == cursorStock
	var stockCard string
	if len(m.stock.cards) == 0 {
		stockCard = renderEmpty(m.styles)
	} else {
		stockCard = renderBack(m.styles)
	}
	if stockCursor {
		stockCard = m.styles.NewStyle().BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(ui.ColorText).Render(stockCard)
	}
	parts = append(parts, stockCard, " ")

	// Waste
	wasteCursor := m.cursor == cursorWaste
	wasteSel := m.sel != nil && m.sel.fromCursor == cursorWaste
	var wasteCard string
	if len(m.waste.cards) == 0 {
		wasteCard = renderEmpty(m.styles)
	} else {
		wasteCard = renderCard(*m.waste.top(), wasteCursor || wasteSel, m.styles)
	}
	if wasteCursor {
		wasteCard = m.styles.NewStyle().BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(ui.ColorText).Render(wasteCard)
	}
	parts = append(parts, wasteCard)

	return lipgloss.JoinHorizontal(lipgloss.Top, parts...)
}

// renderTableau renders all 7 tableau columns side by side.
func (m *Model) renderTableau() string {
	cols := make([]string, 7)
	for i := 0; i < 7; i++ {
		cols[i] = m.renderTableauCol(i)
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, interleave(cols, " ")...)
}

// renderTableauCol renders a single tableau column as a vertical stack of cards.
// Face-down cards overlap heavily; only the top portion is shown.
// Face-up cards are stacked with a visible portion showing rank+suit.
func (m *Model) renderTableauCol(col int) string {
	p := &m.tableau[col]
	cursor := m.cursor == col
	selFrom := m.sel != nil && m.sel.fromCursor == col

	if len(p.cards) == 0 {
		e := renderEmpty(m.styles)
		if cursor {
			e = m.styles.NewStyle().BorderStyle(lipgloss.RoundedBorder()).
				BorderForeground(ui.ColorText).Render(e)
		}
		return e
	}

	// Build lines by overlapping cards - each card except the last shows only
	// its top 1 line (for face-down) or top 2 lines (for face-up).
	var lines []string
	for i, c := range p.cards {
		isLast := i == len(p.cards)-1
		selected := (cursor || selFrom) && m.sel != nil && i >= m.sel.fromIdx
		cardStr := renderCard(c, selected, m.styles)
		cardLines := strings.Split(cardStr, "\n")

		if isLast {
			lines = append(lines, cardLines...)
		} else if c.FaceUp {
			// Show top 2 lines of face-up cards.
			lines = append(lines, cardLines[:2]...)
		} else {
			// Show only the top line of face-down cards.
			lines = append(lines, cardLines[0])
		}
	}

	result := strings.Join(lines, "\n")
	if cursor && m.sel == nil {
		// Highlight the cursor column with a subtle left border.
		result = m.styles.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderLeft(true).
			BorderForeground(ui.ColorText).
			Render(result)
	}
	return result
}

// interleave inserts sep between each element.
func interleave(parts []string, sep string) []string {
	out := make([]string, 0, len(parts)*2-1)
	for i, p := range parts {
		out = append(out, p)
		if i < len(parts)-1 {
			out = append(out, sep)
		}
	}
	return out
}

func (m *Model) renderHelp() string {
	line := func(s string) string { return m.styles.Muted.Render(s) }
	return lipgloss.JoinVertical(lipgloss.Left,
		line("←→   move"),
		line("↑↓   top/tableau"),
		line("enter  pick/place"),
		line("a    auto-foundation"),
		line("d    draw"),
		line("r    restart"),
		line("q    quit"),
	)
}
