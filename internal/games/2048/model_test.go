package _2048

import (
	"math/rand"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/mahyarmirrashed/arcade/internal/ui"
)

// ---- board.go tests ----

func TestCompact_removesZeros(t *testing.T) {
	row := [size]int{0, 2, 0, 4}
	got := compact(row)
	want := [size]int{2, 4, 0, 0}
	if got != want {
		t.Errorf("compact(%v) = %v; want %v", row, got, want)
	}
}

func TestMerge_table(t *testing.T) {
	tests := []struct {
		name       string
		in         [size]int
		wantRow    [size]int
		wantPoints int
	}{
		{
			name:       "no merge",
			in:         [size]int{2, 4, 8, 16},
			wantRow:    [size]int{2, 4, 8, 16},
			wantPoints: 0,
		},
		{
			name:       "single merge at start",
			in:         [size]int{2, 2, 4, 8},
			wantRow:    [size]int{4, 4, 8, 0},
			wantPoints: 4,
		},
		{
			name:       "two independent merges",
			in:         [size]int{2, 2, 4, 4},
			wantRow:    [size]int{4, 8, 0, 0},
			wantPoints: 12,
		},
		{
			name:       "no double-merge on triple",
			in:         [size]int{2, 2, 2, 0},
			wantRow:    [size]int{4, 2, 0, 0},
			wantPoints: 4,
		},
		{
			name:       "merge produces 2048",
			in:         [size]int{1024, 1024, 0, 0},
			wantRow:    [size]int{2048, 0, 0, 0},
			wantPoints: 2048,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotRow, gotPoints := merge(tt.in)
			if gotRow != tt.wantRow {
				t.Errorf("row: got %v; want %v", gotRow, tt.wantRow)
			}
			if gotPoints != tt.wantPoints {
				t.Errorf("points: got %d; want %d", gotPoints, tt.wantPoints)
			}
		})
	}
}

func TestSlide_left(t *testing.T) {
	g := grid{
		{2, 2, 0, 0},
		{0, 4, 4, 0},
		{2, 0, 2, 0},
		{0, 0, 0, 8},
	}
	got, points, moved := slide(g, dirLeft)
	if !moved {
		t.Error("expected move")
	}
	if points != 4+8+4 {
		t.Errorf("points = %d; want %d", points, 4+8+4)
	}
	wantRow0 := [size]int{4, 0, 0, 0}
	if got[0] != wantRow0 {
		t.Errorf("row 0 = %v; want %v", got[0], wantRow0)
	}
}

func TestSlide_right(t *testing.T) {
	g := grid{
		{2, 2, 0, 0},
	}
	got, _, moved := slide(g, dirRight)
	if !moved {
		t.Error("expected move")
	}
	wantRow0 := [size]int{0, 0, 0, 4}
	if got[0] != wantRow0 {
		t.Errorf("row 0 = %v; want %v", got[0], wantRow0)
	}
}

func TestSlide_up(t *testing.T) {
	g := grid{
		{2, 0, 0, 0},
		{2, 0, 0, 0},
		{0, 0, 0, 0},
		{0, 0, 0, 0},
	}
	got, points, moved := slide(g, dirUp)
	if !moved {
		t.Error("expected move")
	}
	if points != 4 {
		t.Errorf("points = %d; want 4", points)
	}
	if got[0][0] != 4 {
		t.Errorf("top-left = %d; want 4", got[0][0])
	}
	if got[1][0] != 0 {
		t.Errorf("[1][0] = %d; want 0", got[1][0])
	}
}

func TestSlide_down(t *testing.T) {
	g := grid{
		{2, 0, 0, 0},
		{2, 0, 0, 0},
		{0, 0, 0, 0},
		{0, 0, 0, 0},
	}
	got, _, moved := slide(g, dirDown)
	if !moved {
		t.Error("expected move")
	}
	if got[size-1][0] != 4 {
		t.Errorf("bottom-left = %d; want 4", got[size-1][0])
	}
}

func TestSlide_noMoveWhenNothing(t *testing.T) {
	g := grid{
		{2, 4, 8, 16},
		{32, 64, 128, 256},
		{512, 1024, 2, 4},
		{8, 16, 32, 64},
	}
	_, _, moved := slide(g, dirLeft)
	if moved {
		t.Error("expected no move on fully distinct left-aligned board")
	}
}

func TestRotate90_roundTrip(t *testing.T) {
	g := grid{
		{1, 2, 3, 4},
		{5, 6, 7, 8},
		{9, 10, 11, 12},
		{13, 14, 15, 16},
	}
	rotated := rotateN(g, 4)
	if rotated != g {
		t.Error("4 rotations should return original grid")
	}
}

