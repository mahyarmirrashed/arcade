package server

import (
	"github.com/charmbracelet/ssh"
	gossh "golang.org/x/crypto/ssh"
)

// identify lets every player in while still learning who has an SSH key.
// The arcade is open to anyone, so these handlers accept everything; they
// exist so that players who offer a key are identified by it. SSH itself
// proves the player holds the key's private half, so nobody can claim
// someone else's leaderboard entries. Players without a key get in through
// keyboard-interactive auth with no questions, as guests.
var identify = []ssh.Option{
	ssh.PublicKeyAuth(func(ssh.Context, ssh.PublicKey) bool { return true }),
	ssh.KeyboardInteractiveAuth(func(ssh.Context, gossh.KeyboardInteractiveChallenge) bool { return true }),
}

// PlayerID returns the fingerprint of the key the player connected with, or
// "" for a guest who connected without one.
func PlayerID(sess ssh.Session) string {
	key := sess.PublicKey()
	if key == nil {
		return ""
	}
	return gossh.FingerprintSHA256(key)
}
