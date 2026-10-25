package server

import (
	"io"
	"testing"

	"github.com/muesli/termenv"
)

func TestNewRenderer_detectsColourFromTerm(t *testing.T) {
	tests := []struct {
		name    string
		term    string
		environ []string
		want    termenv.Profile
	}{
		{"256-colour terminal", "xterm-256color", nil, termenv.ANSI256},
		{"truecolor terminal", "xterm-256color", []string{"COLORTERM=truecolor"}, termenv.TrueColor},
		{"dumb terminal", "dumb", nil, termenv.Ascii},
		{"no TERM", "", nil, termenv.Ascii},
		{"player asked for no colour", "xterm-256color", []string{"NO_COLOR=1"}, termenv.Ascii},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := newRenderer(io.Discard, tt.term, tt.environ)

			if got := r.ColorProfile(); got != tt.want {
				t.Errorf("ColorProfile() = %v; want %v", got, tt.want)
			}
		})
	}
}
