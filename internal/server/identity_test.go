package server_test

import (
	"crypto/ed25519"
	"crypto/rand"
	"net"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/ssh"
	gossh "golang.org/x/crypto/ssh"

	"github.com/mahyarmirrashed/arcade/internal/server"
)

// connect starts a server whose sessions report server.PlayerID, connects to
// it with auth, opens a terminal session, and returns the ID the server saw.
func connect(t *testing.T, auth gossh.AuthMethod) string {
	t.Helper()

	ids := make(chan string, 1)
	handler := func(sess ssh.Session) (tea.Model, []tea.ProgramOption) {
		ids <- server.PlayerID(sess)
		return nil, nil // end the session without running a program
	}
	s, err := server.New(handler)
	if err != nil {
		t.Fatalf("server.New: %v", err)
	}
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go func() { _ = s.Serve(l) }()
	t.Cleanup(func() { _ = s.Close() })

	client, err := gossh.Dial("tcp", l.Addr().String(), &gossh.ClientConfig{
		User:            "player",
		Auth:            []gossh.AuthMethod{auth},
		HostKeyCallback: gossh.InsecureIgnoreHostKey(), // throwaway test server
		Timeout:         5 * time.Second,
	})
	if err != nil {
		t.Fatalf("connecting: %v", err)
	}
	t.Cleanup(func() { _ = client.Close() })

	sess, err := client.NewSession()
	if err != nil {
		t.Fatalf("opening session: %v", err)
	}
	if err := sess.RequestPty("xterm-256color", 24, 80, nil); err != nil {
		t.Fatalf("requesting terminal: %v", err)
	}
	if err := sess.Shell(); err != nil {
		t.Fatalf("starting shell: %v", err)
	}

	select {
	case id := <-ids:
		return id
	case <-time.After(5 * time.Second):
		t.Fatal("server never started the session")
		return ""
	}
}

func TestPlayerID_isTheKeyFingerprint(t *testing.T) {
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	signer, err := gossh.NewSignerFromKey(priv)
	if err != nil {
		t.Fatal(err)
	}

	got := connect(t, gossh.PublicKeys(signer))

	if want := gossh.FingerprintSHA256(signer.PublicKey()); got != want {
		t.Errorf("PlayerID = %q; want the key's fingerprint %q", got, want)
	}
}

func TestPlayerID_playersWithoutAKeyStillGetInAsGuests(t *testing.T) {
	noQuestions := gossh.KeyboardInteractive(
		func(string, string, []string, []bool) ([]string, error) { return nil, nil },
	)

	if got := connect(t, noQuestions); got != "" {
		t.Errorf("PlayerID = %q; want \"\" for a guest", got)
	}
}
