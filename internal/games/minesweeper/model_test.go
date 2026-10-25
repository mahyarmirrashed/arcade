package minesweeper

import (
	"math/rand"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/mahyarmirrashed/arcade/internal/ui"
)

// deterministicRng returns a seeded RNG for reproducible tests.
func deterministicRng() *rand.Rand {
	return rand.New(rand.NewSource(42))
}

// newTestBoard returns a small seeded board for testing.
// 5x5 with 3 mines, seeded around (2,2) so the centre is always safe.
func newTestBoard() board {
	b := newBoard(5, 5, 3)
	b.seed(2, 2, deterministicRng())
	return b
}

func newTestModel() *Model {
	return newModel(80, 24, ui.Styles{})
}

// ---- board.go tests ----

func TestNewBoard_correctDimensions(t *testing.T) {
	b := newBoard(10, 8, 5)
	if b.cols != 10 || b.rows != 8 {
		t.Errorf("dims = %dx%d; want 10x8", b.cols, b.rows)
	}
	if b.mines != 5 {
		t.Errorf("mines = %d; want 5", b.mines)
	}
	if b.seeded {
		t.Error("new board should not be seeded yet")
	}
}

func TestSeed_correctMineCount(t *testing.T) {
	b := newBoard(10, 10, 10)
	b.seed(5, 5, deterministicRng())

	count := 0
	for r := 0; r < b.rows; r++ {
		for c := 0; c < b.cols; c++ {
			if b.cells[r][c].mine {
				count++
			}
		}
	}
	if count != 10 {
		t.Errorf("mine count = %d; want 10", count)
	}
}

func TestSeed_safeZoneAroundFirstReveal(t *testing.T) {
	b := newBoard(10, 10, 20)
	b.seed(5, 5, deterministicRng())

	for dr := -1; dr <= 1; dr++ {
		for dc := -1; dc <= 1; dc++ {
			r, c := 5+dr, 5+dc
			if b.cells[r][c].mine {
				t.Errorf("mine at (%d,%d) which is in the safe zone around (5,5)", r, c)
			}
		}
	}
}

func TestSeed_neighbourCountsCorrect(t *testing.T) {
	b := newTestBoard()
	// For each non-mine cell, verify the stored neighbour count matches
	// the actual number of adjacent mine cells.
	for r := 0; r < b.rows; r++ {
		for c := 0; c < b.cols; c++ {
			if b.cells[r][c].mine {
				continue
			}
			actual := 0
			eachNeighbour(b.cols, b.rows, r, c, func(nr, nc int) {
				if b.cells[nr][nc].mine {
					actual++
				}
			})
			if b.cells[r][c].neighbours != actual {
				t.Errorf("cell (%d,%d): neighbours=%d; want %d",
					r, c, b.cells[r][c].neighbours, actual)
			}
		}
	}
}

func TestReveal_safeCellRevealed(t *testing.T) {
	b := newTestBoard()
	// Find a non-mine cell.
	for r := 0; r < b.rows; r++ {
		for c := 0; c < b.cols; c++ {
			if !b.cells[r][c].mine {
				hitMine := b.reveal(r, c, deterministicRng())
				if hitMine {
					t.Errorf("reveal of non-mine cell (%d,%d) returned hitMine=true", r, c)
				}
				if b.cells[r][c].state != cellRevealed {
					t.Errorf("cell (%d,%d) not revealed after reveal()", r, c)
				}
				return
			}
		}
	}
}

func TestReveal_mineCellReturnsTrue(t *testing.T) {
	b := newTestBoard()
	for r := 0; r < b.rows; r++ {
		for c := 0; c < b.cols; c++ {
			if b.cells[r][c].mine {
				hitMine := b.reveal(r, c, deterministicRng())
				if !hitMine {
					t.Errorf("reveal of mine cell (%d,%d) returned hitMine=false", r, c)
				}
				return
			}
		}
	}
}

