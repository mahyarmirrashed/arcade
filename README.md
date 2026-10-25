# Arcade

A collection of terminal games, playable over SSH.

```
ssh -p 2222 <host>
```

## Running

```
go run ./cmd/arcade        # lobby on 127.0.0.1:2222
go run ./cmd/snake         # standalone snake on 127.0.0.1:2223 (dev)
```

Both binaries accept the following flags:

| Flag | Default | Description |
|------|---------|-------------|
| `--addr` | `127.0.0.1:2222` / `127.0.0.1:2223` | address to listen on; pass e.g. `--addr :2222` to accept connections from other machines |
| `--host-key` | `.ssh/id_ed25519` | path to ed25519 host key |
| `--idle-timeout` | `30m` | disconnect idle sessions (`0` = disabled) |
| `--shutdown-timeout` | `30s` | graceful shutdown window |

## Terminal requirements

- Recommended size: **80×24** or larger.
- Resizing is supported at any time. If the terminal becomes too small for the
  current screen, it shows the size it needs and the game pauses: Snake waits
  for `p` once it fits again, Tetris carries on by itself. A new Snake game
  sizes its board to the terminal.

## Controls

Controls are shown in-game. `q` returns to the lobby from any game.
