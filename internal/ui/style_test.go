package ui_test

import (
	"io"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"

	"github.com/mahyarmirrashed/arcade/internal/ui"
)

// renderer returns a renderer that behaves like a player's terminal with the
// given colour support.
func renderer(p termenv.Profile) *lipgloss.Renderer {
	r := lipgloss.NewRenderer(io.Discard)
	r.SetColorProfile(p)
	return r
}

func TestNewStyles_followTheSessionRenderer(t *testing.T) {
	colour := ui.NewStyles(renderer(termenv.ANSI256))
	plain := ui.NewStyles(renderer(termenv.Ascii))

	if got := colour.Title.Render("x"); !strings.Contains(got, "38;5;255") {
		t.Errorf("256-colour Title.Render = %q; want colour code 38;5;255", got)
	}
	if got := plain.Title.Render("x"); got != "x" {
		t.Errorf("no-colour Title.Render = %q; want plain %q", got, "x")
	}
}

func TestStyles_NewStyle_usesTheSessionRenderer(t *testing.T) {
	s := ui.NewStyles(renderer(termenv.ANSI256))

	got := s.NewStyle().Foreground(ui.Spectrum[0]).Render("x")

	if !strings.Contains(got, "38;5;196") {
		t.Errorf("NewStyle().Render = %q; want colour code 38;5;196", got)
	}
}
