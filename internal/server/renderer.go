package server

import (
	"io"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/ssh"
	"github.com/muesli/termenv"
)

// Renderer returns a Lip Gloss renderer for the session. Like wish's
// bubbletea.MakeRenderer, it detects colour support from the player's TERM and
// environment rather than the server's terminal. Unlike it, it does not ask
// the terminal for its background colour: when a terminal doesn't answer, that
// query waits a second and then swallows the player's first key press, and the
// arcade doesn't use background-dependent colours anyway.
func Renderer(sess ssh.Session) *lipgloss.Renderer {
	pty, _, _ := sess.Pty()
	return newRenderer(sess, pty.Term, sess.Environ())
}

// newRenderer returns a renderer writing to w whose colour profile comes from
// term and the session's environment variables.
func newRenderer(w io.Writer, term string, environ []string) *lipgloss.Renderer {
	if term == "" || term == "dumb" {
		return lipgloss.NewRenderer(w, termenv.WithProfile(termenv.Ascii))
	}
	// TERM goes first so it wins over any TERM the client also sent.
	env := sessionEnviron(append([]string{"TERM=" + term}, environ...))
	return lipgloss.NewRenderer(w,
		termenv.WithEnvironment(env),
		termenv.WithUnsafe(), // w isn't a terminal; trust the environment
		termenv.WithColorCache(true),
	)
}

// sessionEnviron serves a session's environment variables to termenv.
type sessionEnviron []string

func (e sessionEnviron) Environ() []string { return e }

func (e sessionEnviron) Getenv(key string) string {
	for _, kv := range e {
		if v, ok := strings.CutPrefix(kv, key+"="); ok {
			return v
		}
	}
	return ""
}
