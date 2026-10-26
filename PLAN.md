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
The Nix `devShells.default` provides `go`. `nix build` produces `result/bin/arcade`
via the `buildGoModule` derivation in `flake.nix`.

---

## Project Layout

```
arcade/
├── cmd/
│   └── arcade/
│       └── main.go            # entry point - lobby + all games
├── internal/
│   ├── config/                # runtime flag parsing
│   ├── game/                  # Game interface, Registry, Ranked, Result
│   ├── games/
│   │   ├── 2048/
│   │   ├── minesweeper/
│   │   ├── snake/
│   │   ├── solitaire/
│   │   ├── sudoku/
│   │   ├── tetris/
│   │   └── wordle/
│   ├── leaderboard/           # persistent per-game leaderboard
│   ├── lobby/                 # game-picker screen
│   ├── root/                  # lobby/game state machine + name prompt
│   ├── server/                # Wish SSH server, renderer, player identity
│   └── ui/                    # per-session styles and colour palette
├── flake.nix
├── flake.lock
├── go.mod
├── go.sum
├── PLAN.md
└── README.md
```

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

## Completed stages

All stages are done. For historical context:

- **Stage 1**: Go module, Wish SSH server, shared packages, `cmd/arcade` stub
- **Stage 2**: Lobby, Tetris (you), Snake (friend) in parallel
- **Stage 3**: All games integrated into the lobby
- **Stage 4**: Rate limiting, idle timeouts, graceful shutdown (30s hardcoded), runtime flags (`--addr`, `--host-key`, `--idle-timeout`, `--data-dir`), `buildGoModule` in `flake.nix`
- **Stage 4B**: Persistent leaderboards — SSH key identity, display name prompt, per-game ranking rules, atomic JSON storage
- **Stage 5 (games)**: 2048, Minesweeper, Solitaire (Klondike), Sudoku, Wordle added

## Status

### Single-player games — complete

All seven single-player games are implemented, tested, and considered
production-ready. The codebase has been audited for races, style violations,
and consistency issues; all known issues have been resolved.

| Game        | Ranked by                               |
| ----------- | --------------------------------------- |
| 2048        | best score                              |
| Minesweeper | fastest win                             |
| Snake       | best score (fixed 28×20 board)          |
| Solitaire   | fastest win                             |
| Sudoku      | fastest solve                           |
| Tetris      | best score                              |
| Wordle      | longest win streak, then fewest guesses |

### Multiplayer games — in progress

The room-broker infrastructure is complete and proven by Connect 4 and Pasur.
The following games still need to be implemented:

| Game     | Notes                                                         |
| -------- | ------------------------------------------------------------- |
| Pasur    | Persian fishing card game — broker done, gameplay done, needs polish |
| Go Fish  | Standard Go Fish card game, 2 players, same broker pattern    |
| Hearts   | 4-player trick-taking card game; may need a 4-seat broker variant |

## Backlog

| Game            | Notes                                           |
| --------------- | ----------------------------------------------- |
| Conway's Life   | Simulation/screensaver, no interaction needed   |
| Pong (2-player) | Requires server-side room broker - see below    |

---

## Multiplayer Architecture

Used by Pasur and any future 2+ player games. The pattern is a **server-side
room broker** — each SSH session runs its own `tea.Program` as normal, but
they share a single `Room` that holds the authoritative game state.

### Room lifecycle

```
Player A selects game
    → broker.Create() → Room{code: "A2BF", state: waiting}
    → A's model shows "your code: A2BF, waiting for opponent..."

Player B selects game → sees join prompt → types "A2BF"
    → broker.Join("A2BF") → room transitions to playing
    → broker broadcasts stateMsg to both A and B's channels
    → both programs re-render simultaneously
```

### Room codes

4-character uppercase alphanumeric codes (e.g. `A2BF`). Generated
randomly at room creation. Stored in the broker's map for the lifetime
of the room.

### Disconnect and reconnect

- If either player disconnects, the room enters a **grace period** (60s).
- During grace, the remaining player sees "opponent disconnected, waiting..."
- If the disconnected player reconnects and types their room code within
  the grace period, the game resumes from the exact state it was in.
