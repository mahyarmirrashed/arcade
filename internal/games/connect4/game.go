package connect4

import (
	"math/rand"
	"sync"
	"time"
)

const gracePeriod = 60 * time.Second

// playerIndex identifies a seat.
type playerIndex int

const (
	player0 playerIndex = 0
	player1 playerIndex = 1
)

func (p playerIndex) opponent() playerIndex {
	if p == player0 {
		return player1
	}
	return player0
}

func (p playerIndex) piece() cell {
	if p == player0 {
		return p0
	}
	return p1
}

// roomPhase is the overall phase of the room.
type roomPhase int

const (
	phaseWaiting roomPhase = iota
	phasePlaying
	phaseGrace
	phaseOver
)

// GameState holds the authoritative game state for one room.
type GameState struct {
	Board       board
	Turn        playerIndex
	Winner      playerIndex // valid only when Won is true
	Won         bool
	Draw        bool
	WinCells    [][2]int // highlighted winning cells
	LastRow     int      // row of last dropped piece (-1 = none)
	LastCol     int
}

// stateMsg is broadcast to both players when state changes.
type stateMsg struct{ room *Room }

// Room is a live game room.
type Room struct {
	Code      string
	Phase     roomPhase
	Game      GameState
	Cursor    [2]int     // per-player cursor column
	channels  [2]chan stateMsg
	GraceEnds time.Time
	GraceSeat playerIndex
	now       func() time.Time
}

func newGameState() GameState {
	return GameState{LastRow: -1, LastCol: -1}
}

// Broker is the package-level room registry.
var Broker = newBroker()

type broker struct {
	mu    sync.Mutex
	rooms map[string]*Room
	rng   *rand.Rand
}

func newBroker() *broker {
	return &broker{
		rooms: make(map[string]*Room),
		rng:   rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (b *broker) Create() *Room {
	b.mu.Lock()
	defer b.mu.Unlock()
	code := b.uniqueCode()
	r := &Room{
		Code:  code,
		Phase: phaseWaiting,
		now:   time.Now,
	}
	r.channels[0] = make(chan stateMsg, 1)
	b.rooms[code] = r
	return r
}

func (b *broker) Join(code string) (*Room, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	r, ok := b.rooms[code]
	if !ok {
		return nil, false
	}
	switch r.Phase {
	case phaseWaiting:
		r.channels[1] = make(chan stateMsg, 1)
		r.Phase = phasePlaying
		r.Game = newGameState()
		b.broadcast(r)
		return r, true
	case phaseGrace:
		if r.now().Before(r.GraceEnds) && r.GraceSeat == player1 {
			r.channels[1] = make(chan stateMsg, 1)
			r.Phase = phasePlaying
			b.broadcast(r)
			return r, true
		}
	}
	return nil, false
}

func (b *broker) Disconnect(code string, seat playerIndex) {
	b.mu.Lock()
	defer b.mu.Unlock()
	r, ok := b.rooms[code]
	if !ok {
		return
	}
	if r.Phase == phaseOver {
		b.cleanup(code)
		return
	}
	r.channels[seat] = nil
	r.Phase = phaseGrace
	r.GraceSeat = seat
	r.GraceEnds = r.now().Add(gracePeriod)
	b.broadcast(r)
	go func() {
		time.Sleep(gracePeriod + time.Second)
		b.mu.Lock()
		defer b.mu.Unlock()
		if r2, ok := b.rooms[code]; ok && r2.Phase == phaseGrace {
			r2.Phase = phaseOver
			b.broadcast(r2)
			b.cleanup(code)
		}
	}()
}

// Drop drops a piece in col for seat. Returns false if the move is invalid.
func (b *broker) Drop(code string, seat playerIndex, col int) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	r, ok := b.rooms[code]
	if !ok || r.Phase != phasePlaying || r.Game.Turn != seat {
		return false
	}
	if colFull(r.Game.Board, col) {
		return false
	}
	newBoard, row := drop(r.Game.Board, col, seat.piece())
	r.Game.Board = newBoard
	r.Game.LastRow = row
	r.Game.LastCol = col

	if checkWin(r.Game.Board, row, col) {
		r.Game.Won = true
		r.Game.Winner = seat
		r.Game.WinCells = winningCells(r.Game.Board, row, col)
		r.Phase = phaseOver
	} else if boardFull(r.Game.Board) {
		r.Game.Draw = true
		r.Phase = phaseOver
	} else {
		r.Game.Turn = seat.opponent()
	}

	b.broadcast(r)
	return true
}

// MoveCursor updates the cursor column for seat and broadcasts.
func (b *broker) MoveCursor(code string, seat playerIndex, col int) {
	b.mu.Lock()
	defer b.mu.Unlock()
	r, ok := b.rooms[code]
	if !ok || r.Phase != phasePlaying {
		return
	}
	r.Cursor[seat] = col
	b.broadcast(r)
}

func (b *broker) broadcast(r *Room) {
	msg := stateMsg{room: r}
	for _, ch := range r.channels {
		if ch == nil {
			continue
		}
		select {
		case ch <- msg:
		default:
			select {
			case <-ch:
			default:
			}
			ch <- msg
		}
	}
}

func (b *broker) cleanup(code string) {
	delete(b.rooms, code)
}

func (b *broker) uniqueCode() string {
	const chars = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	for {
		code := make([]byte, 4)
		for i := range code {
			code[i] = chars[b.rng.Intn(len(chars))]
		}
		s := string(code)
		if _, exists := b.rooms[s]; !exists {
			return s
		}
	}
}
