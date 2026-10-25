// Package game defines the Game interface and Registry shared across the arcade.
package game

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/mahyarmirrashed/arcade/internal/ui"
)

// Game is implemented by every playable game in the arcade.
// The interface is defined here (the consumer), not in the individual game
// packages — idiomatic Go interface placement.
type Game interface {
	// Name returns the display name shown in the lobby.
	Name() string

	// Description returns a short one-line description shown in the lobby.
	Description() string

	// New returns a fresh Bubbletea model for a new session.
	// width and height are the dimensions of the player's terminal, and styles
	// render for that player's terminal.
	New(width, height int, styles ui.Styles) tea.Model
}

// SizeConstraint is an optional interface a Game may implement to advertise
// the minimum terminal size it requires. The lobby checks this and shows a
// warning when the player's terminal is too small.
type SizeConstraint interface {
	// MinSize returns the minimum terminal width and height required to play.
	MinSize() (width, height int)
}

// Registry holds the set of available games, in registration order.
// It is injected at startup — no global state, no init().
type Registry struct {
	games []Game
}

// NewRegistry returns a Registry containing the provided games in order.
func NewRegistry(games ...Game) *Registry {
	return &Registry{games: games}
}

// All returns the registered games in insertion order.
func (r *Registry) All() []Game {
	return r.games
}
