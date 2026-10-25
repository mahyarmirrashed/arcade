# Arcade

A collection of terminal games, playable over SSH.

```
ssh -p 2222 <host>
```

## Running

```
go run ./cmd/arcade        # lobby on :2222
go run ./cmd/snake         # standalone snake on :2223 (dev)
```

Both binaries accept the following flags:

| Flag | Default | Description |
|------|---------|-------------|
| `--addr` | `:2222` / `:2223` | address to listen on |
| `--host-key` | `.ssh/id_ed25519` | path to ed25519 host key |
| `--idle-timeout` | `30m` | disconnect idle sessions (`0` = disabled) |
| `--shutdown-timeout` | `30s` | graceful shutdown window |

## Terminal requirements

- Minimum size: **80×24**
- **Do not resize your terminal during gameplay.** The layout is fixed to the
  dimensions at connection time. Resizing mid-game will cause the board to
  reflow and may corrupt the display.

## Controls

Controls are shown in-game. `q` returns to the lobby from any game.