func TestReveal_flaggedCellNotRevealed(t *testing.T) {
	b := newTestBoard()
	// Flag a safe cell then try to reveal it.
	for r := 0; r < b.rows; r++ {
		for c := 0; c < b.cols; c++ {
			if !b.cells[r][c].mine {
				b.cells[r][c].state = cellFlagged
				b.reveal(r, c, deterministicRng())
				if b.cells[r][c].state != cellFlagged {
					t.Error("flagged cell should not be revealed")
				}
				return
			}
		}
	}
}

func TestReveal_unseededBoardSeededOnFirstReveal(t *testing.T) {
	b := newBoard(5, 5, 3)
	if b.seeded {
		t.Fatal("board should not be seeded initially")
	}
	// Find a safe spot by trying centre.
	b.reveal(2, 2, deterministicRng())
	if !b.seeded {
		t.Error("board should be seeded after first reveal")
	}
}

func TestToggleFlag_hiddenToFlagged(t *testing.T) {
	b := newTestBoard()
	b.toggleFlag(0, 0)
	if b.cells[0][0].state != cellFlagged {
		t.Error("expected cellFlagged after toggleFlag on hidden cell")
	}
	if b.flagged != 1 {
		t.Errorf("flagged count = %d; want 1", b.flagged)
	}
}

func TestToggleFlag_flaggedToHidden(t *testing.T) {
	b := newTestBoard()
	b.toggleFlag(0, 0)
	b.toggleFlag(0, 0)
	if b.cells[0][0].state != cellHidden {
		t.Error("expected cellHidden after double toggleFlag")
	}
	if b.flagged != 0 {
		t.Errorf("flagged count = %d; want 0", b.flagged)
	}
}

func TestToggleFlag_revealedCellIgnored(t *testing.T) {
	b := newTestBoard()
	b.cells[2][2].state = cellRevealed
	b.toggleFlag(2, 2)
	if b.cells[2][2].state != cellRevealed {
		t.Error("toggleFlag on revealed cell should do nothing")
	}
}

func TestHasWon_notWonInitially(t *testing.T) {
	b := newTestBoard()
	if b.hasWon() {
		t.Error("fresh board should not report win")
	}
}

func TestHasWon_wonWhenAllSafeCellsRevealed(t *testing.T) {
	b := newTestBoard()
	// Manually reveal all non-mine cells.
	for r := 0; r < b.rows; r++ {
		for c := 0; c < b.cols; c++ {
			if !b.cells[r][c].mine {
				b.cells[r][c].state = cellRevealed
				b.revealed++
			}
		}
	}
	if !b.hasWon() {
		t.Error("expected win when all safe cells revealed")
	}
}

func TestRevealAll_allCellsRevealed(t *testing.T) {
	b := newTestBoard()
	b.revealAll()
	for r := 0; r < b.rows; r++ {
		for c := 0; c < b.cols; c++ {
			if b.cells[r][c].state != cellRevealed {
				t.Errorf("cell (%d,%d) not revealed after revealAll()", r, c)
			}
		}
	}
}

func TestFloodReveal_propagatesOnZero(t *testing.T) {
	// Build a board where (0,0) has 0 neighbours and its neighbour (0,1) also
	// has 0 neighbours - flood fill should cascade.
	b := newBoard(5, 5, 1)
	// Place a single mine far away.
	b.cells[4][4].mine = true
	// Recompute neighbours manually.
	for r := 0; r < b.rows; r++ {
		for c := 0; c < b.cols; c++ {
			if b.cells[r][c].mine {
				continue
			}
			count := 0
			eachNeighbour(b.cols, b.rows, r, c, func(nr, nc int) {
				if b.cells[nr][nc].mine {
					count++
				}
			})
			b.cells[r][c].neighbours = count
		}
	}
	b.seeded = true

	b.floodReveal(0, 0)

	// (0,0) has 0 neighbours, so all reachable 0-neighbour cells should reveal.
	// Only cells adjacent to (4,4) should remain hidden.
	if b.cells[0][0].state != cellRevealed {
		t.Error("(0,0) should be revealed")
	}
	if b.cells[0][4].state != cellRevealed {
		t.Error("(0,4) should be revealed via flood fill")
	}
}

