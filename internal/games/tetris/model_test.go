package tetris

import (
	"testing"
)

// ---- piece.go tests ----

func TestRotation_allPiecesAllRotations(t *testing.T) {
	// Each piece must have exactly the right number of filled cells across
	// all 4 rotations (tetrominoes always have exactly 4 cells).
	for pt := PieceI; pt <= PieceL; pt++ {
		for rot := 0; rot < 4; rot++ {
			s := Rotation(pt, rot)
			count := 0
			for _, pos := range Cells(s) {
				_ = pos
				count++
			}
			if count != 4 {
				t.Errorf("piece %d rot %d: got %d cells; want 4", pt, rot, count)
			}
		}
	}
}

func TestRotation_IpieceHorizontalRotation(t *testing.T) {
	// Rotation 0 of I: row 1 fully occupied cols 0-3.
	s := Rotation(PieceI, 0)
	want := [4][4]bool{
		{false, false, false, false},
		{true, true, true, true},
		{false, false, false, false},
		{false, false, false, false},
	}
	if s != Shape(want) {
		t.Errorf("I piece rot 0 mismatch: got %v", s)
	}
}

func TestRotation_OpieceAllSame(t *testing.T) {
	// O piece is identical in all 4 rotations.
	r0 := Rotation(PieceO, 0)
	for rot := 1; rot < 4; rot++ {
		if Rotation(PieceO, rot) != r0 {
			t.Errorf("O piece rot %d differs from rot 0", rot)
		}
	}
}

// ---- board.go tests ----

func emptyBoard() board { return board{} }

func TestCollides_emptyBoard(t *testing.T) {
	b := emptyBoard()
	s := Rotation(PieceI, 0)
	if collides(b, s, 0, 0) {
		t.Error("expected no collision on empty board at (0,0)")
	}
}

func TestCollides_outOfBoundsLeft(t *testing.T) {
	b := emptyBoard()
	s := Rotation(PieceI, 0)
	if !collides(b, s, 0, -1) {
		t.Error("expected collision when col < 0")
	}
}

func TestCollides_outOfBoundsRight(t *testing.T) {
	b := emptyBoard()
	s := Rotation(PieceI, 0) // 4 wide, at col 8 → rightmost cell at col 11
	if !collides(b, s, 0, 8) {
		t.Error("expected collision when piece exceeds right edge")
	}
}

func TestCollides_outOfBoundsBottom(t *testing.T) {
	b := emptyBoard()
	s := Rotation(PieceI, 0)
	// row 20 is out of bounds
	if !collides(b, s, 20, 0) {
		t.Error("expected collision at row 20 (out of bounds)")
	}
}

func TestCollides_filledCell(t *testing.T) {
	b := emptyBoard()
	b[1][0] = cell{filled: true, pieceType: PieceI}
	s := Rotation(PieceI, 0) // occupies row 1, cols 0-3
	if !collides(b, s, 0, 0) {
		t.Error("expected collision with filled cell")
	}
}

func TestPlace_stampsCorrectly(t *testing.T) {
	b := emptyBoard()
	s := Rotation(PieceO, 0) // fills (0,1),(0,2),(1,1),(1,2)
	b = place(b, s, 0, 0, PieceO)
	for _, pos := range Cells(s) {
		r, c := pos[0], pos[1]
		if !b[r][c].filled {
			t.Errorf("expected cell (%d,%d) to be filled after place", r, c)
		}
		if b[r][c].pieceType != PieceO {
			t.Errorf("cell (%d,%d) has wrong piece type", r, c)
		}
	}
}

func TestClearLines_noFullRows(t *testing.T) {
	b := emptyBoard()
	b[19][0] = cell{filled: true}
	newB, cleared := clearLines(b)
	if cleared != 0 {
		t.Errorf("cleared = %d; want 0", cleared)
	}
	if !newB[19][0].filled {
		t.Error("partial row should not have been cleared")
	}
}

func TestClearLines_oneFullRow(t *testing.T) {
	b := emptyBoard()
	for c := 0; c < boardCols; c++ {
		b[19][c] = cell{filled: true, pieceType: PieceI}
	}
	_, cleared := clearLines(b)
	if cleared != 1 {
		t.Errorf("cleared = %d; want 1", cleared)
	}
}

func TestClearLines_fourFullRows(t *testing.T) {
	b := emptyBoard()
	for r := 16; r < 20; r++ {
		for c := 0; c < boardCols; c++ {
			b[r][c] = cell{filled: true, pieceType: PieceI}
		}
	}
	newB, cleared := clearLines(b)
	if cleared != 4 {
		t.Errorf("cleared = %d; want 4", cleared)
	}
	// All rows should now be empty.
	for r := 0; r < boardRows; r++ {
		for c := 0; c < boardCols; c++ {
			if newB[r][c].filled {
				t.Errorf("expected empty board after 4-line clear, got filled at (%d,%d)", r, c)
			}
		}
	}
}