- If the grace period expires, the room is deleted and both players are
  returned to the lobby.
- The player who stayed sees "opponent did not reconnect" and returns to lobby.

### Broker implementation (`internal/games/<game>/game.go`)

Each multiplayer game owns its own broker. The broker is a package-level
singleton (the only acceptable global in the codebase — it must survive
across sessions):

```go
// Broker is the package-level room registry. Safe for concurrent use.
var Broker = newBroker()

type broker struct {
    mu    sync.Mutex
    rooms map[string]*Room
}

type Room struct {
    Code      string
    State     GameState       // authoritative game state
    Players   [2]chan stateMsg // one channel per player; nil if disconnected
    GraceEnds time.Time       // non-zero during grace period
}

// stateMsg is sent to a player's channel whenever the room state changes.
// The player's tea.Program receives it and calls View() to re-render.
type stateMsg struct {
    Room *Room
}
```

Key rules:
- All mutations to `Room.State` happen under `broker.mu`.
- A player's action sends to the room, which updates state and fans out
  `stateMsg` to both players' channels (buffered, size 1 — drop if full
  to avoid blocking the lock).
- The player's `tea.Program` polls its channel via a `tea.Cmd` that blocks
  on receive, returning the `stateMsg` as a `tea.Msg`.
- The room channel is created fresh each time a player (re)connects;
  the old channel is discarded.

### Per-player model states

Every multiplayer game's `Model` has these states in addition to its
game-specific ones:

```
stateWaiting  - player created a room, showing code, waiting for opponent
stateJoining  - player is typing a room code to join
statePlaying  - both players connected, game in progress
stateGrace    - opponent disconnected, grace period countdown
stateOver     - game finished (win/lose/abandoned)
```

### Integration with game.Game

Multiplayer games implement `game.Game` exactly like single-player games.
`New(width, height, styles)` returns the player's model in `stateWaiting`,
which immediately calls `broker.Create()` and displays the code. The
`q`/`ctrl+c` quit-handling convention still applies — the root model
intercepts quit and returns the player to the lobby, which also triggers
room cleanup.

### Testing

The broker and room logic are unit-tested without real `tea.Program`s by
directly calling the exported broker methods and inspecting `Room.State`.
The `stateMsg` fan-out is tested by checking that both channels receive
after a mutation. No `time.Sleep` — grace period expiry is tested by
injecting a clock.

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

### Help text format

There are two distinct help text styles depending on context.

**In-game controls** (left panel, one binding per line):

```
←→   move
↑/x  rotate cw
spc  hard drop
q    quit
```

Each line is `key  action` — key left-aligned, short action label, no connector
word. Rendered with `styles.Muted`.

**Screen-level hints** (bottom of a screen, single line):

```
j to join room · enter to select · esc to quit
```

Format: `key to action` connected by ` · ` (space-middot-space). Every segment
uses "to" between the key and the action. No angle brackets, no parentheses.
Used in the lobby, waiting screens, name prompt, and any non-game screen.

The distinction: in-game controls are a reference card you glance at while
playing; screen hints are a single line of the 2-3 most important actions on
that screen.

### Resizing

Players may resize at any time, so every screen handles `tea.WindowSizeMsg` in
every state, including game over:

- Track the new size and re-centre. Never reset a game in progress.
- If the layout no longer fits, render
  `styles.TooSmall(width, height, minW, minH)` instead of letting Bubble Tea
  clip it, and pause the game until it fits again.
- The root model clears the screen on resize, so games don't need to.

### Leaderboard results

Every game implements `game.Ranked` (`Ranking()` returns `game.ByScore`,
`game.ByStreak` or `game.ByTime`) and, when a game ends, returns
`game.Finished(m, game.Result{...})`. The root model records the result and
replies with a `game.StandingMsg`, which the game shows on its game-over
screen (`Standing.String()` is short enough for a 20-column board). Games never
touch storage themselves.

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
| Graceful shutdown timeout hardcoded at 30s, not a flag                  | Implementation detail; no deployment needs a different value |
