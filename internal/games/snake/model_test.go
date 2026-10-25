package snake

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/mahyarmirrashed/arcade/internal/game"
)

func runeKey(s string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

func TestNew_sizesBoardToTerminal(t *testing.T) {
	var g Model
	m, ok := g.New(80, 24, testStyles()).(*Model)
	if !ok {
		t.Fatalf("New returned %T; want *Model", g.New(80, 24, testStyles()))
	}

	if m.tooSmall {
		t.Fatal("tooSmall = true; want false")
	}
	if m.board.cols != 28 || m.board.rows != 20 {
		t.Errorf("board = %dx%d; want 28x20", m.board.cols, m.board.rows)
	}
	if m.Init() == nil {
		t.Error("Init() = nil; want a tick command")
	}
}

func TestNew_tooSmallTerminalDoesNotTick(t *testing.T) {
	m := newModel(20, 10, testRand(), testStyles())

	if !m.tooSmall {
		t.Error("tooSmall = false; want true")
	}
	if m.Init() != nil {
		t.Error("Init() returned a command; want nil")
	}
}

func TestUpdate_tickAdvancesSnake(t *testing.T) {
	m := newModel(80, 24, testRand(), testStyles())
	head := m.board.body[0]

	_, cmd := m.Update(tickMsg{owner: m, gen: m.gen})

	if got, want := m.board.body[0], head.add(right.delta()); got != want {
		t.Errorf("head = %v; want %v", got, want)
	}
	if cmd == nil {
		t.Error("cmd = nil; want the next tick")
	}
}

func TestUpdate_staleTickIgnored(t *testing.T) {
	m := newModel(80, 24, testRand(), testStyles())
	head := m.board.body[0]

	_, cmd := m.Update(tickMsg{owner: m, gen: m.gen - 1})

	if got := m.board.body[0]; got != head {
		t.Errorf("head = %v; want unchanged %v", got, head)
	}
	if cmd != nil {
		t.Error("cmd != nil; a stale tick must not start another")
	}
}

func TestUpdate_deathStopsTickingAndReportsScore(t *testing.T) {
	m := newModel(80, 24, testRand(), testStyles())
	m.board = heading(m.board.cols, m.board.rows, right, point{m.board.cols - 1, 0})
	m.board.score = 7

	_, cmd := m.Update(tickMsg{owner: m, gen: m.gen})

	if m.board.alive {
		t.Fatal("alive = true; want false after hitting the wall")
	}
	if cmd == nil {
		t.Fatal("cmd = nil; want the result reported")
	}
	want := game.ResultMsg{Game: "snake", Ranking: game.ByScore, Result: game.Result{Score: 7}}
	if msg := cmd(); msg != want {
		t.Errorf("cmd() = %#v; want %#v and no further tick", msg, want)
	}
}

func TestUpdate_fillingTheBoardReportsAWin(t *testing.T) {
	m := newModel(80, 24, testRand(), testStyles())
	m.board = heading(2, 2, left, point{1, 1}, point{1, 0}, point{0, 0})
	m.board.food = point{0, 1}

	_, cmd := m.Update(tickMsg{owner: m, gen: m.gen})

	if msg, ok := cmd().(game.ResultMsg); !ok || !msg.Result.Won || msg.Result.Score != 1 {
		t.Errorf("cmd() = %#v; want a won result with score 1", cmd())
	}
}

func TestUpdate_standingAppearsAtGameOver(t *testing.T) {
	m := newModel(80, 24, testRand(), testStyles())
	m.board.alive = false

	m.Update(game.StandingMsg{Standing: game.Standing{Ranked: true, NewBest: true, Rank: 1, Of: 3}})

	if v := m.View(); !strings.Contains(v, "new best! #1/3") {
		t.Errorf("game-over view doesn't show the standing:\n%s", v)
	}
	m.Update(runeKey("r"))
	if m.standing != nil {
		t.Error("standing kept after restart; want it cleared for the new game")
	}
}

func TestUpdate_keysSteer(t *testing.T) {
	tests := []struct {
		name  string
		key   tea.KeyMsg
		start direction // perpendicular to the key, so the turn is allowed
		want  direction
	}{
		{"up arrow", tea.KeyMsg{Type: tea.KeyUp}, right, up},
		{"k", runeKey("k"), right, up},
		{"down arrow", tea.KeyMsg{Type: tea.KeyDown}, right, down},
		{"j", runeKey("j"), right, down},
		{"left arrow", tea.KeyMsg{Type: tea.KeyLeft}, up, left},
		{"h", runeKey("h"), up, left},
		{"right arrow", tea.KeyMsg{Type: tea.KeyRight}, up, right},
		{"l", runeKey("l"), up, right},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := newModel(80, 24, testRand(), testStyles())
			m.board.dir, m.board.next = tt.start, tt.start

			m.Update(tt.key)

			if m.board.next != tt.want {
				t.Errorf("next = %v; want %v", m.board.next, tt.want)
			}
		})
	}
}

