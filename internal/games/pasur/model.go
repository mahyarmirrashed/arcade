package pasur

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/mahyarmirrashed/arcade/internal/game"
	"github.com/mahyarmirrashed/arcade/internal/ui"
)

// Compile-time check: Model must satisfy game.Game.
var _ game.Game = (*Model)(nil)

// modelState is the per-player UI phase.
type modelState int

const (
	stateWaiting  modelState = iota // created room, showing code
	stateJoining                    // typing a code to join
	statePlaying                    // in game
	stateGrace                      // opponent disconnected
	stateOver                       // game finished
)

type keyMap struct {
	left  key.Binding
	right key.Binding
	pick  key.Binding
	quit  key.Binding
}

var keys = keyMap{
	left:  key.NewBinding(key.WithKeys("left", "h"), key.WithHelp("←/h", "prev card")),
	right: key.NewBinding(key.WithKeys("right", "l"), key.WithHelp("→/l", "next card")),
	pick:  key.NewBinding(key.WithKeys("enter", " "), key.WithHelp("enter", "select/play")),
	quit:  key.NewBinding(key.WithKeys("esc", "ctrl+c"), key.WithHelp("esc", "quit")),
}

// Model is the per-player Bubbletea model for Pasur.
type Model struct {
	seat    playerIndex
	room    *Room  // pointer into broker; read under broker lock via snapshots
	code    string // room code this player belongs to

	state     modelState
	handCursor int    // index into this player's hand
	capCursor  int    // index into CaptureOpts when multiple options exist
	joinInput  textinput.Model
	notice     string // transient error message
	snapshot   roomSnapshot // last received state snapshot

	width  int
	height int
	styles ui.Styles
}

// roomSnapshot is a safe copy of room state for rendering (no lock needed).
type roomSnapshot struct {
	phase        roomPhase
	pool         []Card
	myHand       []Card
	oppHandSize  int
	myCapture    []Card
	oppCapture   []Card
	mySurs       int
	oppSurs      int
	myScore      int
	oppScore     int
	turn         playerIndex
	selectedCard *int
	captureOpts  [][]int
	graceEnds    string
}

// - game.Game implementation -

func (m *Model) Name() string        { return "Pasur" }
func (m *Model) Description() string { return "persian fishing card game, 2 players" }

func (m *Model) New(width, height int, styles ui.Styles) tea.Model {
	return newModel(width, height, styles)
}

func newModel(width, height int, styles ui.Styles) *Model {
	// Create a room immediately; the player sees their code straight away.
	room := Broker.Create()

	ji := textinput.New()
	ji.Placeholder = "XKCD"
	ji.CharLimit = 4
	ji.Width = 6

	return &Model{
		seat:   player0,
		room:   room,
		code:   room.Code,
		state:  stateWaiting,
		width:  width,
		height: height,
		styles: styles,
		joinInput: ji,
		snapshot: snapshotRoom(room, player0),
	}
}

// - tea.Model implementation -

func (m *Model) Init() tea.Cmd {
	return m.waitForUpdate()
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case stateMsg:
		m.snapshot = snapshotRoom(msg.room, m.seat)
		m.syncState()
		// Clamp cursor to valid range after hand size may have changed.
		if n := len(m.snapshot.myHand); n > 0 && m.handCursor >= n {
			m.handCursor = n - 1
		}
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
		m.snapshot = snapshotRoom(msg.room, player1)
		return m, m.waitForUpdate()

	case tea.KeyMsg:
		return m, m.handleKey(msg)
	}
	return m, nil
}

