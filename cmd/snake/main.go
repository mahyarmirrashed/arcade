// Command snake is the standalone Snake SSH server used during parallel
// development. It bypasses the lobby and drops players directly into Snake.
//
// Connect with: ssh -p 2223 localhost
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

	"github.com/mahyarmirrashed/arcade/internal/games/snake"
	"github.com/mahyarmirrashed/arcade/internal/server"
)

const (
	addr    = "127.0.0.1:2223"
	hostKey = ".ssh/id_ed25519"
)

func main() {
	s, err := server.New(handler,
		wish.WithAddress(addr),
		wish.WithHostKeyPath(hostKey),
	)
	if err != nil {
		log.Fatal("could not create server", "error", err)
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	log.Info("starting snake server", "addr", addr)
	go func() {
		if err := s.ListenAndServe(); err != nil && err != ssh.ErrServerClosed {
			log.Fatal("server error", "error", err)
		}
	}()

	<-done
	log.Info("shutting down")

	ctx, cancel := context.WithTimeout(context.Background(), 30*1e9) // 30s
	defer cancel()

	if err := s.Shutdown(ctx); err != nil {
		log.Fatal("shutdown error", "error", err)
	}
}

func handler(sess ssh.Session) (tea.Model, []tea.ProgramOption) {
	pty, _, _ := sess.Pty()
	var g snake.Model
	return g.New(pty.Window.Width, pty.Window.Height), []tea.ProgramOption{tea.WithAltScreen()}
}
