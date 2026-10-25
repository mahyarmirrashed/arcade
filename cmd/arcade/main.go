// Command arcade is the production SSH server for the arcade.
// It serves the lobby on 127.0.0.1:2222; players pick from all registered games.
//
// Flags:
//
//	--addr          address to listen on          (default 127.0.0.1:2222)
//	--host-key      path to ed25519 host key      (default .ssh/id_ed25519)
//	--idle-timeout  disconnect idle sessions       (default 30m, 0 = disabled)
//	--data-dir      where the leaderboard is kept  (default data)
package main

import (
	"context"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/log"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"

	"github.com/mahyarmirrashed/arcade/internal/config"
	"github.com/mahyarmirrashed/arcade/internal/game"
	_2048 "github.com/mahyarmirrashed/arcade/internal/games/2048"
	"github.com/mahyarmirrashed/arcade/internal/games/minesweeper"
	"github.com/mahyarmirrashed/arcade/internal/games/pasur"
	"github.com/mahyarmirrashed/arcade/internal/games/snake"
	"github.com/mahyarmirrashed/arcade/internal/games/solitaire"
	"github.com/mahyarmirrashed/arcade/internal/games/sudoku"
	"github.com/mahyarmirrashed/arcade/internal/games/tetris"
	"github.com/mahyarmirrashed/arcade/internal/games/wordle"
	"github.com/mahyarmirrashed/arcade/internal/leaderboard"
	"github.com/mahyarmirrashed/arcade/internal/root"
	"github.com/mahyarmirrashed/arcade/internal/server"
	"github.com/mahyarmirrashed/arcade/internal/ui"
)

func main() {
	// Listen on loopback unless told otherwise; serving other machines is a
	// deliberate choice made with --addr (e.g. --addr :2222).
	cfg := config.Parse("127.0.0.1:2222", ".ssh/id_ed25519")

	// Games are listed alphabetically in the lobby.
	reg := game.NewRegistry(
		&_2048.Model{},
		&minesweeper.Model{},
		&pasur.Model{},
		&snake.Model{},
		&solitaire.Model{},
		&sudoku.Model{},
		&tetris.Model{},
		&wordle.Model{},
	)

	// The leaderboard lives outside the repo and survives restarts. A file
	// that can't be read stops the server rather than being overwritten.
	board, err := leaderboard.Open(filepath.Join(cfg.DataDir, "leaderboard.json"))
	if err != nil {
		log.Fatal("could not open leaderboard", "error", err)
	}

	opts := []ssh.Option{
		wish.WithAddress(cfg.Addr),
		wish.WithHostKeyPath(cfg.HostKeyPath),
	}
	if cfg.IdleTimeout > 0 {
		opts = append(opts, wish.WithIdleTimeout(cfg.IdleTimeout))
	}

	s, err := server.New(handler(reg, board), opts...)
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
	log.Info("shutting down")

	ctx, cancel := context.WithTimeout(context.Background(), config.ShutdownTimeout)
	defer cancel()

	if err := s.Shutdown(ctx); err != nil {
		log.Fatal("shutdown error", "error", err)
	}
}

func handler(reg *game.Registry, board *leaderboard.Board) func(ssh.Session) (tea.Model, []tea.ProgramOption) {
	return func(sess ssh.Session) (tea.Model, []tea.ProgramOption) {
		pty, _, _ := sess.Pty()
		// Styles come from the session's renderer so colours match the
		// player's terminal, not the server's.
		styles := ui.NewStyles(server.Renderer(sess))
		// Players are known by their SSH key; without one they're a guest.
		lb := board.Session(server.PlayerID(sess))
		m := root.New(pty.Window.Width, pty.Window.Height, reg, styles, lb)
		return m, []tea.ProgramOption{tea.WithAltScreen()}
	}
}
