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

// midGame returns a 14x12 board with a bent four-cell snake and food in view.
func midGame() *Model {
	m := newModel(30, 16, testRand())
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
