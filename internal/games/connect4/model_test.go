package connect4

import (
	"math/rand"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/mahyarmirrashed/arcade/internal/ui"
)

// ---- board.go tests ----

func emptyBoard() board { return board{} }

func TestDrop_landsAtBottom(t *testing.T) {
	b := emptyBoard()
	b2, row := drop(b, 3, p0)
	if row != rows-1 {
		t.Errorf("first drop row = %d; want %d", row, rows-1)
	}
	if b2[rows-1][3] != p0 {
		t.Error("piece not placed at bottom")
	}
}

func TestDrop_stacksUp(t *testing.T) {
	b := emptyBoard()
	var row int
	for i := 0; i < rows; i++ {
		b, row = drop(b, 0, p0)
	}
	if row != 0 {
		t.Errorf("last drop row = %d; want 0", row)
	}
}

func TestDrop_fullColumnReturnsNegOne(t *testing.T) {
	b := emptyBoard()
	for i := 0; i < rows; i++ {
		b, _ = drop(b, 0, p0)
	}
	_, row := drop(b, 0, p1)
	if row != -1 {
		t.Errorf("drop into full col = %d; want -1", row)
	}
}

func TestColFull(t *testing.T) {
	b := emptyBoard()
	if colFull(b, 0) {
		t.Error("empty column should not be full")
	}
	for i := 0; i < rows; i++ {
		b, _ = drop(b, 0, p0)
	}
	if !colFull(b, 0) {
		t.Error("filled column should be full")
	}
}

func TestBoardFull(t *testing.T) {
	b := emptyBoard()
	if boardFull(b) {
		t.Error("empty board should not be full")
	}
	// Fill every cell.
	for c := 0; c < cols; c++ {
		for r := 0; r < rows; r++ {
			b[r][c] = p0
		}
	}
	if !boardFull(b) {
		t.Error("filled board should be full")
	}
}

func TestCheckWin_horizontal(t *testing.T) {
	b := emptyBoard()
	for c := 0; c < winRun; c++ {
		b, _ = drop(b, c, p0)
	}
	if !checkWin(b, rows-1, winRun-1) {
		t.Error("expected horizontal win")
	}
}

func TestCheckWin_vertical(t *testing.T) {
	b := emptyBoard()
	for i := 0; i < winRun; i++ {
		b, _ = drop(b, 0, p0)
	}
	if !checkWin(b, rows-winRun, 0) {
		t.Error("expected vertical win")
	}
}

func TestCheckWin_diagonalDown(t *testing.T) {
	// Build a \ diagonal win: p0 at (rows-4,0),(rows-3,1),(rows-2,2),(rows-1,3)
	b := emptyBoard()
	// Pad columns so pieces land at the right rows.
	// col 0: drop 1 p0 (lands at rows-1, but we need rows-4) - need rows-4 padding
	paddings := []int{rows - winRun, rows - winRun + 1, rows - winRun + 2, rows - winRun + 3}
	for c := 0; c < winRun; c++ {
		for i := 0; i < paddings[c]; i++ {
			b, _ = drop(b, c, p1)
		}
		b, _ = drop(b, c, p0)
	}
	if !checkWin(b, rows-1, winRun-1) {
		// Check the actual landing row.
		for c := 0; c < winRun; c++ {
			for r := rows - 1; r >= 0; r-- {
				if b[r][c] == p0 {
					if checkWin(b, r, c) {
						return // found a win
					}
					break
				}
			}
		}
		t.Error("expected diagonal win")
	}
}

func TestCheckWin_noWin(t *testing.T) {
	b := emptyBoard()
	b, _ = drop(b, 0, p0)
	b, _ = drop(b, 1, p1)
	b, _ = drop(b, 2, p0)
	if checkWin(b, rows-1, 2) {
		t.Error("no win should be detected for 3 non-connected pieces")
	}
}

func TestWinningCells_returnsAtLeast4(t *testing.T) {
	b := emptyBoard()
	for c := 0; c < winRun; c++ {
		b, _ = drop(b, c, p0)
	}
	cells := winningCells(b, rows-1, winRun-1)
	if len(cells) < winRun {
		t.Errorf("winningCells returned %d cells; want >= %d", len(cells), winRun)
	}
}

// ---- game.go tests ----

func newTestBroker() *broker {
	b := newBroker()
	b.rng = rand.New(rand.NewSource(42))
	return b
}

func TestBroker_createAndJoin(t *testing.T) {
	b := newTestBroker()
	r := b.Create()
	if r.Phase != phaseWaiting {
		t.Errorf("phase = %v; want phaseWaiting", r.Phase)
	}
	if len(r.Code) != 4 {
		t.Errorf("code len = %d; want 4", len(r.Code))
	}
	r2, ok := b.Join(r.Code)
	if !ok {
		t.Fatal("Join failed")
	}
	if r2.Phase != phasePlaying {
		t.Errorf("phase after join = %v; want phasePlaying", r2.Phase)
	}
}