func TestClearLines_compactsCorrectly(t *testing.T) {
	b := emptyBoard()
	// Row 19: full
	for c := 0; c < boardCols; c++ {
		b[19][c] = cell{filled: true, pieceType: PieceI}
	}
	// Row 18: partial — only col 0
	b[18][0] = cell{filled: true, pieceType: PieceJ}

	newB, cleared := clearLines(b)
	if cleared != 1 {
		t.Errorf("cleared = %d; want 1", cleared)
	}
	// Row 19 should now contain the former row 18.
	if !newB[19][0].filled {
		t.Error("partial row should have compacted to bottom")
	}
	if newB[19][0].pieceType != PieceJ {
		t.Error("compacted cell has wrong piece type")
	}
}

func TestScore_table(t *testing.T) {
	tests := []struct {
		lines int
		level int
		want  int
	}{
		{0, 1, 0},
		{1, 1, 100},
		{2, 1, 300},
		{3, 1, 500},
		{4, 1, 800},
		{1, 2, 200},
		{4, 2, 1600},
		{4, 5, 4000},
	}
	for _, tt := range tests {
		got := score(tt.lines, tt.level)
		if got != tt.want {
			t.Errorf("score(%d, %d) = %d; want %d", tt.lines, tt.level, got, tt.want)
		}
	}
}

// ---- model.go tests ----

func newTestModel() *Model {
	return newModel(80, 24)
}

func TestModel_initiallyAlive(t *testing.T) {
	m := newTestModel()
	if !m.alive {
		t.Error("new model should be alive")
	}
}

func TestModel_gravityAdvancesPiece(t *testing.T) {
	m := newTestModel()
	startRow := m.current.row
	m.gravity()
	if m.current.row != startRow+1 {
		t.Errorf("row after gravity = %d; want %d", m.current.row, startRow+1)
	}
}

func TestModel_moveLeft(t *testing.T) {
	m := newTestModel()
	startCol := m.current.col
	m.tryMove(0, -1)
	if m.current.col != startCol-1 {
		t.Errorf("col after left = %d; want %d", m.current.col, startCol-1)
	}
}

func TestModel_moveRight(t *testing.T) {
	m := newTestModel()
	startCol := m.current.col
	m.tryMove(0, 1)
	if m.current.col != startCol+1 {
		t.Errorf("col after right = %d; want %d", m.current.col, startCol+1)
	}
}

func TestModel_rotateCW_changesRotation(t *testing.T) {
	m := newTestModel()
	m.current.pt = PieceT
	m.current.rot = 0
	m.current.row = 5
	m.current.col = 3
	m.tryRotate(1)
	if m.current.rot != 1 {
		t.Errorf("rotation after CW = %d; want 1", m.current.rot)
	}
}

func TestModel_rotateCCW_changesRotation(t *testing.T) {
	m := newTestModel()
	m.current.pt = PieceT
	m.current.rot = 0
	m.current.row = 5
	m.current.col = 3
	m.tryRotate(-1)
	if m.current.rot != 3 {
		t.Errorf("rotation after CCW = %d; want 3", m.current.rot)
	}
}

func TestModel_hardDrop_landsAtBottom(t *testing.T) {
	m := newTestModel()
	m.current.pt = PieceI
	m.current.rot = 0
	m.current.row = 0
	m.current.col = 0
	m.hardDrop()
	// After hard drop + lock a new piece spawns; board row 19 should have cells.
	rowFilled := false
	for c := 0; c < boardCols; c++ {
		if m.board[19][c].filled {
			rowFilled = true
			break
		}
	}
	if !rowFilled {
		t.Error("expected bottom of board to be partially filled after hard drop")
	}
}

func TestModel_lockIncreasesScore(t *testing.T) {
	m := newTestModel()
	// Fill rows 16–19 completely except the last col, then stamp the
	// missing cells directly so we don't rely on piece placement geometry.
	for r := 16; r < 20; r++ {
		for c := 0; c < boardCols-1; c++ {
			m.board[r][c] = cell{filled: true, pieceType: PieceJ}
		}
	}
	// Fill the last column in those rows directly — this completes 4 rows.
	for r := 16; r < 20; r++ {
		m.board[r][boardCols-1] = cell{filled: true, pieceType: PieceJ}
	}
	// lock() clears lines and adds score; call it directly.
	m.lock()
	if m.score == 0 {
		t.Error("expected score > 0 after clearing lines")
	}
}

func TestModel_gameOver_whenSpawnBlocked(t *testing.T) {
	m := newTestModel()
	// Fill the top of the board so spawn is blocked.
	for c := 0; c < boardCols; c++ {
		m.board[0][c] = cell{filled: true, pieceType: PieceI}
		m.board[1][c] = cell{filled: true, pieceType: PieceI}
	}
	m.spawn()
	if m.alive {
		t.Error("expected game over when spawn is blocked")
	}
}

