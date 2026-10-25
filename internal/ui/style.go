// Package ui provides shared Lip Gloss styles and the arcade colour palette.
// All game packages and the lobby import from here to keep a consistent look.
//
// Palette: monochrome — white text, dark grey accents, minimal borders.
// Inspired by terminal.shop's minimal aesthetic.
package ui

import "github.com/charmbracelet/lipgloss"

// Palette.
var (
	ColorText    = lipgloss.Color("255") // bright white
	ColorSubtle  = lipgloss.Color("245") // light grey — secondary text
	ColorMuted   = lipgloss.Color("238") // dark grey — de-emphasised / borders
	ColorInverse = lipgloss.Color("0")   // black — used for selected item bg
)

// Base styles — value types; copy freely.
var (
	// Title is used for screen/section headings.
	Title = lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorText)

	// Subtitle is used for secondary labels.
	Subtitle = lipgloss.NewStyle().
		Foreground(ColorSubtle)

	// Muted is used for de-emphasised text (help hints, empty state).
	Muted = lipgloss.NewStyle().
		Foreground(ColorMuted)

	// Accent is used for highlighted values (scores, selected state).
	Accent = lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorText)

	// Error renders game-over or error messages.
	Error = lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorText).
		Reverse(true)

	// Border wraps a content block with a normal border in the muted colour.
	Border = lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(ColorMuted)

	// Selected renders a selected list item — inverted.
	Selected = lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorInverse).
		Background(ColorText)
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
// server's. Over SSH, get the renderer from wish's bubbletea.MakeRenderer.
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