func TestBroker_dropAndWin(t *testing.T) {
	b := newTestBroker()
	r := b.Create()
	b.Join(r.Code)

	// Drop 4 times in col 0, alternating turns: p0,p1,p0,p1... but we need
	// p0 to win horizontally across cols 0-3.
	// Turn alternates so we need to interleave.
	// p0 drops in cols 0,1,2,3; p1 drops in cols 4,4,4 (between).
	for i := 0; i < winRun; i++ {
		ok := b.Drop(r.Code, r.Game.Turn, i) // p0's turn
		if !ok {
			t.Fatalf("drop %d failed", i)
		}
		if r.Game.Won {
			if i < winRun-1 {
				t.Fatalf("won too early on drop %d", i)
			}
			return
		}
		if i < winRun-1 {
			// p1 drops somewhere else
			b.Drop(r.Code, r.Game.Turn, 4)
		}
	}
	if !r.Game.Won {
		t.Error("expected win after 4 in a row")
	}
}

func TestBroker_dropIntoFullCol(t *testing.T) {
	b := newTestBroker()
	r := b.Create()
	b.Join(r.Code)

	// Fill col 0 alternating.
	for i := 0; i < rows; i++ {
		turn := r.Game.Turn
		b.Drop(r.Code, turn, 0)
		if r.Game.Won {
			t.Skip("game won before column full")
		}
	}
	ok := b.Drop(r.Code, r.Game.Turn, 0)
	if ok {
		t.Error("dropping into full column should fail")
	}
}

func TestBroker_disconnectStartsGrace(t *testing.T) {
	b := newTestBroker()
	r := b.Create()
	b.Join(r.Code)
	b.Disconnect(r.Code, player1)
	b.mu.Lock()
	phase := b.rooms[r.Code].Phase
	b.mu.Unlock()
	if phase != phaseGrace {
		t.Errorf("phase after disconnect = %v; want phaseGrace", phase)
	}
}

func TestBroker_rejoinDuringGrace(t *testing.T) {
	b := newTestBroker()
	now := time.Now()
	r := b.Create()
	b.Join(r.Code)
	b.Disconnect(r.Code, player1)
	b.mu.Lock()
	b.rooms[r.Code].GraceEnds = now.Add(60 * time.Second)
	b.rooms[r.Code].now = func() time.Time { return now }
	b.mu.Unlock()
	r2, ok := b.Join(r.Code)
	if !ok || r2 == nil {
		t.Fatal("rejoin during grace should succeed")
	}
	if r2.Phase != phasePlaying {
		t.Errorf("phase after rejoin = %v; want phasePlaying", r2.Phase)
	}
}

func TestBroker_broadcast(t *testing.T) {
	b := newTestBroker()
	r := b.Create()
	b.Join(r.Code)
	b.Drop(r.Code, player0, 0)
	for seat := 0; seat < 2; seat++ {
		ch := r.channels[seat]
		select {
		case msg := <-ch:
			if msg.room == nil {
				t.Errorf("seat %d got nil room", seat)
			}
		default:
			t.Errorf("seat %d channel empty after drop", seat)
		}
	}
}

// ---- model tests ----

func newTestModel() *Model {
	room := Broker.Create()
	ji := textinput.New()
	ji.Placeholder = "XKCD"
	ji.CharLimit = 4
	ji.Width = 6
	return &Model{
		seat:      player0,
		room:      room,
		code:      room.Code,
		state:     stateWaiting,
		width:     120,
		height:    40,
		styles:    ui.Styles{},
		joinInput: ji,
		snap:      takeSnapshot(room, player0),
	}
}

func TestModel_initialState(t *testing.T) {
	m := newTestModel()
	if m.state != stateWaiting {
		t.Errorf("initial state = %v; want stateWaiting", m.state)
	}
}

func TestModel_cursorMovement(t *testing.T) {
	m := newTestModel()
	m.state = statePlaying
	m.snap.turn = player0
	m.snap.phase = phasePlaying
	m.snap.board = emptyBoard()

	m.handleKey(tea.KeyMsg{Type: tea.KeyRight})
	if m.cursor != 1 {
		t.Errorf("cursor after right = %d; want 1", m.cursor)
	}
	m.handleKey(tea.KeyMsg{Type: tea.KeyLeft})
	if m.cursor != 0 {
		t.Errorf("cursor after left = %d; want 0", m.cursor)
	}
}

func TestModel_cursorBounded(t *testing.T) {
	m := newTestModel()
	m.state = statePlaying
	m.snap.turn = player0
	m.snap.phase = phasePlaying
	m.snap.board = emptyBoard()

	m.cursor = 0
	m.handleKey(tea.KeyMsg{Type: tea.KeyLeft})
	if m.cursor != 0 {
		t.Error("cursor should not go below 0")
	}
	m.cursor = cols - 1
	m.handleKey(tea.KeyMsg{Type: tea.KeyRight})
	if m.cursor != cols-1 {
		t.Errorf("cursor should not exceed %d", cols-1)
	}
}

func TestModel_viewRendersWithoutPanic(t *testing.T) {
	m := newTestModel()
	_ = m.View()
}

func TestModel_restartEmitsReplaceMsg(t *testing.T) {
	m := newTestModel()
	m.state = stateOver
	cmd := m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	if cmd == nil {
		t.Fatal("expected restart command")
	}
	msg := cmd()
	if _, ok := msg.(replaceMsg); !ok {
		t.Fatalf("expected replaceMsg, got %T", msg)
	}
}
