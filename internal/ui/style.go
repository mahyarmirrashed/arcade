// Package ui provides shared Lip Gloss styles and the arcade colour palette.
// All game packages and the lobby import from here to keep a consistent look.
package ui

import "github.com/charmbracelet/lipgloss"

// Palette — agree on these before parallel development begins.
var (
	ColorPrimary   = lipgloss.Color("99")  // violet
	ColorSecondary = lipgloss.Color("212") // pink
	ColorAccent    = lipgloss.Color("86")  // cyan
	ColorMuted     = lipgloss.Color("241") // grey
	ColorText      = lipgloss.Color("255") // near-white
	ColorError     = lipgloss.Color("196") // red
)

// Base styles — copy these, then call additional methods on the copy.
// Lip Gloss styles are value types; copying is safe and cheap.
var (
	// Title is used for screen headings.
	Title = lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorPrimary)

	// Subtitle is used for secondary headings and labels.
	Subtitle = lipgloss.NewStyle().
		Foreground(ColorSecondary)

	// Muted is used for de-emphasised text such as keybinding hints.
	Muted = lipgloss.NewStyle().
		Foreground(ColorMuted)

	// Accent highlights values like scores or selected items.
	Accent = lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorAccent)

	// Error renders error or game-over messages.
	Error = lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorError)

	// Border wraps a content block with a rounded border in the primary colour.
	Border = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(ColorPrimary)
)
