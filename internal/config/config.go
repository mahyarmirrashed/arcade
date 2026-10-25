// Package config parses runtime flags shared across all arcade entry points.
package config

import (
	"flag"
	"time"
)

// Config holds the runtime configuration for an arcade server.
type Config struct {
	// Addr is the address the SSH server listens on (e.g. ":2222").
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
}

// Parse parses command-line flags into a Config, applying the given defaults
// for addr and hostKeyPath (they differ between cmd/arcade and cmd/snake).
func Parse(defaultAddr, defaultHostKeyPath string) Config {
	var cfg Config

	flag.StringVar(&cfg.Addr, "addr", defaultAddr, "address to listen on")
	flag.StringVar(&cfg.HostKeyPath, "host-key", defaultHostKeyPath, "path to ed25519 host key")
	flag.DurationVar(&cfg.IdleTimeout, "idle-timeout", 30*time.Minute, "disconnect idle sessions after this duration (0 = disabled)")
	flag.DurationVar(&cfg.ShutdownTimeout, "shutdown-timeout", 30*time.Second, "graceful shutdown timeout")

	flag.Parse()
	return cfg
}
