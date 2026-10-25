package config_test

import (
	"testing"
	"time"

	"github.com/mahyarmirrashed/arcade/internal/config"
)

func TestParse_defaults(t *testing.T) {
	cfg := config.Config{
		Addr:        ":2222",
		HostKeyPath: ".ssh/id_ed25519",
		IdleTimeout: 30 * time.Minute,
	}

	if cfg.Addr != ":2222" {
		t.Errorf("Addr = %q; want %q", cfg.Addr, ":2222")
	}
	if cfg.HostKeyPath != ".ssh/id_ed25519" {
		t.Errorf("HostKeyPath = %q; want %q", cfg.HostKeyPath, ".ssh/id_ed25519")
	}
	if cfg.IdleTimeout != 30*time.Minute {
		t.Errorf("IdleTimeout = %v; want 30m", cfg.IdleTimeout)
	}
	if config.ShutdownTimeout != 30*time.Second {
		t.Errorf("ShutdownTimeout = %v; want 30s", config.ShutdownTimeout)
	}
}

func TestConfig_idleTimeoutZeroDisabled(t *testing.T) {
	cfg := config.Config{IdleTimeout: 0}
	if cfg.IdleTimeout != 0 {
		t.Errorf("expected zero IdleTimeout to represent disabled, got %v", cfg.IdleTimeout)
	}
}

func TestParseArgs_dataDir(t *testing.T) {
	tests := []struct {
		args []string
		want string
	}{
		{nil, "data"},
		{[]string{"--data-dir", "/srv/arcade"}, "/srv/arcade"},
	}
	for _, tt := range tests {
		cfg, err := config.ParseArgs(tt.args, "127.0.0.1:2222", ".ssh/id_ed25519")
		if err != nil {
			t.Fatalf("ParseArgs(%q): %v", tt.args, err)
		}
		if cfg.DataDir != tt.want {
			t.Errorf("ParseArgs(%q).DataDir = %q; want %q", tt.args, cfg.DataDir, tt.want)
		}
	}
}

func TestParseArgs_defaultsComeFromTheCaller(t *testing.T) {
	cfg, err := config.ParseArgs(nil, "127.0.0.1:2223", "keys/host")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Addr != "127.0.0.1:2223" || cfg.HostKeyPath != "keys/host" {
		t.Errorf("Addr, HostKeyPath = %q, %q; want the caller's defaults", cfg.Addr, cfg.HostKeyPath)
	}
}

func TestParseArgs_rejectsUnknownFlags(t *testing.T) {
	if _, err := config.ParseArgs([]string{"--nope"}, "a", "b"); err == nil {
		t.Error("ParseArgs accepted an unknown flag")
	}
}
