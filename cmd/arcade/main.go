// Command arcade is the production SSH server for the arcade.
// It serves the lobby on :2222; players pick from all registered games.
//
// Flags:
//
//	--addr          address to listen on          (default :2222)
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
	"github.com/mahyarmirrashed/arcade/internal/game"
	"github.com/mahyarmirrashed/arcade/internal/games/snake"
	"github.com/mahyarmirrashed/arcade/internal/games/tetris"
	"github.com/mahyarmirrashed/arcade/internal/root"
	"github.com/mahyarmirrashed/arcade/internal/server"
)


func main() {
	cfg := config.Parse(":2222", ".ssh/id_ed25519")

	reg := game.NewRegistry(
		&tetris.Model{},
		&snake.Model{},
	)

	opts := []ssh.Option{
		wish.WithAddress(cfg.Addr),
		wish.WithHostKeyPath(cfg.HostKeyPath),
	}
	if cfg.IdleTimeout > 0 {
		opts = append(opts, wish.WithIdleTimeout(cfg.IdleTimeout))
	}

	s, err := server.New(handler(reg), opts...)
	if err != nil {
		log.Fatal("could not create server", "error", err)
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	log.Info("starting arcade", "addr", cfg.Addr)
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

func handler(reg *game.Registry) func(ssh.Session) (tea.Model, []tea.ProgramOption) {
	return func(sess ssh.Session) (tea.Model, []tea.ProgramOption) {
		pty, _, _ := sess.Pty()
		m := root.New(pty.Window.Width, pty.Window.Height, reg)
		return m, []tea.ProgramOption{tea.WithAltScreen()}
	}
}
