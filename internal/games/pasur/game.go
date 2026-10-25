package pasur

import (
	"math/rand"
	"sync"
	"time"
)

const (
	gracePeriod  = 60 * time.Second
	winScore     = 62
	handSize     = 4
	initialPool  = 4
)

// playerIndex identifies which seat a player occupies.
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

// stateMsg is broadcast to a player's channel whenever the room state changes.
type stateMsg struct{ room *Room }

// roomPhase is the overall phase of the room.
type roomPhase int

const (
	phaseWaiting    roomPhase = iota // host created, waiting for guest
	phasePlaying                     // both connected, game in progress
	phaseGrace                       // one player disconnected
	phaseOver                        // game finished
)

// GameState holds the full authoritative Pasur game state for one room.
type GameState struct {
	Deck     []Card
	Pool     []Card // face-up cards on the table
	Hands    [2][]Card
	Captured [2][]Card
	Surs     [2]int
	Scores   [2]int // accumulated across rounds
	Turn     playerIndex
	LastDeal bool // true on the final deal of the deck
	DealsDone int
}

// Room is a live game room holding two players.
type Room struct {
	Code      string
	Phase     roomPhase
	Game      GameState
	channels  [2]chan stateMsg // one per seat; nil if disconnected
	GraceEnds time.Time
	GraceSeat playerIndex // which seat is absent during grace
	now       func() time.Time

	// selectedCard and captureOpts are ephemeral UI state that must be
	// broadcast so both players see the same highlight.
	SelectedCard *int    // index into current player's hand; nil = none
	CaptureOpts  [][]int // capture options for selected card against pool
}

// newGameState deals a fresh game.
func newGameState(rng *rand.Rand) GameState {
	deck := newDeck(rng)
	gs := GameState{}

	// Deal 4 cards to each player and 4 to the pool.
	// Re-deal if any pool card is a Jack.
	for {
		deck = newDeck(rng)
		pool := deck[:initialPool]
		hasJack := false
		for _, c := range pool {
			if c.Rank == Jack {
				hasJack = true
				break
			}
		}
		if !hasJack {
			break
		}
	}

	gs.Pool = make([]Card, initialPool)
	copy(gs.Pool, deck[:initialPool])
	gs.Hands[0] = make([]Card, handSize)
	copy(gs.Hands[0], deck[initialPool:initialPool+handSize])
	gs.Hands[1] = make([]Card, handSize)
	copy(gs.Hands[1], deck[initialPool+handSize:initialPool+2*handSize])
	gs.Deck = deck[initialPool+2*handSize:]
	gs.Turn = player0
	gs.DealsDone = 1
	return gs
}

// dealHands deals 4 more cards to each player from the deck.
// Returns false if the deck is exhausted.
func (gs *GameState) dealHands(rng *rand.Rand) bool {
	_ = rng // rng not needed; deck is already shuffled
	if len(gs.Deck) < 2*handSize {
		return false
	}
	gs.Hands[0] = append(gs.Hands[0], gs.Deck[:handSize]...)
	gs.Hands[1] = append(gs.Hands[1], gs.Deck[handSize:2*handSize]...)
	gs.Deck = gs.Deck[2*handSize:]
	gs.DealsDone++
	gs.LastDeal = len(gs.Deck) < 2*handSize
	return true
}

// playCard executes a play for the current player: play card at handIdx,
// capturing the pool cards at captureIdxs (may be nil = no capture).
// Returns whether a sur was scored.
func (gs *GameState) playCard(handIdx int, captureIdxs []int, rng *rand.Rand) (sur bool) {
	p := gs.Turn
	card := gs.Hands[p][handIdx]

	// Remove card from hand.
	gs.Hands[p] = append(gs.Hands[p][:handIdx], gs.Hands[p][handIdx+1:]...)

	if len(captureIdxs) == 0 {
		// No capture - add card to pool.
		gs.Pool = append(gs.Pool, card)
	} else {
		// Capture.
		captured := []Card{card}
		newPool := make([]Card, 0, len(gs.Pool))
		captSet := make(map[int]bool, len(captureIdxs))
		for _, i := range captureIdxs {
			captSet[i] = true
		}
		for i, c := range gs.Pool {
			if captSet[i] {
				captured = append(captured, c)
			} else {
				newPool = append(newPool, c)
			}
		}
		if isSur(card, len(captureIdxs), len(gs.Pool), gs.LastDeal) {
			gs.Surs[p]++
			sur = true
		}
		gs.Pool = newPool
		gs.Captured[p] = append(gs.Captured[p], captured...)
	}

	// Advance turn.
	gs.Turn = p.opponent()

	// Deal new hands when both are empty.
	if len(gs.Hands[0]) == 0 && len(gs.Hands[1]) == 0 {
		gs.dealHands(rng)
	}

	return sur
}

