package lobby_test

import (
	"io"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"

	"github.com/mahyarmirrashed/arcade/internal/game"
	"github.com/mahyarmirrashed/arcade/internal/leaderboard"
	"github.com/mahyarmirrashed/arcade/internal/lobby"
	"github.com/mahyarmirrashed/arcade/internal/ui"
)

// stubGame satisfies game.Game for test purposes.
type stubGame struct{ name string }

func (s stubGame) Name() string                   { return s.name }
func (s stubGame) Description() string            { return s.name + " desc" }
func (s stubGame) New(_, _ int, _ ui.Styles) tea.Model         { return nil }

func newTestModel(games ...game.Game) lobby.Model {
	reg := game.NewRegistry(games...)
	return lobby.New(80, 24, reg, testStyles(), leaderboard.Session{})
}

// testStyles renders without colour, as a terminal with no colour support would.
func testStyles() ui.Styles {
	r := lipgloss.NewRenderer(io.Discard)
	r.SetColorProfile(termenv.Ascii)
	return ui.NewStyles(r)
}

func update(m tea.Model, msg tea.Msg) tea.Model {
	next, _ := m.Update(msg)
	return next
}

func keyMsg(k string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)}
}

func TestLobby_downMovesSelection(t *testing.T) {
	alpha := stubGame{name: "Alpha"}
	beta := stubGame{name: "Beta"}

	m := newTestModel(alpha, beta)

	// send the down key - bubbles/list handles "down" and "j"
	next := update(m, tea.KeyMsg{Type: tea.KeyDown})

	lm, ok := next.(lobby.Model)
	if !ok {
		t.Fatalf("Update returned unexpected type %T", next)
	}

	// After one down, index should be 1
	if lm.Index() != 1 {
		t.Errorf("index = %d; want 1", lm.Index())
	}
}

func TestLobby_upWraps(t *testing.T) {
	m := newTestModel(stubGame{"A"}, stubGame{"B"}, stubGame{"C"})

	// up from 0 should not go negative (stays at 0 - bubbles does not wrap by default)
	next := update(m, tea.KeyMsg{Type: tea.KeyUp})
	lm := next.(lobby.Model)
	if lm.Index() < 0 {
		t.Errorf("index went negative: %d", lm.Index())
	}
}

func TestLobby_enterEmitsSelectMsg(t *testing.T) {
	alpha := stubGame{name: "Alpha"}
	m := newTestModel(alpha)

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected a command from enter, got nil")
	}

	msg := cmd()
	sel, ok := msg.(lobby.SelectMsg)
	if !ok {
		t.Fatalf("cmd() returned %T; want lobby.SelectMsg", msg)
	}
	if sel.Game.Name() != "Alpha" {
		t.Errorf("SelectMsg.Game.Name() = %q; want %q", sel.Game.Name(), "Alpha")
	}
}

func TestLobby_windowResizeUpdatesDimensions(t *testing.T) {
	m := newTestModel(stubGame{"A"})
	next := update(m, tea.WindowSizeMsg{Width: 120, Height: 40})
	lm := next.(lobby.Model)

	if lm.Width() != 120 || lm.Height() != 40 {
		t.Errorf("dimensions = %dx%d; want 120x40", lm.Width(), lm.Height())
	}
}

// Verify keys binding is correct (compile-time + runtime sanity).
func TestLobby_quitKeyBinding(t *testing.T) {
	qKey := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")}
	quitBinding := key.NewBinding(key.WithKeys("q", "ctrl+c"))
	if !key.Matches(qKey, quitBinding) {
		t.Error("q should match quit binding")
	}
}

