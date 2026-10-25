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
| `--data-dir` | `data` | where `cmd/arcade` keeps the leaderboard |

## Terminal requirements

- Recommended size: **80×24** or larger.
- Resizing is supported at any time. If the terminal becomes too small for the
  current screen, it shows the size it needs and the game pauses: Snake waits
  for `p` once it fits again, Tetris carries on by itself.

## Leaderboards

Connect with an SSH key to be ranked. The first time you connect with a key
you pick a display name; nobody else can post under it, because SSH proves you
hold the key. Without a key you can still play, unranked.

| Game | Ranked by |
|------|-----------|
| 2048, Snake, Tetris | best score |
| Wordle | longest winning streak, then fewest guesses |
| Minesweeper, Solitaire | fastest win, timed from your first move |

The lobby shows the top 10 for the highlighted game, and the game-over screen
shows where your result placed. Every Snake game uses the same 28×20 board so
scores compare fairly.

Scores are saved to `<data-dir>/leaderboard.json` after every change and
survive restarts. If that file can't be read, the server refuses to start
rather than overwrite it. The standalone `cmd/snake` doesn't record scores.

## Controls

Controls are shown in-game. `q` returns to the lobby from any game.
