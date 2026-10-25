package snake

import (
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

var update = flag.Bool("update", false, "rewrite the golden files in testdata/")

func TestMain(m *testing.M) {
	// Render without colour so the golden files are plain text.
	lipgloss.SetColorProfile(termenv.Ascii)
	os.Exit(m.Run())
}

// checkGolden compares got with testdata/<name>.golden.
// Run `go test ./internal/games/snake -update` to rewrite it.
func checkGolden(t *testing.T, name, got string) {
	t.Helper()
	path := filepath.Join("testdata", name+".golden")
	if *update {
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading golden file (run with -update to create it): %v", err)
	}
	if got != string(want) {
		t.Errorf("View() does not match %s\ngot:\n%s\nwant:\n%s", path, got, want)
	}
}

// midGame returns a 14x12 board with a bent four-cell snake and food in view,
// in the smallest terminal that fits it.
func midGame() *Model {
	m := newModel(52, 16, testRand())
	m.board = heading(14, 12, right, point{6, 5}, point{5, 5}, point{4, 5}, point{4, 6})
	m.board.food = point{10, 3}
	m.board.score = 3
	return m
}

func TestView_playing(t *testing.T) {
	checkGolden(t, "playing", midGame().View())
}

func TestView_tooSmall(t *testing.T) {
	checkGolden(t, "too_small", newModel(30, 10, testRand()).View())
}

func TestView_gameOver(t *testing.T) {
	m := midGame()
	m.board.alive = false
	checkGolden(t, "game_over", m.View())
}

func TestView_won(t *testing.T) {
	m := midGame()
	m.board.won = true
	checkGolden(t, "won", m.View())
}

func TestView_standardTerminal(t *testing.T) {
	checkGolden(t, "wide", newModel(80, 24, testRand()).View())
}

func TestView_smallestBoard(t *testing.T) {
	checkGolden(t, "smallest", newModel(44, 14, testRand()).View())
}

func TestView_paused(t *testing.T) {
	m := midGame()
	m.paused = true
	checkGolden(t, "paused", m.View())
}

func TestRenderSide_fitsReservedWidth(t *testing.T) {
	// gridSize reserves sideW columns; the largest possible score and length
	// on a full 40x20 board must still fit.
	m := newModel(200, 60, testRand())
	m.board.score = maxCols*maxRows - 3
	m.board.body = make([]point, maxCols*maxRows)

	if got := lipgloss.Width(m.renderSide()); got > sideW {
		t.Errorf("side panel width = %d; want <= %d", got, sideW)
	}
}