func TestSpawn_placesOnEmptyCell(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	var g grid
	g, ok := spawn(g, rng)
	if !ok {
		t.Fatal("expected spawn to succeed on empty board")
	}
	count := 0
	for r := 0; r < size; r++ {
		for c := 0; c < size; c++ {
			if g[r][c] != 0 {
				count++
				if g[r][c] != 2 && g[r][c] != 4 {
					t.Errorf("spawned unexpected value %d", g[r][c])
				}
			}
		}
	}
	if count != 1 {
		t.Errorf("expected 1 spawned tile, got %d", count)
	}
}

func TestSpawn_failsOnFullBoard(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	var g grid
	for r := 0; r < size; r++ {
		for c := 0; c < size; c++ {
			g[r][c] = 2
		}
	}
	_, ok := spawn(g, rng)
	if ok {
		t.Error("expected spawn to fail on full board")
	}
}

func TestHasWon_detectsWin(t *testing.T) {
	var g grid
	g[0][0] = 2048
	if !hasWon(g) {
		t.Error("expected win with 2048 tile")
	}
}

func TestHasWon_noWin(t *testing.T) {
	var g grid
	g[0][0] = 1024
	if hasWon(g) {
		t.Error("expected no win with 1024 tile")
	}
}

func TestHasMovesLeft_fullBoardNoMerges(t *testing.T) {
	g := grid{
		{2, 4, 2, 4},
		{4, 2, 4, 2},
		{2, 4, 2, 4},
		{4, 2, 4, 2},
	}
	if hasMovesLeft(g) {
		t.Error("expected no moves left on checkerboard board")
	}
}

func TestHasMovesLeft_hasMerge(t *testing.T) {
	g := grid{
		{2, 2, 4, 8},
		{4, 8, 16, 32},
		{8, 16, 32, 64},
		{16, 32, 64, 128},
	}
	if !hasMovesLeft(g) {
		t.Error("expected moves left when adjacent tiles can merge")
	}
}

func TestHasMovesLeft_hasEmpty(t *testing.T) {
	var g grid
	g[0][0] = 2
	if !hasMovesLeft(g) {
		t.Error("expected moves left when empty cells exist")
	}
}

// ---- model.go tests ----

func newTestModel() *Model {
	return newModel(80, 24, ui.Styles{})
}

func TestModel_initialState(t *testing.T) {
	m := newTestModel()
	if m.state != statePlaying {
		t.Error("new model should be in playing state")
	}
	// Two tiles should be spawned.
	count := 0
	for r := 0; r < size; r++ {
		for c := 0; c < size; c++ {
			if m.g[r][c] != 0 {
				count++
			}
		}
	}
	if count != 2 {
		t.Errorf("expected 2 initial tiles, got %d", count)
	}
}

func TestModel_moveIncreasesScore(t *testing.T) {
	m := newTestModel()
	// Set up a board where a move definitely scores.
	m.g = grid{
		{2, 2, 0, 0},
		{0, 0, 0, 0},
		{0, 0, 0, 0},
		{0, 0, 0, 0},
	}
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyLeft})
	nm := next.(*Model)
	if nm.score == 0 {
		t.Error("expected score > 0 after merging tiles")
	}
}

func TestModel_bestScorePreservedOnRestart(t *testing.T) {
	m := newTestModel()
	m.score = 512
	m.best = 512

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	nm := next.(*Model)
	if nm.best != 512 {
		t.Errorf("best score after restart = %d; want 512", nm.best)
	}
	if nm.score != 0 {
		t.Errorf("score after restart = %d; want 0", nm.score)
	}
}

func TestModel_wonStateOnReaching2048(t *testing.T) {
	m := newTestModel()
	m.g = grid{
		{1024, 1024, 0, 0},
		{0, 0, 0, 0},
		{0, 0, 0, 0},
		{0, 0, 0, 0},
	}
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyLeft})
	nm := next.(*Model)
	if nm.state != stateWon {
		t.Errorf("state = %v; want stateWon", nm.state)
	}
}

func TestModel_noMoveOnLockedBoard(t *testing.T) {
	m := newTestModel()
	m.g = grid{
		{2, 4, 2, 4},
		{4, 2, 4, 2},
		{2, 4, 2, 4},
		{4, 2, 4, 2},
	}
	before := m.g
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyLeft})
	nm := next.(*Model)
	if nm.g != before {
		t.Error("board should not change when no move is possible")
	}
}

func TestModel_viewRendersWithoutPanic(t *testing.T) {
	m := newTestModel()
	_ = m.View()
}

func TestModel_viewRendersGameOver(t *testing.T) {
	m := newTestModel()
	m.state = stateLost
	_ = m.View()
}

func TestModel_viewRendersWon(t *testing.T) {
	m := newTestModel()
	m.state = stateWon
	_ = m.View()
}
