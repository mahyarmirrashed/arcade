package game_test

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/mahyarmirrashed/arcade/internal/game"
	"github.com/mahyarmirrashed/arcade/internal/ui"
)

// stubGame is a minimal Game implementation used only in tests.
type stubGame struct {
	name string
	desc string
}

func (s stubGame) Name() string                      { return s.name }
func (s stubGame) Description() string               { return s.desc }
func (s stubGame) New(_, _ int, _ ui.Styles) tea.Model            { return nil }

func TestRegistry_All_returnsInInsertionOrder(t *testing.T) {
	a := stubGame{name: "Alpha"}
	b := stubGame{name: "Beta"}
	c := stubGame{name: "Gamma"}

	reg := game.NewRegistry(a, b, c)
	got := reg.All()

	if len(got) != 3 {
		t.Fatalf("len(All()) = %d; want 3", len(got))
	}

	want := []string{"Alpha", "Beta", "Gamma"}
	for i, g := range got {
		if g.Name() != want[i] {
			t.Errorf("All()[%d].Name() = %q; want %q", i, g.Name(), want[i])
		}
	}
}

func TestRegistry_All_empty(t *testing.T) {
	reg := game.NewRegistry()
	if got := reg.All(); len(got) != 0 {
		t.Errorf("expected empty registry, got %d games", len(got))
	}
}
