# Arcade — Implementation Plan

An SSH-accessible terminal arcade built in Go using the Charm.sh ecosystem.
Players connect with `ssh -p 2222 <host>` and are greeted by a lobby from which
they pick a game. Each SSH session is fully isolated.

---

## Technology Stack

| Role               | Library   | Import path                          |
| ------------------ | --------- | ------------------------------------ |
| SSH server         | Wish      | `github.com/charmbracelet/wish`      |
| TUI framework      | Bubbletea | `github.com/charmbracelet/bubbletea` |
| Styling            | Lip Gloss | `github.com/charmbracelet/lipgloss`  |
| UI components      | Bubbles   | `github.com/charmbracelet/bubbles`   |
| Structured logging | Charm Log | `github.com/charmbracelet/log`       |
| SSH primitives     | Charm SSH | `github.com/charmbracelet/ssh`       |

All dependencies managed via Go modules (`go.mod` / `go.sum`).  
The Nix `devShells.default` already provides `go`; no additional Nix packages
are required for development. A `buildGoModule` derivation will be added in
Stage 4 for reproducible builds.

---

## Project Layout

```
arcade/
├── cmd/
│   ├── arcade/
│   │   └── main.go            # Production entry point — lobby + all games
│   └── snake/
│       └── main.go            # Standalone Snake entry point (dev / parallel work)
├── internal/
│   ├── server/
│   │   └── server.go          # Wish SSH server constructor (shared)
│   ├── lobby/
│   │   ├── model.go           # Bubbletea model for game-picker screen
│   │   └── model_test.go
│   ├── game/
│   │   ├── game.go            # Game interface + Registry
│   │   └── game_test.go
│   ├── games/
│   │   ├── snake/
│   │   │   ├── model.go       # Snake Bubbletea model
│   │   │   └── model_test.go
│   │   └── tetris/
│   │       ├── model.go       # Tetris Bubbletea model
│   │       └── model_test.go
│   └── ui/
│       └── style.go           # Shared Lip Gloss styles and theme
├── testdata/                  # Golden files for render tests
├── flake.nix
├── flake.lock
├── go.mod
├── go.sum
├── PLAN.md
└── README.md
```

---

## Parallel Development Strategy

Two people work simultaneously without blocking each other:

| Person | Owns           | Entry point                           |
| ------ | -------------- | ------------------------------------- |
| You    | Lobby + Tetris | `cmd/arcade` (full arcade)            |
| Friend | Snake          | `cmd/snake` (standalone Snake server) |

### How `cmd/snake` works

`cmd/snake/main.go` wires the shared `internal/server` package directly to the
Snake model — no lobby, no registry. It starts its own SSH server (default
`:2223` to avoid collision with `cmd/arcade` on `:2222`):

```
ssh -p 2223 localhost   →   Snake game, immediately
ssh -p 2222 localhost   →   Arcade lobby (pick a game)
```

Once the lobby is ready and Snake is complete, Snake is registered in the
central `internal/game` registry and surfaced through `cmd/arcade`. The
standalone `cmd/snake` binary is kept permanently as a useful dev/debug tool.

### Shared contracts that must be agreed upon before parallel work begins

1. The `Game` interface (see below) — your friend implements it on
   `*snake.Model`; you consume it in the lobby.
2. The `internal/ui/style.go` colour theme — agree on palette upfront so both
   games look consistent.
3. `internal/server.NewServer` signature — shared between both `cmd/` entry
   points; changes here affect both.

---

## Core Abstractions

### The `Game` Interface (`internal/game/game.go`)

Defined at the consumer (lobby), not in the game packages — idiomatic Go:

```go
// Game is implemented by every playable game in the arcade.
type Game interface {
    // Name returns the display name shown in the lobby.
    Name() string
    // Description returns a short one-line description shown in the lobby.
    Description() string
    // New returns a fresh Bubbletea model for a new session.
    // width and height are the dimensions of the player's terminal, and
    // styles render for that player's terminal (see Shared styles below).
    New(width, height int, styles ui.Styles) tea.Model
}
```

Compile-time interface check in each game package:

```go
// internal/games/snake/model.go
var _ game.Game = (*Model)(nil)
```

### Game Registry (`internal/game/registry.go`)

Simple slice, injected at startup — no `init()`, no globals:

```go
type Registry struct{ games []Game }

func NewRegistry(games ...Game) *Registry { return &Registry{games: games} }
func (r *Registry) All() []Game           { return r.games }
```