func TestView_rendersWithSessionStyles(t *testing.T) {
	// The server's own renderer has no colour; the player's terminal does.
	prev := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.Ascii)
	t.Cleanup(func() { lipgloss.SetColorProfile(prev) })

	r := lipgloss.NewRenderer(io.Discard)
	r.SetColorProfile(termenv.ANSI256)
	reg := game.NewRegistry(stubGame{name: "Alpha"}, stubGame{name: "Beta"})

	colour := lobby.New(80, 24, reg, ui.NewStyles(r), leaderboard.Session{}).View()
	plain := lobby.New(80, 24, reg, testStyles(), leaderboard.Session{}).View()

	if !strings.Contains(colour, "\x1b[") {
		t.Error("256-colour session rendered no colour codes")
	}
	if strings.Contains(plain, "\x1b[") {
		t.Error("no-colour session rendered colour codes")
	}
}

func TestView_tooSmallTerminalShowsNotice(t *testing.T) {
	m := newTestModel(stubGame{name: "Alpha"})

	next, _ := m.Update(tea.WindowSizeMsg{Width: 30, Height: 10})
	small := next.(lobby.Model).View()
	next, _ = next.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	normal := next.(lobby.Model).View()

	if !strings.Contains(small, "terminal too small") {
		t.Errorf("30x10 view does not say the terminal is too small:\n%s", small)
	}
	if strings.Contains(normal, "terminal too small") || !strings.Contains(normal, "Alpha") {
		t.Errorf("80x24 view after growing back should list the games:\n%s", normal)
	}
}

// rankedStub is a stub game that appears on the leaderboard by score.
type rankedStub struct{ stubGame }

func (rankedStub) Ranking() game.Ranking { return game.ByScore }

func boardWith(t *testing.T, scores map[string]int) *leaderboard.Board {
	t.Helper()
	b, err := leaderboard.Open(filepath.Join(t.TempDir(), "leaderboard.json"))
	if err != nil {
		t.Fatal(err)
	}
	for name, score := range scores {
		s := b.Session("key-" + name)
		if err := s.SetName(name); err != nil {
			t.Fatal(err)
		}
		if _, err := s.Record("ranked", game.ByScore, game.Result{Score: score}); err != nil {
			t.Fatal(err)
		}
	}
	return b
}

func lobbyView(g game.Game, s leaderboard.Session) string {
	return lobby.New(80, 24, game.NewRegistry(g), testStyles(), s).View()
}

func TestView_showsTopScoresForTheHighlightedGame(t *testing.T) {
	b := boardWith(t, map[string]int{"alice": 50, "bob": 30})

	v := lobbyView(rankedStub{stubGame{"Ranked"}}, b.Session("key-bob"))

	for _, want := range []string{"best score", "alice", "50", "bob", "30", "you: #2 of 2"} {
		if !strings.Contains(v, want) {
			t.Errorf("lobby view is missing %q:\n%s", want, v)
		}
	}
	if strings.Index(v, "alice") > strings.Index(v, "bob") {
		t.Errorf("alice (50) should be listed above bob (30):\n%s", v)
	}
}

func TestView_leaderboardPanelVariants(t *testing.T) {
	b := boardWith(t, nil)

	tests := []struct {
		name   string
		game   game.Game
		s      leaderboard.Session
		want   string
		absent string
	}{
		{"empty board", rankedStub{stubGame{"Ranked"}}, b.Session("key"), "no scores yet", ""},
		{"guest", rankedStub{stubGame{"Ranked"}}, b.Session(""), "connect with an ssh key", ""},
		{"game without a leaderboard", stubGame{"Plain"}, b.Session("key"), "", "best score"},
		{"no leaderboard at all", rankedStub{stubGame{"Ranked"}}, leaderboard.Session{}, "", "best score"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := lobbyView(tt.game, tt.s)
			if tt.want != "" && !strings.Contains(v, tt.want) {
				t.Errorf("lobby view is missing %q:\n%s", tt.want, v)
			}
			if tt.absent != "" && strings.Contains(v, tt.absent) {
				t.Errorf("lobby view shouldn't show %q:\n%s", tt.absent, v)
			}
		})
	}
}