// syncState updates m.state from the latest snapshot.
func (m *Model) syncState() {
	switch m.snapshot.phase {
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

// waitForUpdate returns a tea.Cmd that blocks until the room broadcasts.
func (m *Model) waitForUpdate() tea.Cmd {
	ch := m.room.channels[m.seat]
	if ch == nil {
		return nil
	}
	return func() tea.Msg {
		return <-ch
	}
}

func (m *Model) handleKey(msg tea.KeyMsg) tea.Cmd {
	if msg.String() == "r" && m.state == stateOver {
		fresh := newModel(m.width, m.height, m.styles)
		return func() tea.Msg { return replaceMsg{fresh} }
	}

	switch m.state {
	case stateWaiting:
		// While waiting the player can switch to join-mode to type a code.
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
		if m.snapshot.turn != m.seat {
			return nil // not our turn
		}
		myHand := m.snapshot.myHand
		if len(myHand) == 0 {
			return nil
		}
		switch {
		case key.Matches(msg, keys.left):
			if m.handCursor > 0 {
				m.handCursor--
			}
			Broker.Select(m.code, m.seat, &m.handCursor)
		case key.Matches(msg, keys.right):
			if m.handCursor < len(myHand)-1 {
				m.handCursor++
			}
			Broker.Select(m.code, m.seat, &m.handCursor)
		case key.Matches(msg, keys.pick):
			return m.confirmPlay()
		}
	}
	return nil
}

// confirmPlay decides whether we need to choose among capture options or
// can play directly.
func (m *Model) confirmPlay() tea.Cmd {
	opts := m.snapshot.captureOpts
	if len(opts) == 0 {
		// No captures possible - play card to pool.
		if canCapture(m.snapshot.myHand[m.handCursor], m.snapshot.pool) {
			m.notice = "you must capture"
			return nil
		}
		return playCard(m.code, m.seat, m.handCursor, nil)
	}
	if len(opts) == 1 {
		return playCard(m.code, m.seat, m.handCursor, opts[0])
	}
	// Multiple options - cycle through with left/right, confirm with enter.
	// For simplicity, auto-select the option that captures the most cards.
	best := opts[0]
	for _, o := range opts[1:] {
		if len(o) > len(best) {
			best = o
		}
	}
	return playCard(m.code, m.seat, m.handCursor, best)
}

// - async commands -

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

func playCard(code string, seat playerIndex, handIdx int, captureIdxs []int) tea.Cmd {
	return func() tea.Msg {
		Broker.Play(code, seat, handIdx, captureIdxs)
		return nil
	}
}

// - View -

func (m *Model) View() string {
	var content string
	switch m.state {
	case stateWaiting:
		content = m.viewWaiting()
	case stateJoining:
		content = m.viewJoining()
	case statePlaying, stateGrace:
		content = m.viewGame()
	case stateOver:
		content = m.viewOver()
	default:
		content = m.viewWaiting()
	}

	return lipgloss.Place(m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		content,
	)
}

func (m *Model) viewWaiting() string {
	lines := lipgloss.JoinVertical(lipgloss.Left,
		m.styles.Title.Render("pasur"),
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
	return lines
}

func (m *Model) viewJoining() string {
	notice := ""
	if m.notice != "" {
		notice = m.styles.Error.Render(" " + m.notice + " ")
	}
	return lipgloss.JoinVertical(lipgloss.Left,
		m.styles.Title.Render("pasur"),
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
	s := m.snapshot
	help := m.renderHelp()
	board := m.renderBoard()
	side := m.renderSide()

	title := m.styles.Title.Render("pasur")
	boardWithTitle := lipgloss.JoinVertical(lipgloss.Left, title, "", board)

	pad := m.styles.NewStyle().PaddingBottom(1)
	layout := lipgloss.JoinHorizontal(lipgloss.Bottom,
		pad.Render(help), "  ", boardWithTitle, "  ", pad.Render(side),
	)

	if m.state == stateGrace {
		w := lipgloss.Width(layout)
		h := lipgloss.Height(layout)
		overlay := m.styles.Error.Render(" opponent disconnected ") + "\n" +
			m.styles.Muted.Render("waiting "+s.graceEnds+"...")
		layout = lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center,
			overlay, lipgloss.WithWhitespaceChars(" "))
	}

	return layout
}

func (m *Model) renderBoard() string {
	s := m.snapshot

	// Opponent's hand (face-down).
	oppRow := m.renderOppHand(s.oppHandSize)

	// Pool: only highlight captures for the player whose turn it is.
	var captureOpts [][]int
	if s.turn == m.seat {
		captureOpts = s.captureOpts
	}
	poolRow := m.renderPool(s.pool, captureOpts)

	// My hand.
	myRow := m.renderMyHand(s.myHand, s.selectedCard)

	turnNote := ""
	if s.turn == m.seat {
		turnNote = m.styles.Accent.Render("your turn")
	} else {
		turnNote = m.styles.Muted.Render("opponent's turn")
	}

	notice := ""
	if m.notice != "" {
		notice = m.styles.Error.Render(" " + m.notice + " ")
		m.notice = ""
	}

	inner := lipgloss.JoinVertical(lipgloss.Left,
		m.styles.Muted.Render("opponent: "+strconv.Itoa(s.oppHandSize)+" cards"),
		oppRow,
		"",
		m.styles.Subtitle.Render("pool"),
		poolRow,
		"",
		m.styles.Subtitle.Render("your hand"),
		myRow,
		"",
		turnNote,
		notice,
	)
	return m.styles.Border.Render(inner)
}

func (m *Model) renderOppHand(count int) string {
	if count == 0 {
		return m.styles.Muted.Render("(empty)")
	}
	cards := make([]string, count)
	back := m.styles.NewStyle().Foreground(ui.ColorMuted).Render("[?]")
	for i := range cards {
		cards[i] = back
	}
	return strings.Join(cards, " ")
}

func (m *Model) renderPool(pool []Card, captureOpts [][]int) string {
	if len(pool) == 0 {
		return m.styles.Muted.Render("(empty)")
	}

	// Build a set of pool indices that are highlighted (part of a capture option).
	highlighted := make(map[int]bool)
	for _, opt := range captureOpts {
		for _, i := range opt {
			highlighted[i] = true
		}
	}

	parts := make([]string, len(pool))
	for i, c := range pool {
		st := m.styles.NewStyle().Foreground(ui.ColorSubtle)
		if highlighted[i] {
			st = m.styles.NewStyle().Foreground(ui.ColorText).Bold(true).Reverse(true)
		}
		parts[i] = st.Render(c.String())
	}
	return strings.Join(parts, " ")
}

func (m *Model) renderMyHand(hand []Card, selected *int) string {
	if len(hand) == 0 {
		return m.styles.Muted.Render("(empty)")
	}
	parts := make([]string, len(hand))
	for i, c := range hand {
		st := m.styles.NewStyle()
		if c.Suit.isRed() {
			st = st.Foreground(ui.Spectrum[0])
		} else {
			st = st.Foreground(ui.ColorText)
		}
		if i == m.handCursor {
			st = st.Reverse(true).Bold(true)
		}
		parts[i] = st.Render(c.String())
	}
	return strings.Join(parts, " ")
}

func (m *Model) renderSide() string {
	s := m.snapshot
	return lipgloss.JoinVertical(lipgloss.Left,
		m.styles.Subtitle.Render("score"),
		fmt.Sprintf("you  %d", s.myScore),
		fmt.Sprintf("opp  %d", s.oppScore),
		"",
		m.styles.Subtitle.Render("surs"),
		fmt.Sprintf("you  %d", s.mySurs),
		fmt.Sprintf("opp  %d", s.oppSurs),
		"",
		m.styles.Subtitle.Render("target"),
		strconv.Itoa(winScore),
		"",
		m.styles.Subtitle.Render("code"),
		m.styles.Accent.Render(m.code),
	)
}

func (m *Model) renderHelp() string {
	line := func(s string) string { return m.styles.Muted.Render(s) }
	return lipgloss.JoinVertical(lipgloss.Left,
		line("←→   select card"),
		line("enter  play"),
		line("esc   quit"),
	)
}

func (m *Model) viewOver() string {
	s := m.snapshot
	var result string
	if s.myScore > s.oppScore {
		result = m.styles.Accent.Render("you win!")
	} else if s.myScore < s.oppScore {
		result = m.styles.Error.Render("you lose")
	} else {
		result = m.styles.Subtitle.Render("draw")
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		m.styles.Title.Render("pasur"),
		"",
		result,
		"",
		fmt.Sprintf("you %d  ·  opponent %d", s.myScore, s.oppScore),
		"",
		m.styles.Muted.Render("r to play again · esc to quit"),
	)
}

// snapshotRoom takes a safe copy of room state for a given seat.
func snapshotRoom(r *Room, seat playerIndex) roomSnapshot {
	opp := seat.opponent()
	s := roomSnapshot{
		phase:       r.Phase,
		pool:        append([]Card(nil), r.Game.Pool...),
		myHand:      append([]Card(nil), r.Game.Hands[seat]...),
		oppHandSize: len(r.Game.Hands[opp]),
		myCapture:   append([]Card(nil), r.Game.Captured[seat]...),
		oppCapture:  append([]Card(nil), r.Game.Captured[opp]...),
		mySurs:      r.Game.Surs[seat],
		oppSurs:     r.Game.Surs[opp],
		myScore:     r.Game.Scores[seat],
		oppScore:    r.Game.Scores[opp],
		turn:        r.Game.Turn,
		captureOpts: r.CaptureOpts,
	}
	if r.SelectedCard != nil {
		idx := *r.SelectedCard
		s.selectedCard = &idx
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