func TestUpdate_sameSizeKeepsGame(t *testing.T) {
	// Wish sends the starting size again once the session opens.
	m := newModel(80, 24, testRand(), testStyles())
	m.board.score = 5

	_, cmd := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	if m.board.score != 5 {
		t.Errorf("score = %d; want 5 (no reset)", m.board.score)
	}
	if cmd != nil {
		t.Error("cmd != nil; the existing tick chain is still running")
	}
}

func TestUpdate_quitKeysLeftToRoot(t *testing.T) {
	// The root model owns q and ctrl+c.
	for _, key := range []tea.KeyMsg{runeKey("q"), {Type: tea.KeyCtrlC}} {
		t.Run(key.String(), func(t *testing.T) {
			m := newModel(80, 24, testRand(), testStyles())

			_, cmd := m.Update(key)

			if cmd != nil {
				t.Errorf("cmd() = %T; want nil", cmd())
			}
		})
	}
}

func TestUpdate_restart(t *testing.T) {
	tests := []struct {
		name  string
		setup func(*board)
	}{
		{"after game over", func(b *board) { b.alive = false }},
		{"after winning", func(b *board) { b.won = true }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := newModel(80, 24, testRand(), testStyles())
			m.board.score = 7
			tt.setup(&m.board)
			gen := m.gen

			_, cmd := m.Update(runeKey("r"))

			if !m.board.alive || m.board.won || m.board.score != 0 {
				t.Errorf("alive, won, score = %v, %v, %d; want a fresh game",
					m.board.alive, m.board.won, m.board.score)
			}
			if m.gen == gen {
				t.Error("gen unchanged; want old ticks invalidated")
			}
			if cmd == nil {
				t.Error("cmd = nil; want ticking to restart")
			}
		})
	}
}

func TestUpdate_restartIgnoredMidGame(t *testing.T) {
	m := newModel(80, 24, testRand(), testStyles())
	m.board.score = 7

	_, cmd := m.Update(runeKey("r"))

	if m.board.score != 7 {
		t.Errorf("score = %d; want 7 (r only restarts a finished game)", m.board.score)
	}
	if cmd != nil {
		t.Error("cmd != nil; want nil")
	}
}

func TestUpdate_pause(t *testing.T) {
	for _, key := range []tea.KeyMsg{runeKey("p"), {Type: tea.KeySpace}} {
		t.Run(key.String(), func(t *testing.T) {
			m := newModel(80, 24, testRand(), testStyles())
			head := m.board.body[0]
			gen := m.gen

			_, cmd := m.Update(key)

			if !m.paused {
				t.Fatal("paused = false; want true")
			}
			if cmd != nil {
				t.Error("cmd != nil; want no tick while paused")
			}
			m.Update(tickMsg{owner: m, gen: gen})
			m.Update(tickMsg{owner: m, gen: m.gen})
			if got := m.board.body[0]; got != head {
				t.Errorf("head = %v; want unchanged %v while paused", got, head)
			}
		})
	}
}

func TestUpdate_resume(t *testing.T) {
	m := newModel(80, 24, testRand(), testStyles())
	m.Update(runeKey("p"))
	gen := m.gen

	_, cmd := m.Update(runeKey("p"))

	if m.paused {
		t.Fatal("paused = true; want false")
	}
	if cmd == nil {
		t.Error("cmd = nil; want ticking to resume")
	}
	if m.gen == gen {
		t.Error("gen unchanged; a tick from before the pause would double the speed")
	}
}

func TestUpdate_pausedIgnoresTurns(t *testing.T) {
	m := newModel(80, 24, testRand(), testStyles())
	m.Update(runeKey("p"))

	m.Update(tea.KeyMsg{Type: tea.KeyUp})

	if m.board.next != right {
		t.Errorf("next = %v; want right (turns ignored while paused)", m.board.next)
	}
}

func TestUpdate_pauseIgnoredWhenGameOver(t *testing.T) {
	m := newModel(80, 24, testRand(), testStyles())
	m.board.alive = false

	m.Update(runeKey("p"))

	if m.paused {
		t.Error("paused = true; want false once the game is over")
	}
}

func TestUpdate_tickFromEarlierGameIgnored(t *testing.T) {
	// In the arcade, leaving Snake and starting it again creates a new Model
	// while the old one's tick may still be in flight. Both start at the same
	// generation, so only the owner tells them apart.
	old := newModel(80, 24, testRand(), testStyles())
	m := newModel(80, 24, testRand(), testStyles())
	head := m.board.body[0]

	_, cmd := m.Update(tickMsg{owner: old, gen: old.gen})

	if got := m.board.body[0]; got != head {
		t.Errorf("head = %v; want unchanged %v", got, head)
	}
	if cmd != nil {
		t.Error("cmd != nil; another game's tick must not start a second chain")
	}
}