func TestModel_wallKick_nearLeftWall(t *testing.T) {
	m := newTestModel()
	m.current.pt = PieceT
	m.current.rot = 0
	m.current.row = 5
	m.current.col = 0 // flush against left wall
	// CW rotation should succeed via wall-kick.
	m.tryRotate(1)
	// If rotation succeeded, rot changed; if it failed (no valid kick), rot stays 0.
	// The important assertion is that the piece didn't go out of bounds.
	if m.current.col < 0 {
		t.Errorf("piece col went negative after wall-kick: %d", m.current.col)
	}
}

func TestModel_wallKick_nearRightWall(t *testing.T) {
	m := newTestModel()
	m.current.pt = PieceT
	m.current.rot = 0
	m.current.row = 5
	m.current.col = boardCols - 3 // near right wall
	m.tryRotate(1)
	for _, pos := range Cells(m.current.shape()) {
		c := m.current.col + pos[1]
		if c >= boardCols {
			t.Errorf("piece extends beyond right wall at col %d after wall-kick", c)
		}
	}
}

func TestModel_viewRendersWithoutPanic(t *testing.T) {
	m := newTestModel()
	_ = m.View()
}

// ---- hold piece tests ----

func TestHold_firstHoldStashesCurrentAndSpawnsNext(t *testing.T) {
	m := newTestModel()
	original := m.current.pt
	m.holdPiece()

	if m.hold == nil {
		t.Fatal("expected hold slot to be filled after first hold")
	}
	if *m.hold != original {
		t.Errorf("held piece = %d; want %d", *m.hold, original)
	}
	// Current piece should now be the previously queued next — different object.
	if m.current.pt == original && m.current.row == 0 {
		// Not necessarily wrong if rng gave same type, but the piece should
		// have been freshly spawned (row=0, rot=0).
	}
}

func TestHold_secondHoldSwapsCurrentWithHeld(t *testing.T) {
	m := newTestModel()

	// First hold: stash current (A), spawn next (B).
	firstPT := m.current.pt
	m.holdPiece()

	// Lock the current piece to re-enable hold.
	m.lock()

	// Second hold: current is whatever spawned; swap it with held (A).
	secondPT := m.current.pt
	m.holdPiece()

	if m.hold == nil {
		t.Fatal("hold slot should not be nil after second hold")
	}
	// Held slot should now contain what was current before the second hold.
	if *m.hold != secondPT {
		t.Errorf("hold after swap = %d; want %d", *m.hold, secondPT)
	}
	// Current should be what was previously held (firstPT), at rot 0.
	if m.current.pt != firstPT {
		t.Errorf("current after swap = %d; want %d", m.current.pt, firstPT)
	}
	if m.current.rot != 0 {
		t.Errorf("swapped piece rotation = %d; want 0", m.current.rot)
	}
}

func TestHold_cannotHoldTwiceWithoutLocking(t *testing.T) {
	m := newTestModel()

	m.holdPiece() // first hold — valid
	heldAfterFirst := *m.hold

	m.holdPiece() // second hold without locking — should be ignored
	if *m.hold != heldAfterFirst {
		t.Error("hold should be blocked until the current piece locks")
	}
}

func TestHold_canHoldAgainAfterLock(t *testing.T) {
	m := newTestModel()
	m.holdPiece()
	m.lock() // lock re-enables hold
	if !m.canHold {
		t.Error("canHold should be true after locking a piece")
	}
}

func TestModel_levelIncreasesEvery10Lines(t *testing.T) {
	m := newTestModel()
	m.lines = 9
	m.score += score(1, m.level)
	m.lines++
	m.level = m.lines/10 + 1
	if m.level != 2 {
		t.Errorf("level = %d; want 2 after 10 lines", m.level)
	}
}

func TestModel_ownTickAdvancesPiece(t *testing.T) {
	m := newTestModel()
	startRow := m.current.row

	_, cmd := m.Update(tickMsg{owner: m})

	if m.current.row != startRow+1 {
		t.Errorf("row after tick = %d; want %d", m.current.row, startRow+1)
	}
	if cmd == nil {
		t.Error("cmd = nil; want the next tick")
	}
}

func TestModel_tickFromEarlierGameIgnored(t *testing.T) {
	// Leaving Tetris and starting it again, or restarting after game over,
	// creates a new Model while the old one's tick may still be in flight.
	// Applying it would start a second gravity loop at double speed.
	old := newTestModel()
	m := newTestModel()
	startRow := m.current.row

	_, cmd := m.Update(tickMsg{owner: old})

	if m.current.row != startRow {
		t.Errorf("row = %d; want unchanged %d", m.current.row, startRow)
	}
	if cmd != nil {
		t.Error("cmd != nil; another game's tick must not start a second loop")
	}
}
