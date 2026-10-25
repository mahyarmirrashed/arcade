package leaderboard

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/mahyarmirrashed/arcade/internal/game"
)

// testBoard opens a board in a fresh directory with a clock that advances one
// second per reading, so every result gets a distinct, ordered timestamp.
func testBoard(t *testing.T) (*Board, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "data", "leaderboard.json")
	b, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	clock := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	b.now = func() time.Time {
		clock = clock.Add(time.Second)
		return clock
	}
	return b, path
}

// named returns a session for id with the given display name already set.
func named(t *testing.T, b *Board, id, name string) Session {
	t.Helper()
	s := b.Session(id)
	if err := s.SetName(name); err != nil {
		t.Fatalf("SetName(%q): %v", name, err)
	}
	return s
}

func mustRecord(t *testing.T, s Session, gameID string, r game.Ranking, res game.Result) game.Standing {
	t.Helper()
	st, err := s.Record(gameID, r, res)
	if err != nil {
		t.Fatalf("Record: %v", err)
	}
	return st
}

func names(entries []Entry) []string {
	out := make([]string, len(entries))
	for i, e := range entries {
		out[i] = e.Name
	}
	return out
}

func TestOpen_missingFileIsEmptyBoard(t *testing.T) {
	b, _ := testBoard(t)

	if got := b.Session("key").Top("snake", game.ByScore, 10); len(got) != 0 {
		t.Errorf("Top on a new board = %v; want empty", got)
	}
}

func TestOpen_refusesCorruptFile(t *testing.T) {
	// Starting with an empty board would overwrite everyone's scores on the
	// next save.
	path := filepath.Join(t.TempDir(), "leaderboard.json")
	if err := os.WriteFile(path, []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := Open(path); err == nil {
		t.Error("Open on a corrupt file succeeded; want an error")
	}
}

func TestOpen_refusesUnknownVersion(t *testing.T) {
	path := filepath.Join(t.TempDir(), "leaderboard.json")
	if err := os.WriteFile(path, []byte(`{"version": 99}`), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := Open(path); err == nil {
		t.Error("Open on an unknown version succeeded; want an error")
	}
}

func TestSetName_validation(t *testing.T) {
	tests := []struct {
		name    string
		want    error
		comment string
	}{
		{"alice", nil, ""},
		{"a_b-9", nil, ""},
		{"abcdefghijkl", nil, "12 characters"},
		{"ab", ErrNameInvalid, "too short"},
		{"abcdefghijklm", ErrNameInvalid, "too long"},
		{"al ice", ErrNameInvalid, "space"},
		{"al\x1b[31mice", ErrNameInvalid, "terminal escape code"},
		{"ålice", ErrNameInvalid, "non-ASCII"},
		{"Guest", ErrNameTaken, "reserved for players without a key"},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%q", tt.name), func(t *testing.T) {
			b, _ := testBoard(t)

			if err := b.Session("key").SetName(tt.name); !errors.Is(err, tt.want) {
				t.Errorf("SetName(%q) = %v; want %v %s", tt.name, err, tt.want, tt.comment)
			}
		})
	}
}

func TestSetName_namesAreUniqueIgnoringCase(t *testing.T) {
	b, _ := testBoard(t)
	named(t, b, "key-a", "alice")

	if err := b.Session("key-b").SetName("ALICE"); !errors.Is(err, ErrNameTaken) {
		t.Errorf("second player taking ALICE = %v; want ErrNameTaken", err)
	}
	if err := b.Session("key-a").SetName("Alice"); err != nil {
		t.Errorf("owner re-casing their own name = %v; want nil", err)
	}
}

func TestNeedsName(t *testing.T) {
	b, _ := testBoard(t)
	s := b.Session("key")

	if !s.NeedsName() {
		t.Error("new key: NeedsName = false; want true")
	}
	if err := s.SetName("alice"); err != nil {
		t.Fatal(err)
	}
	if s.NeedsName() || s.Name() != "alice" {
		t.Errorf("after SetName: NeedsName, Name = %v, %q; want false, alice", s.NeedsName(), s.Name())
	}
	if (Session{}).NeedsName() || b.Session("").NeedsName() {
		t.Error("guest: NeedsName = true; want false (guests have nothing to name)")
	}
}

