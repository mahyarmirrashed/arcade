package sudoku

import (
	"math/rand"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/mahyarmirrashed/arcade/internal/ui"
)

func deterministicRng() *rand.Rand { return rand.New(rand.NewSource(42)) }

func newTestModel() *Model { return newModel(120, 40, ui.Styles{}) }

// ---- board.go tests ----

func TestIsValidPlacement_row(t *testing.T) {
	var g grid
	g[0][0] = 5
	if isValidPlacement(g, 0, 3, 5) {
		t.Error("5 already in row 0, should be invalid")
	}
	if !isValidPlacement(g, 0, 3, 6) {
		t.Error("6 not in row 0, should be valid")
	}
}

func TestIsValidPlacement_col(t *testing.T) {
	var g grid
	g[2][4] = 7
	if isValidPlacement(g, 0, 4, 7) {
		t.Error("7 already in col 4, should be invalid")
	}
}

func TestIsValidPlacement_box(t *testing.T) {
	var g grid
	g[0][0] = 3
	if isValidPlacement(g, 1, 1, 3) {
		t.Error("3 already in top-left box, should be invalid")
	}
	if !isValidPlacement(g, 1, 1, 4) {
		t.Error("4 not in top-left box, should be valid")
	}
}

func TestSolve_solvesSolvable(t *testing.T) {
	// A partially filled grid that has a solution.
	var g grid
	g[0][0] = 5; g[0][1] = 3
	g[0][3] = 6
	g[0][6] = 9; g[0][7] = 8
	if !solve(&g) {
		t.Error("expected solve to succeed")
	}
	if !isSolved(g) {
		t.Error("solved grid should pass isSolved")
	}
}

func TestGenerate_produces52Cards(t *testing.T) {
	g := generate(deterministicRng())
	if !isSolved(g) {
		t.Error("generated grid should be a valid solved puzzle")
	}
}

func TestNewPuzzle_clueCount(t *testing.T) {
	puzzle, solution := newPuzzle(deterministicRng(), cluesTarget)
	// Count given clues.
	clues := 0
	for r := 0; r < size; r++ {
		for c := 0; c < size; c++ {
			if puzzle[r][c] != 0 {
				clues++
			}
		}
	}
	if clues < cluesTarget {
		t.Errorf("clues = %d; want >= %d", clues, cluesTarget)
	}
	// Solution should be solved.
	if !isSolved(solution) {
		t.Error("solution should be a valid solved grid")
	}
}

func TestNewPuzzle_cluesMatchSolution(t *testing.T) {
	puzzle, solution := newPuzzle(deterministicRng(), cluesTarget)
	for r := 0; r < size; r++ {
		for c := 0; c < size; c++ {
			if puzzle[r][c] != 0 && puzzle[r][c] != solution[r][c] {
				t.Errorf("clue (%d,%d)=%d does not match solution %d",
					r, c, puzzle[r][c], solution[r][c])
			}
		}
	}
}

func TestIsSolved_empty(t *testing.T) {
	var g grid
	if isSolved(g) {
		t.Error("empty grid should not be solved")
	}
}

func TestIsSolved_valid(t *testing.T) {
	g := generate(deterministicRng())
	if !isSolved(g) {
		t.Error("fully generated grid should be solved")
	}
}

func TestIsSolved_withDuplicate(t *testing.T) {
	g := generate(deterministicRng())
	// Corrupt: place same value twice in a row.
	g[0][1] = g[0][0]
	if isSolved(g) {
		t.Error("grid with duplicate in row should not be solved")
	}
}

func TestSameBox(t *testing.T) {
	if !sameBox(0, 0, 2, 2) {
		t.Error("(0,0) and (2,2) should be in the same box")
	}
	if sameBox(0, 0, 3, 3) {
		t.Error("(0,0) and (3,3) should be in different boxes")
	}
}

func TestCountSolutions_uniquePuzzle(t *testing.T) {
	puzzle, _ := newPuzzle(deterministicRng(), cluesTarget)
	n := countSolutions(puzzle, 2)
	if n != 1 {
		t.Errorf("puzzle should have exactly 1 solution, got %d", n)
	}
}

// ---- model tests ----

func TestModel_initialState(t *testing.T) {
	m := newTestModel()
	if m.state != statePlaying {
		t.Error("new model should be in playing state")
	}
	if m.curR != 0 || m.curC != 0 {
		t.Errorf("initial cursor = (%d,%d); want (0,0)", m.curR, m.curC)
	}
}

