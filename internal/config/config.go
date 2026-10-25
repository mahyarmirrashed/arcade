// Package config parses runtime flags shared across all arcade entry points.
package config

import (
	"errors"
	"flag"
	"os"
	"time"
)

// Config holds the runtime configuration for an arcade server.
type Config struct {
	// Addr is the address the SSH server listens on (e.g. "127.0.0.1:2222").
	Addr string

	// HostKeyPath is the path to the ed25519 host key file.
	// If the file does not exist, Wish generates one automatically.
	HostKeyPath string

	// IdleTimeout is the maximum duration a session may be idle before it is
	// disconnected. Zero disables the timeout.
	IdleTimeout time.Duration

	// ShutdownTimeout is how long the server waits for active sessions to
	// finish after a SIGINT/SIGTERM before forcefully closing them.
	ShutdownTimeout time.Duration

	// DataDir is where the arcade keeps its leaderboard.
	DataDir string
}

// Parse parses command-line flags into a Config, applying the given defaults
// for addr and hostKeyPath (they differ between cmd/arcade and cmd/snake).
// Like the flag package, it exits on bad flags and after printing -help.
func Parse(defaultAddr, defaultHostKeyPath string) Config {
	cfg, err := ParseArgs(os.Args[1:], defaultAddr, defaultHostKeyPath)
	switch {
	case errors.Is(err, flag.ErrHelp):
		os.Exit(0)
	case err != nil:
		os.Exit(2) // the flag set has already printed the error and usage
	}
	return cfg
}

// ParseArgs parses args into a Config, applying the given defaults for addr
// and hostKeyPath.
func ParseArgs(args []string, defaultAddr, defaultHostKeyPath string) (Config, error) {
	var cfg Config
	fs := flag.NewFlagSet(os.Args[0], flag.ContinueOnError)

	fs.StringVar(&cfg.Addr, "addr", defaultAddr, "address to listen on")
	fs.StringVar(&cfg.HostKeyPath, "host-key", defaultHostKeyPath, "path to ed25519 host key")
	fs.DurationVar(&cfg.IdleTimeout, "idle-timeout", 30*time.Minute, "disconnect idle sessions after this duration (0 = disabled)")
	fs.DurationVar(&cfg.ShutdownTimeout, "shutdown-timeout", 30*time.Second, "graceful shutdown timeout")
	fs.StringVar(&cfg.DataDir, "data-dir", "data", "directory for the leaderboard (cmd/arcade only)")

	err := fs.Parse(args)
	return cfg, err
}