func TestRecord_guestsAndUnnamedPlayersAreUnranked(t *testing.T) {
	b, _ := testBoard(t)

	for name, s := range map[string]Session{
		"no board":       {},
		"no key":         b.Session(""),
		"no name chosen": b.Session("key"),
	} {
		st := mustRecord(t, s, "snake", game.ByScore, game.Result{Score: 50})
		if st.Ranked {
			t.Errorf("%s: Standing.Ranked = true; want false", name)
		}
	}
	if got := b.Session("x").Top("snake", game.ByScore, 10); len(got) != 0 {
		t.Errorf("Top = %v; want nothing recorded", got)
	}
}

func TestRecord_byScore(t *testing.T) {
	b, _ := testBoard(t)
	alice := named(t, b, "a", "alice")
	bob := named(t, b, "b", "bob")

	if st := mustRecord(t, alice, "snake", game.ByScore, game.Result{Score: 10}); st != (game.Standing{Ranked: true, NewBest: true, Rank: 1, Of: 1}) {
		t.Errorf("first result: %+v; want new best, #1 of 1", st)
	}
	if st := mustRecord(t, bob, "snake", game.ByScore, game.Result{Score: 30}); st != (game.Standing{Ranked: true, NewBest: true, Rank: 1, Of: 2}) {
		t.Errorf("bob beats alice: %+v; want new best, #1 of 2", st)
	}
	if st := mustRecord(t, alice, "snake", game.ByScore, game.Result{Score: 5}); st != (game.Standing{Ranked: true, Rank: 2, Of: 2}) {
		t.Errorf("alice scores lower: %+v; want no new best, #2 of 2", st)
	}
	if st := mustRecord(t, alice, "snake", game.ByScore, game.Result{Score: 30}); !st.NewBest || st.Rank != 2 {
		t.Errorf("alice ties bob later: %+v; want her new best, but #2 (bob's equal score came first)", st)
	}

	top := alice.Top("snake", game.ByScore, 10)
	if got := names(top); len(got) != 2 || got[0] != "bob" || got[1] != "alice" {
		t.Fatalf("Top = %v; want [bob alice]", got)
	}
	if top[0].Value != "30" || top[0].Rank != 1 || top[0].You || !top[1].You {
		t.Errorf("Top entries = %+v; want bob #1 30, alice marked as you", top)
	}
}

func TestRecord_byStreak(t *testing.T) {
	b, _ := testBoard(t)
	alice := named(t, b, "a", "alice")
	win := func(guesses int) game.Result { return game.Result{Won: true, Guesses: guesses} }

	if st := mustRecord(t, alice, "wordle", game.ByStreak, game.Result{}); st.Rank != 0 || st.Of != 0 {
		t.Errorf("a loss before any win: %+v; want not ranked yet", st)
	}
	mustRecord(t, alice, "wordle", game.ByStreak, win(4))
	if st := mustRecord(t, alice, "wordle", game.ByStreak, win(3)); !st.NewBest || st.Rank != 1 {
		t.Errorf("second win in a row: %+v; want new best streak", st)
	}
	mustRecord(t, alice, "wordle", game.ByStreak, game.Result{}) // loss resets the streak
	if st := mustRecord(t, alice, "wordle", game.ByStreak, win(2)); st.NewBest {
		t.Errorf("one win after a loss: %+v; want no new best (best streak is still 2)", st)
	}
	if got := alice.Top("wordle", game.ByStreak, 1)[0].Value; got != "2" {
		t.Errorf("best streak shown as %q; want 2", got)
	}
}

func TestRecord_byStreakTieBreaksOnGuesses(t *testing.T) {
	b, _ := testBoard(t)
	alice := named(t, b, "a", "alice")
	bob := named(t, b, "b", "bob")

	// Same streak of 2; bob averages 3 guesses, alice 5.
	for _, g := range []int{5, 5} {
		mustRecord(t, alice, "wordle", game.ByStreak, game.Result{Won: true, Guesses: g})
	}
	for _, g := range []int{3, 3} {
		mustRecord(t, bob, "wordle", game.ByStreak, game.Result{Won: true, Guesses: g})
	}

	if got := names(alice.Top("wordle", game.ByStreak, 10)); got[0] != "bob" {
		t.Errorf("Top = %v; want bob first on fewer guesses", got)
	}
}

