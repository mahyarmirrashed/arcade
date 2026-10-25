package server_test

import (
	"context"
	"net"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"

	"github.com/mahyarmirrashed/arcade/internal/server"
)

// nullModel is the simplest valid tea.Model - does nothing.
type nullModel struct{}

func (nullModel) Init() tea.Cmd                        { return nil }
func (nullModel) Update(tea.Msg) (tea.Model, tea.Cmd)  { return nullModel{}, nil }
func (nullModel) View() string                         { return "" }

func nullHandler(_ ssh.Session) (tea.Model, []tea.ProgramOption) {
	return nullModel{}, nil
}

func TestNew_startsAndShutsDown(t *testing.T) {
	// Pick a random free port by binding to :0 then releasing it.
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("find free port: %v", err)
	}
	addr := l.Addr().String()
	l.Close()

	s, err := server.New(nullHandler, wish.WithAddress(addr))
	if err != nil {
		t.Fatalf("server.New: %v", err)
	}

	errCh := make(chan error, 1)
	go func() {
		err := s.ListenAndServe()
		if err != nil && err != ssh.ErrServerClosed {
			errCh <- err
		}
		close(errCh)
	}()

	// Give the server a moment to bind.
	time.Sleep(50 * time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.Shutdown(ctx); err != nil {
		t.Fatalf("shutdown: %v", err)
	}

	if err := <-errCh; err != nil {
		t.Fatalf("server error: %v", err)
	}
}
