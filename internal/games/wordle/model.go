package wordle

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

type keyMap struct {
	quit key.Binding
}

var keys = keyMap{
	quit: key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
}

// state is the game phase.
type state int

const (
	statePlaying state = iota
	stateWon
	stateLost
)

// Model is the Bubbletea model for Wordle.
type Model struct {
	answer  string
	rows    [maxGuess]row
	current int    // index of the row being typed
	input   string // letters typed so far for the current row
	state   state
	notice  string // ephemeral message (e.g. "not a word")
	width   int
	height  int
	rng     *rand.Rand
	styles  ui.Styles
}

// - game.Game implementation -

func (m *Model) Name() string        { return "Wordle" }
func (m *Model) Description() string { return "guess the five-letter word" }

func (m *Model) New(width, height int, styles ui.Styles) tea.Model {
	return newModel(width, height, styles)
}

func newModel(width, height int, styles ui.Styles) *Model {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	return &Model{
		answer: randomAnswer(rng),
		width:  width,
		height: height,
		rng:    rng,
		state:  statePlaying,
		styles: styles,
	}
}

// - tea.Model implementation -

func (m *Model) Init() tea.Cmd { return nil }

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Intercept restart before anything else.
	if rm, ok := msg.(replaceMsg); ok {
		return rm.m, nil
	}
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case tea.KeyMsg:
		return m, m.handleKey(msg)
	}
	return m, nil
}

func (m *Model) handleKey(msg tea.KeyMsg) tea.Cmd {
	// r restarts only when the game is over.
	if msg.String() == "r" && m.state != statePlaying {
		fresh := newModel(m.width, m.height, m.styles)
		return func() tea.Msg { return replaceMsg{fresh} }
	}

	if m.state != statePlaying {
		return nil
	}

	m.notice = ""

	switch msg.String() {
	case "backspace", "ctrl+h":
		if len(m.input) > 0 {
			m.input = m.input[:len(m.input)-1]
		}
	case "enter":
		m.submitGuess()
	default:
		// Accept a-z only.
		if len(msg.String()) == 1 {
			ch := rune(msg.String()[0])
			if ch >= 'a' && ch <= 'z' && len(m.input) < wordLen {
				m.input += string(ch)
			}
		}
	}
	return nil
}

// replaceMsg is emitted by handleKey("r") to replace the model with a fresh one.
// The root model passes it through; Update intercepts it before anything else.
type replaceMsg struct{ m *Model }

func (m *Model) submitGuess() {
	if len(m.input) < wordLen {
		m.notice = "not enough letters"
		return
	}
	if !isValid(m.input) {
		m.notice = "not in word list"
		return
	}

	hints := evaluate(m.input, m.answer)
	var letters [wordLen]rune
	for i, ch := range m.input {
		letters[i] = ch
	}
	m.rows[m.current] = row{letters: letters, hints: hints}
	m.input = ""

	if isWin(hints) {
		m.state = stateWon
		return
	}

	m.current++
	if m.current >= maxGuess {
		m.state = stateLost
	}
}

// - View -

// hintStyle returns the Lip Gloss style for a letter tile given its hint.
func (m *Model) hintStyle(h hint) lipgloss.Style {
	base := m.styles.NewStyle().
		Width(4).
		Align(lipgloss.Center).
		Bold(true)
	switch h {
	case hintCorrect:
		return base.Foreground(ui.ColorInverse).Background(ui.Spectrum[3]) // green
	case hintPresent:
		return base.Foreground(ui.ColorInverse).Background(ui.Spectrum[2]) // yellow
	default:
		return base.Foreground(ui.ColorMuted)
	}
}

func (m *Model) emptyStyle() lipgloss.Style {
	return m.styles.NewStyle().
		Width(4).
		Align(lipgloss.Center).
		Foreground(ui.ColorMuted)
}

func (m *Model) activeStyle() lipgloss.Style {
	return m.styles.NewStyle().
		Width(4).
		Align(lipgloss.Center).
		Foreground(ui.ColorText).
		Bold(true)
}

func (m *Model) View() string {
	help := m.renderHelp()
	board := m.renderBoard()

	title := m.styles.Title.Render("wordle")
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
	var rowStrs [maxGuess]string

	for r := 0; r < maxGuess; r++ {
		var cells [wordLen]string

		switch {
		case r < m.current:
			// Submitted row - show hints.
			for c := 0; c < wordLen; c++ {
				ch := strings.ToUpper(string(m.rows[r].letters[c]))
				cells[c] = m.hintStyle(m.rows[r].hints[c]).Render(ch)
			}
		case r == m.current && m.state == statePlaying:
			// Active row - show typed letters.
			for c := 0; c < wordLen; c++ {
				if c < len(m.input) {
					ch := strings.ToUpper(string(m.input[c]))
					cells[c] = m.activeStyle().Render(ch)
				} else {
					cells[c] = m.emptyStyle().Render("·")
				}
			}
		default:
			// Future row - empty.
			for c := 0; c < wordLen; c++ {
				cells[c] = m.emptyStyle().Render("·")
			}
		}

		rowStrs[r] = lipgloss.JoinHorizontal(lipgloss.Center,
			cells[0], cells[1], cells[2], cells[3], cells[4],
		)
	}

	inner := lipgloss.JoinVertical(lipgloss.Left,
		rowStrs[0], rowStrs[1], rowStrs[2],
		rowStrs[3], rowStrs[4], rowStrs[5],
	)

	// Status overlay.
	var overlay string
	switch m.state {
	case stateWon:
		overlay = m.styles.Error.Render(" you got it! ")
	case stateLost:
		overlay = m.styles.Error.Render(" " + strings.ToUpper(m.answer) + " ")
	}
	if overlay != "" {
		w := lipgloss.Width(inner)
		h := lipgloss.Height(inner)
		inner = lipgloss.Place(w, h,
			lipgloss.Center, lipgloss.Center,
			overlay,
			lipgloss.WithWhitespaceChars(" "),
		)
	}

	notice := ""
	if m.notice != "" {
		notice = "\n" + m.styles.Muted.Render(m.notice)
	}

	return m.styles.Border.Render(inner) + notice
}

func (m *Model) renderHelp() string {
	line := func(s string) string { return m.styles.Muted.Render(s) }
	return lipgloss.JoinVertical(lipgloss.Left,
		line("a-z  type"),
		line("⌫    delete"),
		line("enter  guess"),
		line("r    restart"),
		line("q    quit"),
		"",
		m.styles.Muted.Render("green  = correct"),
		m.styles.Muted.Render("yellow = wrong place"),
		m.styles.Muted.Render("grey   = absent"),
	)
}