func TestRecord_byTime(t *testing.T) {
	b, _ := testBoard(t)
	alice := named(t, b, "a", "alice")
	won := func(d time.Duration) game.Result { return game.Result{Won: true, Duration: d} }

	if st := mustRecord(t, alice, "minesweeper", game.ByTime, game.Result{}); st.Rank != 0 {
		t.Errorf("a loss before any win: %+v; want not ranked yet", st)
	}
	if st := mustRecord(t, alice, "minesweeper", game.ByTime, won(95*time.Second)); !st.NewBest || st.Rank != 1 {
		t.Errorf("first win: %+v; want new best, #1", st)
	}
	if st := mustRecord(t, alice, "minesweeper", game.ByTime, won(2*time.Minute)); st.NewBest {
		t.Errorf("slower win: %+v; want no new best", st)
	}
	if st := mustRecord(t, alice, "minesweeper", game.ByTime, won(42*time.Second)); !st.NewBest {
		t.Errorf("faster win: %+v; want new best", st)
	}
	if got := alice.Top("minesweeper", game.ByTime, 1)[0].Value; got != "0:42" {
		t.Errorf("best time shown as %q; want 0:42", got)
	}
}

func TestTop_limitsToN(t *testing.T) {
	b, _ := testBoard(t)
	for i := range 5 {
		s := named(t, b, fmt.Sprintf("k%d", i), fmt.Sprintf("player%d", i))
		mustRecord(t, s, "tetris", game.ByScore, game.Result{Score: i * 100})
	}

	top := b.Session("").Top("tetris", game.ByScore, 3)

	if got := names(top); len(got) != 3 || got[0] != "player4" || got[2] != "player2" {
		t.Errorf("Top(3) = %v; want [player4 player3 player2]", got)
	}
}

func TestRank(t *testing.T) {
	b, _ := testBoard(t)
	alice := named(t, b, "a", "alice")
	bob := named(t, b, "b", "bob")
	mustRecord(t, alice, "2048", game.ByScore, game.Result{Score: 100})
	mustRecord(t, bob, "2048", game.ByScore, game.Result{Score: 200})

	if rank, of := alice.Rank("2048", game.ByScore); rank != 2 || of != 2 {
		t.Errorf("alice Rank = %d of %d; want 2 of 2", rank, of)
	}
	if rank, _ := b.Session("nobody").Rank("2048", game.ByScore); rank != 0 {
		t.Errorf("player with no result Rank = %d; want 0", rank)
	}
}

func TestBoard_survivesRestart(t *testing.T) {
	b, path := testBoard(t)
	alice := named(t, b, "a", "alice")
	mustRecord(t, alice, "snake", game.ByScore, game.Result{Score: 42})

	reopened, err := Open(path)
	if err != nil {
		t.Fatalf("reopening: %v", err)
	}
	s := reopened.Session("a")

	if s.Name() != "alice" {
		t.Errorf("name after restart = %q; want alice", s.Name())
	}
	if top := s.Top("snake", game.ByScore, 10); len(top) != 1 || top[0].Value != "42" {
		t.Errorf("Top after restart = %+v; want alice with 42", top)
	}
}

func TestBoard_saveIsPrivateAndLeavesNoTempFiles(t *testing.T) {
	b, path := testBoard(t)
	named(t, b, "a", "alice")

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("leaderboard file not written: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("file permissions = %o; want 600", perm)
	}
	entries, err := os.ReadDir(filepath.Dir(path))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Errorf("data directory holds %d files; want only the leaderboard", len(entries))
	}
}

func TestBoard_concurrentSessions(t *testing.T) {
	b, path := testBoard(t)
	const players = 20

	var wg sync.WaitGroup
	for i := range players {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s := b.Session(fmt.Sprintf("key%d", i))
			if err := s.SetName(fmt.Sprintf("player%d", i)); err != nil {
				t.Error(err)
				return
			}
			if _, err := s.Record("snake", game.ByScore, game.Result{Score: i}); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()

	reopened, err := Open(path)
	if err != nil {
		t.Fatalf("reopening after concurrent writes: %v", err)
	}
	if got := reopened.Session("").Top("snake", game.ByScore, players); len(got) != players {
		t.Errorf("recorded %d players; want %d", len(got), players)
	}
}

func TestLabel(t *testing.T) {
	for r, want := range map[game.Ranking]string{
		game.ByScore:  "best score",
		game.ByStreak: "best streak",
		game.ByTime:   "fastest win",
	} {
		if got := Label(r); got != want {
			t.Errorf("Label(%v) = %q; want %q", r, got, want)
		}
	}
}

func TestSession_Enabled(t *testing.T) {
	b, _ := testBoard(t)

	if (Session{}).Enabled() {
		t.Error("zero Session: Enabled = true; want false")
	}
	if !b.Session("").Enabled() || !b.Session("key").Enabled() {
		t.Error("sessions on a board: Enabled = false; want true, guests included")
	}
}
