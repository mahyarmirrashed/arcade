package snake

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"

	"github.com/mahyarmirrashed/arcade/internal/ui"
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
	m := newModel(52, 16, testRand(), testStyles())
	m.board = heading(14, 12, right, point{6, 5}, point{5, 5}, point{4, 5}, point{4, 6})
	m.board.food = point{10, 3}
	m.board.score = 3
	return m
}

func TestView_playing(t *testing.T) {
	checkGolden(t, "playing", midGame().View())
}

func TestView_tooSmall(t *testing.T) {
	checkGolden(t, "too_small", newModel(30, 10, testRand(), testStyles()).View())
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
	checkGolden(t, "wide", newModel(80, 24, testRand(), testStyles()).View())
}

func TestView_smallestBoard(t *testing.T) {
	checkGolden(t, "smallest", newModel(44, 14, testRand(), testStyles()).View())
}

func TestView_paused(t *testing.T) {
	m := midGame()
	m.paused = true
	checkGolden(t, "paused", m.View())
}

func TestRenderSide_fitsReservedWidth(t *testing.T) {
	// gridSize reserves sideW columns; the largest possible score and length
	// on a full 40x20 board must still fit.
	m := newModel(200, 60, testRand(), testStyles())
	m.board.score = maxCols*maxRows - 3
	m.board.body = make([]point, maxCols*maxRows)

	if got := lipgloss.Width(m.renderSide()); got > sideW {
		t.Errorf("side panel width = %d; want <= %d", got, sideW)
	}
}

func TestSegmentColor_cyclesThroughSpectrumFromNeck(t *testing.T) {
	n := len(ui.Spectrum)
	tests := []struct {
		segment int
		want    lipgloss.Color
	}{
		{1, ui.Spectrum[0]},
		{2, ui.Spectrum[1]},
		{n, ui.Spectrum[n-1]},
		{n + 1, ui.Spectrum[0]},
		{2*n + 3, ui.Spectrum[2]},
	}
	for _, tt := range tests {
		if got := segmentColor(tt.segment); got != tt.want {
			t.Errorf("segmentColor(%d) = %v; want %v", tt.segment, got, tt.want)
		}
	}
}

func TestFoodColor_isTheSegmentItAdds(t *testing.T) {
	b := heading(10, 10, right, point{5, 5}, point{4, 5}, point{3, 5})
	b.food = point{6, 5}
	want := foodColor(&b)

	b.step(testRand())

	if got := segmentColor(len(b.body) - 1); got != want {
		t.Errorf("new tail colour = %v; want the food's colour %v", got, want)
	}
}

func TestRenderGrid_coloursEachSegment(t *testing.T) {
	lipgloss.SetColorProfile(termenv.ANSI256)
	t.Cleanup(func() { lipgloss.SetColorProfile(termenv.Ascii) })

	m := midGame()
	grid := m.renderGrid()

	// midGame's snake has three body segments behind the head.
	for i := 1; i <= 3; i++ {
		code := "38;5;" + string(segmentColor(i))
		if !strings.Contains(grid, code) {
			t.Errorf("grid has no %q for segment %d", code, i)
		}
	}
	if code := "38;5;" + string(foodColor(&m.board)); !strings.Contains(grid, code) {
		t.Errorf("grid has no %q for the food", code)
	}
}
