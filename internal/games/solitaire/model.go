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

// Compile-time check: Model must be a ranked game.
var _ game.Ranked = (*Model)(nil)

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
	left    key.Binding
	right   key.Binding
	up      key.Binding
	down    key.Binding
	pick    key.Binding // select / place
	auto    key.Binding // auto-move to foundation
	autoWin key.Binding // auto-complete when all face-up
	draw    key.Binding // draw from stock
	quit    key.Binding
}

var keys = keyMap{
	left:    key.NewBinding(key.WithKeys("left", "h"), key.WithHelp("←/h", "left")),
	right:   key.NewBinding(key.WithKeys("right", "l"), key.WithHelp("→/l", "right")),
	up:      key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
	down:    key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
	pick:    key.NewBinding(key.WithKeys("enter", " "), key.WithHelp("enter", "pick/place")),
	auto:    key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "auto-foundation")),
	autoWin: key.NewBinding(key.WithKeys("w"), key.WithHelp("w", "auto-win")),
	draw:    key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "draw")),
	quit:    key.NewBinding(key.WithKeys("esc", "ctrl+c"), key.WithHelp("esc", "quit")),
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

	now      func() time.Time // the clock; replaced in tests
	started  time.Time        // when the player first moved
	standing *game.Standing   // where the finished game placed, once known
}

// - game.Game implementation -

func (m *Model) Name() string        { return "Solitaire" }
func (m *Model) Description() string { return "klondike - clear the tableau" }

// Ranking implements game.Ranked.
func (m *Model) Ranking() game.Ranking { return game.ByTime }

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
		now:    time.Now,
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
		if !m.fits() {
			return m, nil
		}
		return m, m.handleKey(msg)
	case game.StandingMsg:
		if m.state == stateWon {
			s := msg.Standing
			m.standing = &s
		}
	}
	return m, nil
}

type restartMsg struct{ m *Model }

func (m *Model) handleKey(msg tea.KeyMsg) tea.Cmd {
	if msg.String() == "r" {
		fresh := newModel(m.width, m.height, m.styles)
		fresh.now = m.now
		return func() tea.Msg { return restartMsg{fresh} }
	}

	if m.state != statePlaying {
		return nil
	}
	if m.started.IsZero() {
		m.started = m.now()
	}

	switch {
	case key.Matches(msg, keys.left):
		if m.cursor > 0 {
			m.cursor--
		}
		m.sel = nil
	case key.Matches(msg, keys.right):
		if m.cursor < cursorMax {
			m.cursor++
		}
		m.sel = nil
	case key.Matches(msg, keys.up):
		m.moveCursorUp()
		m.sel = nil
	case key.Matches(msg, keys.down):
		m.moveCursorDown()
		m.sel = nil
	case key.Matches(msg, keys.pick):
		m.pick()
	case key.Matches(msg, keys.auto):
		m.autoMoveAll()
	case key.Matches(msg, keys.autoWin):
		m.autoWin()
	case key.Matches(msg, keys.draw):
		m.draw()
	}
	if m.state == stateWon {
		return game.Finished(m, game.Result{Won: true, Duration: m.now().Sub(m.started)})
	}
	return nil
}

