// Command arcade is the production SSH server for the arcade.
// It serves the lobby on :2222; players pick from all registered games.
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/log"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"

	"github.com/mahyarmirrashed/arcade/internal/game"
	"github.com/mahyarmirrashed/arcade/internal/games/snake"
	"github.com/mahyarmirrashed/arcade/internal/games/tetris"
	"github.com/mahyarmirrashed/arcade/internal/root"
	"github.com/mahyarmirrashed/arcade/internal/server"
)

const (
	addr    = "127.0.0.1:2222"
	hostKey = ".ssh/id_ed25519"
)

func main() {
	reg := game.NewRegistry(
		&tetris.Model{},
		&snake.Model{},
	)

	s, err := server.New(handler(reg),
		wish.WithAddress(addr),
		wish.WithHostKeyPath(hostKey),
	)
	if err != nil {
		log.Fatal("could not create server", "error", err)
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	log.Info("starting arcade", "addr", addr)
	go func() {
		if err := s.ListenAndServe(); err != nil && err != ssh.ErrServerClosed {
			log.Fatal("server error", "error", err)
		}
	}()

	<-done
	log.Info("shutting down")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
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
