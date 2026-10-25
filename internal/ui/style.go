// Package ui provides shared Lip Gloss styles and the arcade colour palette.
// All game packages and the lobby import from here to keep a consistent look.
// Styles are built per SSH session with NewStyles; there are deliberately no
// global styles, because those would take colour support from the server's
// terminal instead of the player's.
//
// Palette: monochrome - white text, dark grey accents, minimal borders.
// Inspired by terminal.shop's minimal aesthetic.
package ui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

// Palette.
var (
	ColorText    = lipgloss.Color("255") // bright white
	ColorSubtle  = lipgloss.Color("245") // light grey - secondary text
	ColorMuted   = lipgloss.Color("238") // dark grey - de-emphasised / borders
	ColorInverse = lipgloss.Color("0")   // black - used for selected item bg
)


// Spectrum is a rainbow of bright colours for game pieces, matching the Tetris
// piece colours: red, orange, yellow, green, cyan, blue, purple.
var Spectrum = []lipgloss.Color{
	lipgloss.Color("196"), // red
	lipgloss.Color("208"), // orange
	lipgloss.Color("226"), // yellow
	lipgloss.Color("46"),  // green
	lipgloss.Color("51"),  // cyan
	lipgloss.Color("21"),  // blue
	lipgloss.Color("129"), // purple
}

// Styles are the shared styles for one SSH session. They are built from that
// session's renderer, so colours follow the player's terminal rather than the
// server's. Over SSH, get the renderer from server.Renderer.
type Styles struct {
	r *lipgloss.Renderer

	Title    lipgloss.Style // screen/section headings
	Subtitle lipgloss.Style // secondary labels
	Muted    lipgloss.Style // de-emphasised text (help hints, empty state)
	Accent   lipgloss.Style // highlighted values (scores, selected state)
	Error    lipgloss.Style // game-over or error messages
	Border   lipgloss.Style // a normal border in the muted colour
	Selected lipgloss.Style // a selected list item, inverted
}

// NewStyles returns the shared styles for the session rendered by r.
func NewStyles(r *lipgloss.Renderer) Styles {
	return Styles{
		r:        r,
		Title:    r.NewStyle().Bold(true).Foreground(ColorText),
		Subtitle: r.NewStyle().Foreground(ColorSubtle),
		Muted:    r.NewStyle().Foreground(ColorMuted),
		Accent:   r.NewStyle().Bold(true).Foreground(ColorText),
		Error:    r.NewStyle().Bold(true).Foreground(ColorText).Reverse(true),
		Border:   r.NewStyle().BorderStyle(lipgloss.NormalBorder()).BorderForeground(ColorMuted),
		Selected: r.NewStyle().Bold(true).Foreground(ColorInverse).Background(ColorText),
	}
}

// NewStyle returns a blank style for the same session, for game-specific
// styling such as piece colours.
func (s Styles) NewStyle() lipgloss.Style { return s.r.NewStyle() }

// TooSmall is the notice shown in place of a screen that doesn't fit a
// width x height terminal. minW and minH are the size the screen needs.
func (s Styles) TooSmall(width, height, minW, minH int) string {
	notice := lipgloss.JoinVertical(lipgloss.Center,
		s.Error.Render(" terminal too small "),
		s.Muted.Render(fmt.Sprintf("resize to at least %dx%d", minW, minH)),
	)
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, notice)
}