// moveCursorUp moves the cursor from tableau row to the top row.
func (m *Model) moveCursorUp() {
	if m.cursor <= cursorTableauEnd {
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
	return m.pileAt(m.cursor)
}

// pick handles the select/place action at the current cursor.
func (m *Model) pick() {
	p := m.currentPile()

	if m.sel == nil {
		// Nothing selected - try to select.
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

		// For tableau, select the deepest face-up card (the whole run).
		// For waste/foundation, select just the top card.
		fromIdx := len(p.cards) - 1
		if m.cursor <= cursorTableauEnd {
			for fromIdx > 0 && p.cards[fromIdx-1].FaceUp {
				fromIdx--
			}
		}
		m.sel = &selection{fromCursor: m.cursor, fromIdx: fromIdx}
		return
	}

	// Already have a selection.

	// If picking the same pile again: cycle the selection up one card (select
	// a smaller sub-stack). If we're already at the top card, deselect.
	if m.cursor == m.sel.fromCursor && m.cursor <= cursorTableauEnd {
		src := m.pileAt(m.sel.fromCursor)
		if m.sel.fromIdx < len(src.cards)-1 {
			m.sel.fromIdx++
		} else {
			m.sel = nil // deselect
		}
		return
	}

	// Try to place the selected cards at the cursor.
	src := m.pileAt(m.sel.fromCursor)
	cards := src.cards[m.sel.fromIdx:]

	placed := false

	switch {
	case m.cursor <= cursorTableauEnd:
		// Place a sequence onto tableau.
		if canPlaceSequence(p, src.cards, m.sel.fromIdx) {
			p.cards = append(p.cards, cards...)
			src.cards = src.cards[:m.sel.fromIdx]
			m.flipTop(src)
			placed = true
		}
	case m.cursor <= cursorFoundationEnd:
		// Place a single card onto foundation.
		if len(cards) == 1 && canPlaceOnFoundation(p, cards[0]) {
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

// autoMoveAll sweeps all tableau tops and waste top to foundations in a loop
// until nothing more can be moved.
func (m *Model) autoMoveAll() {
	for {
		moved := false
		// Try waste first.
		if autoFoundation(&m.waste, &m.foundations) >= 0 {
			moved = true
		}
		// Try each tableau column.
		for i := 0; i < 7; i++ {
			if autoFoundation(&m.tableau[i], &m.foundations) >= 0 {
				m.flipTop(&m.tableau[i])
				moved = true
			}
		}
		if !moved {
			break
		}
	}
	if isComplete(m.foundations) {
		m.state = stateWon
	}
}

// autoWin checks whether all remaining cards are face-up (no hidden cards,
// stock and waste may still have cards). If so, it moves everything to
// foundations automatically.
func (m *Model) autoWin() {
	if !m.allFaceUp() {
		return
	}
	// Repeatedly move the lowest-rank available card to its foundation.
	for !isComplete(m.foundations) {
		moved := false
		if autoFoundation(&m.waste, &m.foundations) >= 0 {
			moved = true
			continue
		}
		for i := 0; i < 7; i++ {
			if autoFoundation(&m.tableau[i], &m.foundations) >= 0 {
				moved = true
				break
			}
		}
		if !moved {
			break // shouldn't happen if allFaceUp is correct, but safety
		}
	}
	if isComplete(m.foundations) {
		m.state = stateWon
	}
}

// allFaceUp reports whether every card not yet on a foundation is face-up.
func (m *Model) allFaceUp() bool {
	if len(m.stock.cards) > 0 {
		return false
	}
	for i := 0; i < 7; i++ {
		for _, c := range m.tableau[i].cards {
			if !c.FaceUp {
				return false
			}
		}
	}
	// Waste cards are always face-up by definition.
	return true
}

// draw turns one card from stock to waste. If stock is empty, recycles waste.
func (m *Model) draw() {
	if len(m.stock.cards) == 0 {
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
	layout := m.layout()
	if w, h := lipgloss.Size(layout); w > m.width || h > m.height {
		return m.styles.TooSmall(m.width, m.height, w, h)
	}
	return lipgloss.Place(m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		layout,
	)
}

func (m *Model) fits() bool {
	w, h := lipgloss.Size(m.layout())
	return w <= m.width && h <= m.height
}

func (m *Model) layout() string {
	help := m.renderHelp()
	board := m.renderBoard()

	title := m.styles.Title.Render("solitaire")
	boardWithTitle := lipgloss.JoinVertical(lipgloss.Left, title, "", board)

	pad := m.styles.NewStyle().PaddingBottom(1)
	return lipgloss.JoinHorizontal(lipgloss.Bottom,
		pad.Render(help), "  ", boardWithTitle,
	)
}

func (m *Model) renderBoard() string {
	topRow := m.renderTopRow()
	tableauRow := m.renderTableau()

	board := lipgloss.JoinVertical(lipgloss.Left, topRow, "", tableauRow)

	if m.state == stateWon {
		w := lipgloss.Width(board)
		h := lipgloss.Height(board)
		overlay := m.styles.Error.Render(" you win! ")
		if m.standing != nil {
			overlay = lipgloss.JoinVertical(lipgloss.Center,
				overlay, "", m.styles.Muted.Render(m.standing.String()))
		}
		board = lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center,
			overlay,
			lipgloss.WithWhitespaceChars(" "),
		)
	}

	return m.styles.Border.Render(board)
}

func (m *Model) indicator(active bool) string {
	if active {
		return m.styles.NewStyle().
			Width(cardW).
			Align(lipgloss.Center).
			Reverse(true).
			Bold(true).
			Render("▼")
	}
	return strings.Repeat(" ", cardW)
}

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
		col := lipgloss.JoinVertical(lipgloss.Left, m.indicator(cursor), card)
		parts = append(parts, col)
		if i < 3 {
			parts = append(parts, " ")
		}
	}

	parts = append(parts, "   ")

	stockCursor := m.cursor == cursorStock
	var stockCard string
	if len(m.stock.cards) == 0 {
		stockCard = renderEmpty(m.styles)
	} else {
		stockCard = renderBack(m.styles)
	}
	parts = append(parts, lipgloss.JoinVertical(lipgloss.Left, m.indicator(stockCursor), stockCard), " ")

	wasteCursor := m.cursor == cursorWaste
	wasteSel := m.sel != nil && m.sel.fromCursor == cursorWaste
	var wasteCard string
	if len(m.waste.cards) == 0 {
		wasteCard = renderEmpty(m.styles)
	} else {
		wasteCard = renderCard(*m.waste.top(), wasteCursor || wasteSel, m.styles)
	}
	parts = append(parts, lipgloss.JoinVertical(lipgloss.Left, m.indicator(wasteCursor), wasteCard))

	return lipgloss.JoinHorizontal(lipgloss.Top, parts...)
}

func (m *Model) renderTableau() string {
	cols := make([]string, 7)
	for i := 0; i < 7; i++ {
		cols[i] = m.renderTableauCol(i)
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, interleave(cols, " ")...)
}

func (m *Model) renderTableauCol(col int) string {
	p := &m.tableau[col]
	cursor := m.cursor == col
	selFrom := m.sel != nil && m.sel.fromCursor == col

	indicator := strings.Repeat(" ", cardW)
	if cursor {
		indicator = m.styles.NewStyle().
			Width(cardW).
			Align(lipgloss.Center).
			Reverse(true).
			Bold(true).
			Render("▼")
	}

	if len(p.cards) == 0 {
		e := renderEmpty(m.styles)
		return lipgloss.JoinVertical(lipgloss.Left, indicator, e)
	}

	var lines []string
	for i, c := range p.cards {
		isLast := i == len(p.cards)-1
		selected := selFrom && m.sel != nil && i >= m.sel.fromIdx
		cardStr := renderCard(c, selected, m.styles)
		cardLines := strings.Split(cardStr, "\n")

		if isLast {
			lines = append(lines, cardLines...)
		} else {
			lines = append(lines, cardLines[0])
		}
	}

	colContent := strings.Join(lines, "\n")
	return lipgloss.JoinVertical(lipgloss.Left, indicator, colContent)
}

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
	lines := []string{
		line("←→   move"),
		line("↑↓   top/tableau"),
		line("enter  pick/place"),
		line("a    auto-foundation"),
		line("d    draw"),
		line("r    restart"),
		line("esc  quit"),
	}
	if m.allFaceUp() && m.state == statePlaying {
		lines = append(lines, "", line("w    auto-win!"))
	}
	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}