### Session Flow (full arcade)

```
SSH connect
    │
    ▼
Wish middleware chain
(logging → activeterm → recover → bubbletea)
    │
    ▼
Root Bubbletea model
    │
    ├── state: lobby  →  lobby.Model  (game picker)
    │                         │
    │                    player selects game
    │                         │
    └── state: playing →  game.Model  (the game itself)
                               │
                          player quits / game over
                               │
                          return to lobby
```

Navigation is handled by a root model that delegates to whichever child is
active. A single `tea.Program` runs per SSH session for the lifetime of the
connection.

---

## Stages

### Stage 1 — Bootstrap (both people, ~day 1)

**Goal:** Go module initialised, shared packages exist, both entry points
compile and accept SSH connections.

**Tasks:**

1. `go mod init github.com/mahyarmirrashed/arcade`
2. `go get github.com/charmbracelet/wish github.com/charmbracelet/bubbletea github.com/charmbracelet/lipgloss github.com/charmbracelet/bubbles github.com/charmbracelet/log`
3. `internal/server/server.go` —
   `NewServer(addr string, handler ...) *wish.Server` using functional options;
   middleware: `logging`, `activeterm`, `recover`, `bubbletea`
4. `internal/game/game.go` — `Game` interface + `Registry` (agreed contract)
5. `internal/ui/style.go` — shared colour palette and base Lip Gloss styles
6. `cmd/arcade/main.go` — stub lobby model, server on `:2222`
7. `cmd/snake/main.go` — stub "coming soon" model, server on `:2223`
8. Both binaries compile: `go build ./...`

**Tests:**

- `internal/server`: `NewServer` starts without error on a random port
- `internal/game`: `Registry.All()` returns games in insertion order

---

### Stage 2A — Snake (friend, parallel with 2B)

**Goal:** Fully playable Snake accessible via `ssh -p 2223 localhost`.

**Game mechanics:**

- Grid fills the terminal (width × height minus header/footer rows)
- Tick-based game loop; tick rate increases with score
- Arrow keys + vi-keys (`h j k l`) for direction; cannot reverse 180°
- Snake grows on eating food; collision with wall or self ends the game
- Score displayed in header bar

**Tasks:**

1. `internal/games/snake/model.go` — `Model` struct with fields: body (slice of
   positions), direction, food, score, alive, width, height
2. `Init()` returns first tick command
3. `Update()` handles `TickMsg` (advance), `tea.KeyPressMsg` (steer),
   `tea.WindowSizeMsg` (resize + reset)
4. `View()` renders grid + score using shared `ui/style` theme
5. `cmd/snake/main.go` updated to use real `snake.Model`
6. Compile-time check: `var _ game.Game = (*Model)(nil)`

**Tests (`internal/games/snake/model_test.go`):**

- Collision: wall hit → not alive; self hit → not alive; open space → alive
- Direction: reverse direction input ignored (table-driven, all 4 directions)
- Growth: eating food increments score and length by 1
- Tick advance: head moves one cell in current direction
- Food placement: always on an empty cell, never on snake body

---

### Stage 2B — Lobby + Tetris (you, parallel with 2A)

**Goal:** Interactive lobby served at `:2222`; Tetris playable from it.

#### Lobby (`internal/lobby/model.go`)

- `bubbles/list` component for game selection
- `bubbles/key` + `bubbles/help` for keybinding display
- Keys: `↑/↓` or `k/j` navigate, `enter` select, `q` quit
- Receives `*game.Registry`; list entries built from `registry.All()`
- On select: emits a `SelectMsg{Game: game.Game}` that the root model handles
- `tea.WindowSizeMsg` propagated to list component for responsive sizing

**Tests (`internal/lobby/model_test.go`):**

- Up/down keys change selected index
- Enter emits `SelectMsg` with correct game
- Window resize propagates to list dimensions

#### Tetris (`internal/games/tetris/model.go`)

**Game mechanics:**

- Standard 10×20 board; `[20][10]cell` — zero value = empty
- 7 tetrominoes (I O T S Z J L) with SRS wall-kick rotation
- Controls: `←/→` move, `↑` rotate CW, `z` rotate CCW, `↓` soft drop, `space`
  hard drop, `c` hold
- Gravity tick rate increases per level (every 10 lines)
- Scoring: 1 line = 100×level, 2 = 300×level, 3 = 500×level, 4 = 800×level

**Tasks:**

