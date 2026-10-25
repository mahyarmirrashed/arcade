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