// ---- model tests ----

func TestModel_initialState(t *testing.T) {
	m := newTestModel()
	if m.state != statePlaying {
		t.Error("new model should be in playing state")
	}
}

func TestModel_cursorMovement(t *testing.T) {
	m := newTestModel()
	startR, startC := m.curR, m.curC

	m.handleKey(tea.KeyMsg{Type: tea.KeyUp})
	if m.curR != startR-1 {
		t.Errorf("curR after up = %d; want %d", m.curR, startR-1)
	}

	m.handleKey(tea.KeyMsg{Type: tea.KeyDown})
	m.handleKey(tea.KeyMsg{Type: tea.KeyDown})
	if m.curR != startR+1 {
		t.Errorf("curR after down twice = %d; want %d", m.curR, startR+1)
	}

	m.curR = startR
	m.handleKey(tea.KeyMsg{Type: tea.KeyLeft})
	if m.curC != startC-1 {
		t.Errorf("curC after left = %d; want %d", m.curC, startC-1)
	}

	m.handleKey(tea.KeyMsg{Type: tea.KeyRight})
	m.handleKey(tea.KeyMsg{Type: tea.KeyRight})
	if m.curC != startC+1 {
		t.Errorf("curC after right twice = %d; want %d", m.curC, startC+1)
	}
}

func TestModel_cursorBoundedAtEdges(t *testing.T) {
	m := newTestModel()
	m.curR = 0
	m.curC = 0
	m.handleKey(tea.KeyMsg{Type: tea.KeyUp})
	m.handleKey(tea.KeyMsg{Type: tea.KeyLeft})
	if m.curR != 0 || m.curC != 0 {
		t.Errorf("cursor went below 0: (%d,%d)", m.curR, m.curC)
	}

	m.curR = m.board.rows - 1
	m.curC = m.board.cols - 1
	m.handleKey(tea.KeyMsg{Type: tea.KeyDown})
	m.handleKey(tea.KeyMsg{Type: tea.KeyRight})
	if m.curR != m.board.rows-1 || m.curC != m.board.cols-1 {
		t.Errorf("cursor exceeded board bounds: (%d,%d)", m.curR, m.curC)
	}
}

func TestModel_flagToggle(t *testing.T) {
	m := newTestModel()
	m.curR, m.curC = 0, 0
	m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("f")})
	if m.board.cells[0][0].state != cellFlagged {
		t.Error("expected cell to be flagged")
	}
}

func TestModel_revealLosesOnMine(t *testing.T) {
	m := newTestModel()
	// Seed the board first by revealing a safe centre cell.
	m.board.reveal(m.board.rows/2, m.board.cols/2, m.rng)
	// Find a mine and move cursor to it.
	for r := 0; r < m.board.rows; r++ {
		for c := 0; c < m.board.cols; c++ {
			if m.board.cells[r][c].mine {
				m.curR, m.curC = r, c
				m.board.cells[r][c].state = cellHidden // ensure it's hidden
				m.handleKey(tea.KeyMsg{Type: tea.KeyEnter})
				if m.state != stateLost {
					t.Error("expected stateLost after revealing mine")
				}
				return
			}
		}
	}
}

func TestModel_restartResets(t *testing.T) {
	m := newTestModel()
	m.state = stateLost
	cmd := m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	if cmd == nil {
		t.Fatal("expected restart command")
	}
	msg := cmd()
	rm, ok := msg.(restartMsg)
	if !ok {
		t.Fatalf("expected restartMsg, got %T", msg)
	}
	if rm.m.state != statePlaying {
		t.Error("restarted model should be in playing state")
	}
}

func TestModel_viewRendersWithoutPanic(t *testing.T) {
	m := newTestModel()
	_ = m.View()
}

func TestModel_viewRendersWon(t *testing.T) {
	m := newTestModel()
	m.state = stateWon
	_ = m.View()
}

func TestModel_viewRendersLost(t *testing.T) {
	m := newTestModel()
	m.state = stateLost
	_ = m.View()
}