1. Piece definitions: rotation matrices for all 7 tetrominoes
2. SRS wall-kick offset tables
3. `Model` struct: board, current piece, next piece, hold piece, score, level,
   lines, alive, width, height
4. `Init()` spawns first piece + starts gravity tick
5. `Update()` handles gravity tick, all key inputs, line clear, game over
6. `View()` renders board + next preview + hold slot + score/level/lines panel
7. Compile-time check: `var _ game.Game = (*Model)(nil)`

**Tests (`internal/games/tetris/model_test.go`):**

- Rotation: all 4 rotations of each piece produce correct cell patterns
- Wall-kick: rotation near left/right wall succeeds without going out of bounds
- Line clear: 1/2/3/4 simultaneous clears collapse correct rows (table-driven)
- Scoring: correct points for 1–4 line clears at varying levels (table-driven)
- Collision: piece lands on floor and on stacked pieces correctly
- Game over: spawn into occupied cells → game-over state
- Gravity tick: current piece advances one row

---

### Stage 3 — Integration

**Goal:** Snake and Tetris both reachable through the lobby at `:2222`.

**Tasks:**

1. Register both games in `cmd/arcade/main.go`:
   ```go
   reg := game.NewRegistry(
       snake.New(),
       tetris.New(),
   )
   ```
2. Root model in `cmd/arcade/main.go`: holds `lobby.Model` or active
   `game.Model`; handles `lobby.SelectMsg` to transition into a game and
   `game.QuitMsg` (or `ctrl+c`) to return to lobby
3. Smoke-test: connect via SSH, navigate lobby, play both games, return to lobby
   after each

**Tests:**

- Root model: `SelectMsg` transitions state from lobby to game
- Root model: quit from game transitions back to lobby
- Both `snake.Model` and `tetris.Model` satisfy `game.Game` (compile-time)

---

### Stage 4 — Polish & Hardening

**Goal:** Production-ready server; reproducible Nix build.

**Tasks:**

1. Rate limiting: `github.com/charmbracelet/wish/ratelimiter` middleware
2. Idle/max session timeouts: `wish.WithIdleTimeout`, `wish.WithMaxTimeout`
3. Graceful shutdown: catch `SIGINT`/`SIGTERM`, call `server.Shutdown(ctx)` with
   30-second timeout
4. Runtime flags: `--addr` (default `:2222`), `--host-key` (default
   `.ssh/id_ed25519`)
5. Add `buildGoModule` derivation to `flake.nix` for `cmd/arcade`
6. `README.md`: install, run, connect instructions

**Tests:**

- Integration: server starts on random port, accepts connection, shuts down
  cleanly within timeout

---

### Stage 5 — Future Games (Backlog)

Adding a game = implement `Game` interface + one line in `cmd/arcade/main.go`.

| Game            | Notes                                       |
| --------------- | ------------------------------------------- |
| Minesweeper     | Grid-based; no tick loop needed             |
| 2048            | Slide-and-merge; simple state machine       |
| Wordle          | Text-based; no tick loop needed             |
| Conway's Life   | Simulation/screensaver                      |
| Pong (2-player) | Requires server-side room broker — deferred |

---

## Commit Format

All commits follow the pattern observed in the repository history:

```
<filename>: <verb>
```

Rules:

- **All lowercase, no exceptions** — this includes file names that are
  conventionally uppercase (e.g. `README.md` is written `readme.md` in the
  commit message)
- Filename is the primary file changed (e.g. `plan.md`, `flake.nix`, `go.mod`)
- For a directory or package, use the path (e.g. `internal/server: init`)
- Verb is a short imperative word: `init`, `add`, `fix`, `remove`, `update`
- No period at the end

Examples:

```
go.mod: init
internal/game: add registry
internal/games/snake: init
cmd/arcade: add lobby integration
flake.nix: add buildGoModule derivation
readme.md: add setup instructions
```

---

## Testing Philosophy

- **TDD**: write the failing test first, then implement
- **Table-driven tests** for all pure game logic (collision, rotation, scoring,
  line clear)
- **Compile-time interface checks** for every `Game` implementation
- **No `time.Sleep`** in tests — inject a clock interface or use deterministic
  tick commands
- **Race detector** on every run: `go test -race ./...`
- **Coverage target**: ≥ 80 % on `internal/game` and all `internal/games/*`

---

## UI & Styling Conventions

These conventions are established in the Tetris implementation and **must be
followed by all game implementations** (including Snake) before merging into
`main`.

### Palette — `internal/ui/style.go`

Do not define colours inline in game packages. Use only the shared variables:

