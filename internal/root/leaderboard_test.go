package root_test

import (
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/mahyarmirrashed/arcade/internal/game"
	"github.com/mahyarmirrashed/arcade/internal/leaderboard"
	"github.com/mahyarmirrashed/arcade/internal/lobby"
	"github.com/mahyarmirrashed/arcade/internal/root"
)

func openBoard(t *testing.T) *leaderboard.Board {
	t.Helper()
	b, err := leaderboard.Open(filepath.Join(t.TempDir(), "leaderboard.json"))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func rootFor(s leaderboard.Session) root.Model {
	return root.New(80, 24, game.NewRegistry(stubGame{name: "Stub"}), testStyles(), s)
}

func typeText(m root.Model, s string) root.Model {
	for _, r := range s {
		m = update(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	return m
}

var enter = tea.KeyMsg{Type: tea.KeyEnter}

func TestRoot_playerWithANewKeyChoosesAName(t *testing.T) {
	m := rootFor(openBoard(t).Session("key"))

	if m.State() != root.StateName {
		t.Fatalf("state = %v; want StateName", m.State())
	}
	if v := m.View(); !strings.Contains(v, "choose a name") {
		t.Errorf("view doesn't ask for a name:\n%s", v)
	}
}

func TestRoot_guestsAndReturningPlayersGoStraightToTheLobby(t *testing.T) {
	b := openBoard(t)
	if err := b.Session("key").SetName("alice"); err != nil {
		t.Fatal(err)
	}

	for name, s := range map[string]leaderboard.Session{
		"no leaderboard":   {},
		"guest":            b.Session(""),
		"returning player": b.Session("key"),
	} {
		if got := rootFor(s).State(); got != root.StateLobby {
			t.Errorf("%s: state = %v; want StateLobby", name, got)
		}
	}
}

func TestRoot_namePromptSavesTheName(t *testing.T) {
	b := openBoard(t)
	m := rootFor(b.Session("key"))

	m = update(typeText(m, "alice"), enter)

	if m.State() != root.StateLobby {
		t.Errorf("state = %v; want StateLobby after choosing a name", m.State())
	}
	if got := b.Session("key").Name(); got != "alice" {
		t.Errorf("saved name = %q; want alice", got)
	}
}

func TestRoot_namePromptExplainsRejectedNames(t *testing.T) {
	b := openBoard(t)
	if err := b.Session("other").SetName("alice"); err != nil {
		t.Fatal(err)
	}

	for name, tt := range map[string]struct{ typed, says string }{
		"invalid": {"a!", "letters, digits"},
		"taken":   {"alice", "taken"},
	} {
		t.Run(name, func(t *testing.T) {
			m := update(typeText(rootFor(b.Session("key")), tt.typed), enter)

			if m.State() != root.StateName {
				t.Errorf("state = %v; want to stay on the prompt", m.State())
			}
			if v := m.View(); !strings.Contains(v, tt.says) {
				t.Errorf("view doesn't explain the problem (%q):\n%s", tt.says, v)
			}
		})
	}
}

func TestRoot_escSkipsTheNamePrompt(t *testing.T) {
	b := openBoard(t)

	m := update(rootFor(b.Session("key")), tea.KeyMsg{Type: tea.KeyEsc})

	if m.State() != root.StateLobby || b.Session("key").Name() != "" {
		t.Errorf("state, name = %v, %q; want the lobby with no name", m.State(), b.Session("key").Name())
	}
}

func TestRoot_qIsALetterInTheNamePrompt(t *testing.T) {
	m := rootFor(openBoard(t).Session("key"))

	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})

	if next.(root.Model).State() != root.StateName {
		t.Error("q left the name prompt; want it typed")
	}
	if cmd != nil {
		if _, quit := cmd().(tea.QuitMsg); quit {
			t.Error("q quit the session; want it typed")
		}
	}
}

func TestRoot_recordsResultsAndTellsTheGame(t *testing.T) {
	b := openBoard(t)
	s := b.Session("key")
	if err := s.SetName("alice"); err != nil {
		t.Fatal(err)
	}
	m := update(rootFor(s), lobby.SelectMsg{Game: stubGame{name: "Stub"}})

	_, cmd := m.Update(game.ResultMsg{Game: "stub", Ranking: game.ByScore, Result: game.Result{Score: 5}})

	want := game.StandingMsg{Standing: game.Standing{Ranked: true, NewBest: true, Rank: 1, Of: 1, Best: 5}}
	if cmd == nil || cmd() != want {
		t.Errorf("cmd() = %v; want %#v", cmd, want)
	}
	if top := s.Top("stub", game.ByScore, 10); len(top) != 1 || top[0].Value != "5" {
		t.Errorf("leaderboard = %+v; want alice with 5", top)
	}
}

func TestRoot_guestsAreToldTheyAreUnranked(t *testing.T) {
	m := update(rootFor(openBoard(t).Session("")), lobby.SelectMsg{Game: stubGame{name: "Stub"}})

	_, cmd := m.Update(game.ResultMsg{Game: "stub", Result: game.Result{Score: 5}})

	if cmd == nil || cmd() != (game.StandingMsg{}) {
		t.Errorf("cmd() = %v; want an unranked StandingMsg", cmd)
	}
}

func TestRoot_resultArrivingAfterQuitIsStillRecorded(t *testing.T) {
	// A game can finish and quit in the same instant; its result may arrive
	// once the player is back in the lobby.
	b := openBoard(t)
	s := b.Session("key")
	if err := s.SetName("alice"); err != nil {
		t.Fatal(err)
	}
	m := rootFor(s)

	m.Update(game.ResultMsg{Game: "stub", Ranking: game.ByScore, Result: game.Result{Score: 9}})

	if top := s.Top("stub", game.ByScore, 10); len(top) != 1 {
		t.Errorf("leaderboard = %+v; want the result recorded", top)
	}
}