// scoreRound tallies points for the round and resets for the next.
// Returns [2]int of points earned this round.
func (gs *GameState) scoreRound() [2]int {
	clubs0 := clubCount(gs.Captured[0])
	clubs1 := clubCount(gs.Captured[1])
	var earned [2]int
	earned[0] = roundScore(gs.Captured[0], clubs0 > clubs1, gs.Surs[0])
	earned[1] = roundScore(gs.Captured[1], clubs1 > clubs0, gs.Surs[1])
	gs.Scores[0] += earned[0]
	gs.Scores[1] += earned[1]
	return earned
}

// roundOver reports whether the current round is finished (deck and all hands
// exhausted).
func (gs *GameState) roundOver() bool {
	return len(gs.Deck) == 0 && len(gs.Hands[0]) == 0 && len(gs.Hands[1]) == 0
}

// gameOver reports whether any player has reached winScore.
func (gs *GameState) gameOver() bool {
	return gs.Scores[0] >= winScore || gs.Scores[1] >= winScore
}

// Broker is the package-level room registry. It is the only acceptable
// package-level mutable state in the codebase: it must outlive individual
// sessions, and there is exactly one arcade server process.
var Broker = newBroker()

type broker struct {
	mu   sync.Mutex
	rooms map[string]*Room
	rng  *rand.Rand
}

func newBroker() *broker {
	return &broker{
		rooms: make(map[string]*Room),
		rng:   rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// Create creates a new room, returns it with a fresh code.
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

// Join joins an existing room as player 1. Returns the room and true, or nil
// and false if the code is unknown or the room is already full/over.
func (b *broker) Join(code string) (*Room, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()

	r, ok := b.rooms[code]
	if !ok {
		return nil, false
	}

	switch r.Phase {
	case phaseWaiting:
		// Fresh join.
		r.channels[1] = make(chan stateMsg, 1)
		r.Phase = phasePlaying
		r.Game = newGameState(b.rng)
		b.broadcast(r)
		return r, true

	case phaseGrace:
		// Reconnect during grace period.
		if r.now().Before(r.GraceEnds) && r.GraceSeat == player1 {
			r.channels[1] = make(chan stateMsg, 1)
			r.Phase = phasePlaying
			b.broadcast(r)
			return r, true
		}
		return nil, false
	}

	return nil, false
}

// Rejoin allows player 0 (the host) to reconnect during a grace period.
func (b *broker) Rejoin(code string) (*Room, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()

	r, ok := b.rooms[code]
	if !ok {
		return nil, false
	}
	if r.Phase == phaseGrace && r.now().Before(r.GraceEnds) && r.GraceSeat == player0 {
		r.channels[0] = make(chan stateMsg, 1)
		r.Phase = phasePlaying
		b.broadcast(r)
		return r, true
	}
	return nil, false
}

// Disconnect handles a player disconnecting. Starts grace period or ends room.
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

	// Schedule cleanup after grace period.
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

// Play executes a card play under the broker lock and broadcasts.
func (b *broker) Play(code string, seat playerIndex, handIdx int, captureIdxs []int) bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	r, ok := b.rooms[code]
	if !ok || r.Phase != phasePlaying || r.Game.Turn != seat {
		return false
	}

	r.Game.playCard(handIdx, captureIdxs, b.rng)
	r.SelectedCard = nil
	r.CaptureOpts = nil

	if r.Game.roundOver() {
		r.Game.scoreRound()
		if r.Game.gameOver() {
			r.Phase = phaseOver
		} else {
			// Start new round.
			r.Game = newGameState(b.rng)
		}
	}

	b.broadcast(r)
	return true
}

// Select sets the selected card highlight for seat.
func (b *broker) Select(code string, seat playerIndex, handIdx *int) {
	b.mu.Lock()
	defer b.mu.Unlock()

	r, ok := b.rooms[code]
	if !ok || r.Phase != phasePlaying || r.Game.Turn != seat {
		return
	}
	r.SelectedCard = handIdx
	if handIdx != nil {
		card := r.Game.Hands[seat][*handIdx]
		r.CaptureOpts = captures(card, r.Game.Pool)
	} else {
		r.CaptureOpts = nil
	}
	b.broadcast(r)
}

// broadcast sends the current room snapshot to both players' channels.
// Must be called with b.mu held.
func (b *broker) broadcast(r *Room) {
	msg := stateMsg{room: r}
	for _, ch := range r.channels {
		if ch == nil {
			continue
		}
		select {
		case ch <- msg:
		default:
			// Channel full - drain and resend so the player always gets latest.
			select {
			case <-ch:
			default:
			}
			ch <- msg
		}
	}
}

// cleanup removes a room from the map. Must be called with b.mu held.
func (b *broker) cleanup(code string) {
	delete(b.rooms, code)
}

// uniqueCode generates a random 4-character uppercase code not already in use.
// Must be called with b.mu held.
func (b *broker) uniqueCode() string {
	const chars = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789" // no O/0/I/1 to avoid confusion
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