func TestModel_cursorMovement(t *testing.T) {
	m := newTestModel()
	m.handleKey(tea.KeyMsg{Type: tea.KeyDown})
	if m.curR != 1 {
		t.Errorf("curR after down = %d; want 1", m.curR)
	}
	m.handleKey(tea.KeyMsg{Type: tea.KeyRight})
	if m.curC != 1 {
		t.Errorf("curC after right = %d; want 1", m.curC)
	}
	m.handleKey(tea.KeyMsg{Type: tea.KeyUp})
	if m.curR != 0 {
		t.Errorf("curR after up = %d; want 0", m.curR)
	}
	m.handleKey(tea.KeyMsg{Type: tea.KeyLeft})
	if m.curC != 0 {
		t.Errorf("curC after left = %d; want 0", m.curC)
	}
}

func TestModel_cursorBounded(t *testing.T) {
	m := newTestModel()
	m.curR, m.curC = 0, 0
	m.handleKey(tea.KeyMsg{Type: tea.KeyUp})
	m.handleKey(tea.KeyMsg{Type: tea.KeyLeft})
	if m.curR != 0 || m.curC != 0 {
		t.Error("cursor should not go below 0")
	}
	m.curR, m.curC = size-1, size-1
	m.handleKey(tea.KeyMsg{Type: tea.KeyDown})
	m.handleKey(tea.KeyMsg{Type: tea.KeyRight})
	if m.curR != size-1 || m.curC != size-1 {
		t.Error("cursor should not exceed size-1")
	}
}

func TestModel_placeDigit(t *testing.T) {
	m := newTestModel()
	// Move to a non-locked cell.
	for r := 0; r < size; r++ {
		for c := 0; c < size; c++ {
			if !m.locked(r, c) {
				m.curR, m.curC = r, c
				m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("5")})
				if m.current[r][c] != 5 {
					t.Errorf("expected 5 at (%d,%d), got %d", r, c, m.current[r][c])
				}
				return
			}
		}
	}
}

func TestModel_clearCell(t *testing.T) {
	m := newTestModel()
	for r := 0; r < size; r++ {
		for c := 0; c < size; c++ {
			if !m.locked(r, c) {
				m.curR, m.curC = r, c
				m.current[r][c] = 7
				m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("0")})
				if m.current[r][c] != 0 {
					t.Errorf("expected 0 after clear, got %d", m.current[r][c])
				}
				return
			}
		}
	}
}

func TestModel_lockedCellCannotBeChanged(t *testing.T) {
	m := newTestModel()
	for r := 0; r < size; r++ {
		for c := 0; c < size; c++ {
			if m.locked(r, c) {
				original := m.current[r][c]
				m.curR, m.curC = r, c
				// Try to overwrite with a different digit.
				digit := (original % 9) + 1
				m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(string(rune('0' + digit)))})
				if m.current[r][c] != original {
					t.Errorf("locked cell (%d,%d) was modified", r, c)
				}
				return
			}
		}
	}
}

func TestModel_winOnCorrectSolution(t *testing.T) {
	m := newTestModel()
	// Fill all empty cells with the solution.
	for r := 0; r < size; r++ {
		for c := 0; c < size; c++ {
			if !m.locked(r, c) {
				m.current[r][c] = m.solution[r][c]
			}
		}
	}
	// Place the last cell via handleKey to trigger win check.
	// Find the last empty cell and place it.
	for r := 0; r < size; r++ {
		for c := 0; c < size; c++ {
			if !m.locked(r, c) {
				m.current[r][c] = 0 // clear it so place() triggers the check
				m.curR, m.curC = r, c
				m.handleKey(tea.KeyMsg{Type: tea.KeyRunes,
					Runes: []rune(string(rune('0' + m.solution[r][c])))})
				if m.state != stateWon {
					t.Error("expected stateWon after placing correct solution")
				}
				return
			}
		}
	}
}

func TestModel_restartProducesNewGame(t *testing.T) {
	m := newTestModel()
	m.state = stateWon
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
		t.Error("restarted model should be playing")
	}
}

func TestModel_viewRendersWithoutPanic(t *testing.T) {
	m := newTestModel()
	_ = m.View()
}

func TestModel_viewWonRendersWithoutPanic(t *testing.T) {
	m := newTestModel()
	m.state = stateWon
	_ = m.View()
}
