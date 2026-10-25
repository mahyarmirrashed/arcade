# Arcade

A collection of terminal games, playable over SSH.

```
ssh -p 2222 <host>
```

## Running

```
go run ./cmd/arcade
```

Flags:

| Flag | Default | Description |
|------|---------|-------------|
| `--addr` | `127.0.0.1:2222` | address to listen on; pass e.g. `--addr :2222` to accept connections from other machines |
| `--host-key` | `.ssh/id_ed25519` | path to ed25519 host key |
| `--idle-timeout` | `30m` | disconnect idle sessions (`0` = disabled) |
| `--data-dir` | `data` | directory for the leaderboard |

## Terminal requirements

- Recommended size: **80×24** or larger.

## Leaderboards

Connect with an SSH key to be ranked. The first time you connect with a key
you pick a display name; nobody else can post under it, because SSH proves you
hold the key. Without a key you can still play, unranked.

Scores are saved to `<data-dir>/leaderboard.json` after every change and
survive restarts. If that file can't be read, the server refuses to start
rather than overwrite it.

## Controls

Controls are shown in-game. `q` returns to the lobby from any game.