| Variable | ANSI | Usage |
|----------|------|-------|
| `ui.ColorText` | 255 (bright white) | primary text, selected state |
| `ui.ColorSubtle` | 245 (light grey) | secondary labels |
| `ui.ColorMuted` | 238 (dark grey) | de-emphasised text, borders, help |
| `ui.ColorInverse` | 0 (black) | selected item background |
| `ui.Spectrum` | 196, 208, 226, 46, 51, 21, 129 | game pieces in rainbow order (matches the Tetris piece colours; Snake's segments) |

### Shared styles

Styles are built **per SSH session**, never globally. A global Lip Gloss style
takes its colour support from the server's terminal, so a server started in the
background (or deployed) would send every player plain text. Each SSH handler
builds the session's styles once and passes them down:

```go
styles := ui.NewStyles(server.Renderer(sess)) // colour support from the player's TERM
m := root.New(width, height, reg, styles)      // root hands them to the lobby and Game.New
```

`server.Renderer` is used instead of wish's `bubbletea.MakeRenderer` because
the latter asks the terminal for its background colour; when a terminal doesn't
answer, the player's first key press is swallowed.

Games keep the `ui.Styles` they are given and render only with it:

| Field | Usage |
|-------|-------|
| `styles.Title` | Game name heading — bold, `ColorText` |
| `styles.Subtitle` | Section labels (score, level, next, hold) — `ColorSubtle` |
| `styles.Muted` | Help text, empty state — `ColorMuted` |
| `styles.Accent` | Highlighted values — bold, `ColorText` |
| `styles.Error` | Game-over message — bold, reversed |
| `styles.Border` | Content border — `NormalBorder`, `ColorMuted` |
| `styles.Selected` | Selected list item — bold, inverted |

For anything else (piece colours, padding), use `styles.NewStyle()`, never
`lipgloss.NewStyle()`. Tests build styles from a renderer with a fixed colour
profile, e.g. `lipgloss.NewRenderer(io.Discard)` with `termenv.Ascii`.

### Game layout structure

Every game `View()` must follow this pattern:

```
┌──────────────┬──────────────────────┬─────────────┐
│              │                      │             │
│  help text   │  title               │  side panel │
│  (bottom     │  (above board)       │  (bottom    │
│   aligned)   │                      │   aligned)  │
│              │  ┌──────────────┐    │             │
│              │  │              │    │             │
│              │  │    board     │    │             │
│              │  │              │    │             │
│              │  └──────────────┘    │             │
└──────────────┴──────────────────────┴─────────────┘
```

- `JoinHorizontal(lipgloss.Bottom, help, "  ", boardWithTitle, "  ", side)`
- Title rendered with `styles.Title` above the board using `JoinVertical`
- Help and side panels padded `PaddingBottom(1)` to align with the board's
  inner content (the border adds 1 row top and bottom)
- All multi-line content built with `lipgloss.JoinVertical` — no raw `\n`
  concatenation

### Quit handling

Games must **not** call `tea.Quit` or handle `q`/`ctrl+c` themselves.
The root model (`internal/root`) intercepts those keys and returns the player
to the lobby. Games only need to handle their own game-over state.

### Game name casing

Game names are lowercase everywhere — in `Name()`, titles, commit messages,
and display strings. `"tetris"`, `"snake"`, not `"Tetris"`, `"Snake"`.

> Exception: `Name()` returns the display name used in the lobby list. It may
> be capitalised there if desired (e.g. `"Snake"`), but the `styles.Title.Render`
> call in `View()` should always use lowercase.

---

## Key Idiomatic Go Decisions

| Decision                                                                | Rationale                                                 |
| ----------------------------------------------------------------------- | --------------------------------------------------------- |
| `Game` interface defined in `internal/game` (consumed by lobby)         | Interfaces belong where they are used                     |
| Functional options on `NewServer`                                       | Extensible config, no breaking API changes                |
| Pure functions for game logic (collision, rotation, line clear)         | Trivially unit-testable without constructing a full model |
| Zero-value-useful model structs                                         | `New(w, h)` is the only constructor needed                |
| `context.Context` for shutdown                                          | Idiomatic Go cancellation                                 |
| No global mutable state; registry injected at `main.go`                 | No `init()` surprises; fully testable                     |
| Single `tea.Program` per SSH session; root model owns lobby↔game switch | Avoids nested-program complexity                          |
| `cmd/snake` standalone binary kept permanently                          | Useful dev tool; zero coupling cost                       |