func TestUpdate_resizeKeepsGame(t *testing.T) {
	m := newModel(80, 24, testRand(), testStyles())
	m.board.score = 5
	head := m.board.body[0]
	gen := m.gen

	_, cmd := m.Update(tea.WindowSizeMsg{Width: 120, Height: 30})

	if m.board.score != 5 || m.board.body[0] != head {
		t.Errorf("score, head = %d, %v; want the game kept (5, %v)", m.board.score, m.board.body[0], head)
	}
	if m.board.cols != 28 || m.board.rows != 20 {
		t.Errorf("board = %dx%d; want it kept at 28x20", m.board.cols, m.board.rows)
	}
	if m.tooSmall || m.paused || m.gen != gen {
		t.Errorf("tooSmall, paused, gen changed = %v, %v, %v; want the game running on", m.tooSmall, m.paused, m.gen != gen)
	}
	if cmd != nil {
		t.Error("cmd != nil; the existing tick chain is still running")
	}
}

func TestUpdate_shrinkingBelowBoardPauses(t *testing.T) {
	// A 28x20 board needs 80 columns; 70 can't show it.
	m := newModel(80, 24, testRand(), testStyles())
	m.board.score = 5
	gen := m.gen

	_, cmd := m.Update(tea.WindowSizeMsg{Width: 70, Height: 24})

	if !m.tooSmall || !m.paused {
		t.Errorf("tooSmall, paused = %v, %v; want true, true", m.tooSmall, m.paused)
	}
	if m.gen == gen {
		t.Error("gen unchanged; the tick in flight would move the snake while it can't be seen")
	}
	if cmd != nil {
		t.Error("cmd != nil; want no ticking while too small")
	}
	if m.board.score != 5 {
		t.Errorf("score = %d; want 5 (game kept)", m.board.score)
	}
}

func TestUpdate_growingBackLeavesGamePaused(t *testing.T) {
	m := newModel(80, 24, testRand(), testStyles())
	m.board.score = 5
	m.Update(tea.WindowSizeMsg{Width: 70, Height: 24})

	_, cmd := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	if m.tooSmall {
		t.Error("tooSmall = true; want false once the board fits again")
	}
	if !m.paused || cmd != nil {
		t.Error("game resumed by itself; want it to wait for p")
	}
	if _, cmd := m.Update(runeKey("p")); cmd == nil {
		t.Error("p did not resume the game")
	}
	if m.board.score != 5 {
		t.Errorf("score = %d; want 5 (game kept)", m.board.score)
	}
}

func TestUpdate_resizeWhilePausedStaysPaused(t *testing.T) {
	m := newModel(80, 24, testRand(), testStyles())
	m.Update(runeKey("p"))

	_, cmd := m.Update(tea.WindowSizeMsg{Width: 120, Height: 30})

	if !m.paused || cmd != nil {
		t.Error("resize unpaused the game; want it to stay paused")
	}
}

func TestUpdate_startsGameOnceTerminalFits(t *testing.T) {
	m := newModel(20, 10, testRand(), testStyles())

	_, cmd := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	if m.tooSmall || !m.board.alive {
		t.Fatalf("tooSmall, alive = %v, %v; want a new game", m.tooSmall, m.board.alive)
	}
	if m.board.cols != 28 || m.board.rows != 20 {
		t.Errorf("board = %dx%d; want 28x20", m.board.cols, m.board.rows)
	}
	if cmd == nil {
		t.Error("cmd = nil; want the new game to start ticking")
	}
}

func TestMinSize_fitsTheStandardBoard(t *testing.T) {
	// The 28x20 board plus the help and side panels fills an 80x24 terminal.
	var g Model
	if w, h := g.MinSize(); w != 80 || h != 24 {
		t.Errorf("MinSize = %dx%d; want 80x24", w, h)
	}
}

func TestNew_boardIsTheSameSizeInAnyTerminal(t *testing.T) {
	// Every ranked game is played on the same board, so scores compare fairly.
	for _, size := range [][2]int{{80, 24}, {120, 40}, {300, 100}} {
		m := newModel(size[0], size[1], testRand(), testStyles())
		if m.board.cols != boardCols || m.board.rows != boardRows {
			t.Errorf("%dx%d terminal: board = %dx%d; want %dx%d",
				size[0], size[1], m.board.cols, m.board.rows, boardCols, boardRows)
		}
	}
}

func TestUpdate_restartKeepsTheStandardBoard(t *testing.T) {
	m := newModel(80, 24, testRand(), testStyles())
	m.Update(tea.WindowSizeMsg{Width: 120, Height: 30})
	m.board.alive = false

	m.Update(runeKey("r"))

	if m.board.cols != boardCols || m.board.rows != boardRows {
		t.Errorf("board = %dx%d; want %dx%d", m.board.cols, m.board.rows, boardCols, boardRows)
	}
}
