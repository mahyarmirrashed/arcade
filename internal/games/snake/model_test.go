package snake

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func runeKey(s string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

func TestGridSize(t *testing.T) {
	tests := []struct {
		name               string
		termW, termH       int
		wantCols, wantRows int
		wantOK             bool
	}{
		// The help panel, side panel, gaps and border take 24 columns; the
		// title, blank line and border take 4 rows.
		{"standard 80x24", 80, 24, 28, 20, true},
		{"capped when huge", 200, 60, maxCols, maxRows, true},
		{"smallest playable", 44, 14, 10, 10, true},
		{"one column short", 43, 14, 9, 10, false},
		{"one row short", 44, 13, 10, 9, false},
		{"zero size", 0, 0, -12, -4, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cols, rows, ok := gridSize(tt.termW, tt.termH)

			if cols != tt.wantCols || rows != tt.wantRows || ok != tt.wantOK {
				t.Errorf("gridSize(%d, %d) = %d, %d, %v; want %d, %d, %v",
					tt.termW, tt.termH, cols, rows, ok, tt.wantCols, tt.wantRows, tt.wantOK)
			}
		})
	}
}

func TestNew_sizesBoardToTerminal(t *testing.T) {
	var g Model
	m, ok := g.New(80, 24).(*Model)
	if !ok {
		t.Fatalf("New returned %T; want *Model", g.New(80, 24))
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
	m := newModel(20, 10, testRand())

	if !m.tooSmall {
		t.Error("tooSmall = false; want true")
	}
	if m.Init() != nil {
		t.Error("Init() returned a command; want nil")
	}
}

func TestUpdate_tickAdvancesSnake(t *testing.T) {
	m := newModel(80, 24, testRand())
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
	m := newModel(80, 24, testRand())
	head := m.board.body[0]

	_, cmd := m.Update(tickMsg{owner: m, gen: m.gen - 1})

	if got := m.board.body[0]; got != head {
		t.Errorf("head = %v; want unchanged %v", got, head)
	}
	if cmd != nil {
		t.Error("cmd != nil; a stale tick must not start another")
	}
}

func TestUpdate_deathStopsTicking(t *testing.T) {
	m := newModel(80, 24, testRand())
	m.board = heading(m.board.cols, m.board.rows, right, point{m.board.cols - 1, 0})

	_, cmd := m.Update(tickMsg{owner: m, gen: m.gen})

	if m.board.alive {
		t.Fatal("alive = true; want false after hitting the wall")
	}
	if cmd != nil {
		t.Error("cmd != nil; want ticking to stop on death")
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
			m := newModel(80, 24, testRand())
			m.board.dir, m.board.next = tt.start, tt.start

			m.Update(tt.key)

			if m.board.next != tt.want {
				t.Errorf("next = %v; want %v", m.board.next, tt.want)
			}
		})
	}
}

func TestUpdate_resizeResets(t *testing.T) {
	m := newModel(80, 24, testRand())
	m.board.score = 5
	gen := m.gen

	_, cmd := m.Update(tea.WindowSizeMsg{Width: 120, Height: 30})

	if m.board.score != 0 {
		t.Errorf("score = %d; want 0 after reset", m.board.score)
	}
	if m.board.cols != maxCols || m.board.rows != maxRows {
		t.Errorf("board = %dx%d; want %dx%d", m.board.cols, m.board.rows, maxCols, maxRows)
	}
	if m.gen == gen {
		t.Error("gen unchanged; old ticks would keep the previous game running")
	}
	if cmd == nil {
		t.Error("cmd = nil; want a tick for the new game")
	}
}

func TestUpdate_sameSizeKeepsGame(t *testing.T) {
	// Wish sends the starting size again once the session opens.
	m := newModel(80, 24, testRand())
	m.board.score = 5

	_, cmd := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	if m.board.score != 5 {
		t.Errorf("score = %d; want 5 (no reset)", m.board.score)
	}
	if cmd != nil {
		t.Error("cmd != nil; the existing tick chain is still running")
	}
}

func TestUpdate_resizeTooSmallStopsTicking(t *testing.T) {
	m := newModel(80, 24, testRand())

	_, cmd := m.Update(tea.WindowSizeMsg{Width: 20, Height: 10})

	if !m.tooSmall {
		t.Error("tooSmall = false; want true")
	}
	if cmd != nil {
		t.Error("cmd != nil; want no ticking while too small")
	}
}

func TestUpdate_quitKeysLeftToRoot(t *testing.T) {
	// The root model (or cmd/snake's wrapper) owns q and ctrl+c.
	for _, key := range []tea.KeyMsg{runeKey("q"), {Type: tea.KeyCtrlC}} {
		t.Run(key.String(), func(t *testing.T) {
			m := newModel(80, 24, testRand())

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
			m := newModel(80, 24, testRand())
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
	m := newModel(80, 24, testRand())
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
			m := newModel(80, 24, testRand())
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
	m := newModel(80, 24, testRand())
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
	m := newModel(80, 24, testRand())
	m.Update(runeKey("p"))

	m.Update(tea.KeyMsg{Type: tea.KeyUp})

	if m.board.next != right {
		t.Errorf("next = %v; want right (turns ignored while paused)", m.board.next)
	}
}

func TestUpdate_pauseIgnoredWhenGameOver(t *testing.T) {
	m := newModel(80, 24, testRand())
	m.board.alive = false

	m.Update(runeKey("p"))

	if m.paused {
		t.Error("paused = true; want false once the game is over")
	}
}

func TestUpdate_resizeUnpauses(t *testing.T) {
	m := newModel(80, 24, testRand())
	m.Update(runeKey("p"))

	_, cmd := m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})

	if m.paused {
		t.Error("paused = true; want a fresh, running game after resize")
	}
	if cmd == nil {
		t.Error("cmd = nil; want a tick for the new game")
	}
}

func TestUpdate_tickFromEarlierGameIgnored(t *testing.T) {
	// In the arcade, leaving Snake and starting it again creates a new Model
	// while the old one's tick may still be in flight. Both start at the same
	// generation, so only the owner tells them apart.
	old := newModel(80, 24, testRand())
	m := newModel(80, 24, testRand())
	head := m.board.body[0]

	_, cmd := m.Update(tickMsg{owner: old, gen: old.gen})

	if got := m.board.body[0]; got != head {
		t.Errorf("head = %v; want unchanged %v", got, head)
	}
	if cmd != nil {
		t.Error("cmd != nil; another game's tick must not start a second chain")
	}
}
