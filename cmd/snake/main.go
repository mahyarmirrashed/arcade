// Command snake is the standalone Snake SSH server used during parallel
// development. It bypasses the lobby and drops players directly into Snake.
//
// Flags:
//
//	--addr          address to listen on          (default 127.0.0.1:2223)
//	--host-key      path to ed25519 host key      (default .ssh/id_ed25519)
//	--idle-timeout  disconnect idle sessions       (default 30m, 0 = disabled)
//	--shutdown-timeout  graceful shutdown window   (default 30s)
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/log"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"

	"github.com/mahyarmirrashed/arcade/internal/config"
	"github.com/mahyarmirrashed/arcade/internal/games/snake"
	"github.com/mahyarmirrashed/arcade/internal/server"
	"github.com/mahyarmirrashed/arcade/internal/ui"
)

func main() {
	// Listen on loopback unless told otherwise; serving other machines is a
	// deliberate choice made with --addr (e.g. --addr :2223).
	cfg := config.Parse("127.0.0.1:2223", ".ssh/id_ed25519")

	opts := []ssh.Option{
		wish.WithAddress(cfg.Addr),
		wish.WithHostKeyPath(cfg.HostKeyPath),
	}
	if cfg.IdleTimeout > 0 {
		opts = append(opts, wish.WithIdleTimeout(cfg.IdleTimeout))
	}

	s, err := server.New(handler, opts...)
	if err != nil {
		log.Fatal("could not create server", "error", err)
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	log.Info("starting snake server", "addr", cfg.Addr)
	go func() {
		if err := s.ListenAndServe(); err != nil && err != ssh.ErrServerClosed {
			log.Fatal("server error", "error", err)
		}
	}()

	<-done
	log.Info("shutting down", "timeout", cfg.ShutdownTimeout)

	ctx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	if err := s.Shutdown(ctx); err != nil {
		log.Fatal("shutdown error", "error", err)
	}
}

func handler(sess ssh.Session) (tea.Model, []tea.ProgramOption) {
	pty, _, _ := sess.Pty()
	// Styles come from the session's renderer so colours match the player's
	// terminal, not the server's.
	styles := ui.NewStyles(server.Renderer(sess))
	var g snake.Model
	return standalone{g.New(pty.Window.Width, pty.Window.Height, styles)}, []tea.ProgramOption{tea.WithAltScreen()}
}

// standalone runs a game outside the arcade. Games leave q and ctrl+c to the
// arcade's root model, so here they end the session instead.
type standalone struct{ tea.Model }

// Update implements tea.Model.
func (s standalone) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if km, ok := msg.(tea.KeyMsg); ok {
		switch km.String() {
		case "q", "ctrl+c":
			return s, tea.Quit
		}
	}
	next, cmd := s.Model.Update(msg)
	if _, ok := msg.(tea.WindowSizeMsg); ok {
		// Bubble Tea redraws on resize but doesn't clear, so remnants of the
		// old layout could stay on screen.
		cmd = tea.Batch(tea.ClearScreen, cmd)
	}
	return standalone{next}, cmd
}
