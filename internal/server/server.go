// Package server provides the Wish SSH server used by all arcade entry points.
package server

import (
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	"github.com/charmbracelet/wish/activeterm"
	"github.com/charmbracelet/wish/bubbletea"
	"github.com/charmbracelet/wish/logging"
	"github.com/charmbracelet/wish/recover"
)

// New constructs a Wish SSH server with the standard middleware stack wired up.
// The caller supplies the Bubbletea session handler and any additional
// ssh.Options (e.g. wish.WithAddress, wish.WithHostKeyPath).
//
// Middleware order (outermost → innermost):
//
//	logging → recover → activeterm → bubbletea(handler)
func New(handler bubbletea.Handler, opts ...ssh.Option) (*ssh.Server, error) {
	middleware := wish.WithMiddleware(
		bubbletea.Middleware(handler),
		activeterm.Middleware(),
		recover.Middleware(),
		logging.Middleware(),
	)

	return wish.NewServer(append(opts, middleware)...)
}
