// Package server provides the Wish SSH server used by all arcade entry points.
package server

import (
	"time"

	"golang.org/x/time/rate"

	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	"github.com/charmbracelet/wish/activeterm"
	"github.com/charmbracelet/wish/bubbletea"
	"github.com/charmbracelet/wish/logging"
	"github.com/charmbracelet/wish/ratelimiter"
	"github.com/charmbracelet/wish/recover"
)

// New constructs a Wish SSH server with the standard middleware stack wired up.
// The caller supplies the Bubbletea session handler and any additional
// ssh.Options (e.g. wish.WithAddress, wish.WithHostKeyPath).
//
// Middleware order (outermost → innermost):
//
//	logging → ratelimiter → recover → activeterm → bubbletea(handler)
func New(handler bubbletea.Handler, opts ...ssh.Option) (*ssh.Server, error) {
	// Allow 5 new connections per minute per IP, burst of 3.
	rl := ratelimiter.NewRateLimiter(rate.Every(time.Minute/5), 3, 1024)

	middleware := wish.WithMiddleware(
		bubbletea.Middleware(handler),
		activeterm.Middleware(),
		recover.Middleware(),
		ratelimiter.Middleware(rl),
		logging.Middleware(),
	)

	return wish.NewServer(append(opts, middleware)...)
}
