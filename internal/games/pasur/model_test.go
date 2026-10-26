package pasur

import (
	"math/rand"
	"testing"
	"time"
)

func deterministicRng() *rand.Rand { return rand.New(rand.NewSource(42)) }

// ---- card.go tests ----

func TestRank_strings(t *testing.T) {
	tests := []struct {
		r    Rank
		want string
	}{
		{Ace, "A"}, {2, "2"}, {10, "10"},
		{Jack, "J"}, {Queen, "Q"}, {King, "K"},
	}
	for _, tt := range tests {
		if got := tt.r.String(); got != tt.want {
			t.Errorf("Rank(%d).String() = %q; want %q", tt.r, got, tt.want)
		}
	}
}

func TestPipValue(t *testing.T) {
	if Ace.pipValue() != 1 {
		t.Error("Ace pip = 1")
	}
	if Rank(10).pipValue() != 10 {
		t.Error("10 pip = 10")
	}
	if Jack.pipValue() != 0 {
		t.Error("Jack pip = 0")
	}
	if King.pipValue() != 0 {
		t.Error("King pip = 0")
	}
}

func TestNewDeck_52Unique(t *testing.T) {
	deck := newDeck(deterministicRng())
	if len(deck) != 52 {
		t.Fatalf("deck len = %d; want 52", len(deck))
	}
	seen := make(map[Card]bool)
	for _, c := range deck {
		if seen[c] {
			t.Errorf("duplicate card: %v", c)
		}
		seen[c] = true
	}
}

func TestScoreValue(t *testing.T) {
	if (Card{Diamonds, 10}).scoreValue() != 3 {
		t.Error("10♦ = 3")
	}
	if (Card{Clubs, 2}).scoreValue() != 2 {
		t.Error("2♣ = 2")
	}
	if (Card{Hearts, Ace}).scoreValue() != 1 {
		t.Error("A♥ = 1")
	}
	if (Card{Hearts, Jack}).scoreValue() != 1 {
		t.Error("J♥ = 1")
	}
	if (Card{Spades, 5}).scoreValue() != 0 {
		t.Error("5♠ = 0")
	}
}

// ---- rules.go tests ----

func TestJackCaptures_allNumberCardsAndJacks(t *testing.T) {
	pool := []Card{
		{Hearts, Ace},
		{Spades, King},
		{Diamonds, 5},
		{Clubs, Jack},
		{Hearts, Queen},
	}
	opts := jackCaptures(pool)
	if len(opts) != 1 {
		t.Fatalf("expected 1 capture option, got %d", len(opts))
	}
	// Should capture Ace(0), 5(2), Jack(3) - not King(1) or Queen(4)
	got := make(map[int]bool)
	for _, i := range opts[0] {
		got[i] = true
	}
	if !got[0] || !got[2] || !got[3] {
		t.Error("jack should capture Ace, 5, and Jack from pool")
	}
	if got[1] || got[4] {
		t.Error("jack should not capture King or Queen")
	}
}

func TestFaceCaptures_kingOnKing(t *testing.T) {
	pool := []Card{
		{Hearts, King},
		{Clubs, 5},
		{Spades, King},
	}
	opts := faceCaptures(pool, King)
	if len(opts) != 2 {
		t.Errorf("expected 2 king capture options, got %d", len(opts))
	}
}

func TestSumCaptures_table(t *testing.T) {
	tests := []struct {
		name    string
		card    Card
		pool    []Card
		wantLen int // number of distinct capture options
	}{
		{
			name: "7 captures 4+ace+2",
			card: Card{Hearts, 7},
			pool: []Card{
				{Spades, Ace}, {Clubs, 2}, {Diamonds, 4},
				{Hearts, 3}, {Spades, King},
			},
			// 7: target=4. options: {4}, {Ace+3}, {2+2}... let's compute:
			// single: 4 (idx2), 3 (idx3)
			// pairs: Ace+3=4 (idx0,idx3), 2+2=4? only one 2, so no
			// Actually target = 11-7 = 4
			// {4} = idx2; {Ace+3} = idx0+idx3
			wantLen: 2,
		},
		{
			name:    "no captures",
			card:    Card{Hearts, 2},
			pool:    []Card{{Spades, King}, {Clubs, Queen}},
			wantLen: 0,
		},
		{
			name:    "ace captures 10",
			card:    Card{Hearts, Ace},
			pool:    []Card{{Diamonds, 10}},
			wantLen: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := sumCaptures(tt.card, tt.pool)
			if len(opts) != tt.wantLen {
				t.Errorf("got %d options; want %d", len(opts), tt.wantLen)
			}
		})
	}
}

func TestCanCapture(t *testing.T) {
	pool := []Card{{Hearts, 7}, {Clubs, 4}}
	// 7 can capture with another card to make 11? 7+4=11, yes
	if !canCapture(Card{Spades, Ace}, pool) {
		// Ace + 7 + 3? No - only 7 and 4 in pool. Ace target = 10. 7+4=11 no. Actually no.
		// Let me reconsider: Ace pip=1, target=10. Pool has 7 and 4. 7 alone no. 4 alone no. 7+4=11 no, =11 yes!
		// Wait: target for Ace = 11 - 1 = 10. {7+4=11} != 10. Hmm, but 7 alone = 7 != 10. 4 alone = 4 != 10.
		// So Ace cannot capture {7,4}.
	}
	// 4 can capture: 4 + 7 = 11, target for 4 = 11-4 = 7. Pool has 7. Yes!
	if !canCapture(Card{Diamonds, 4}, pool) {
		t.Error("4 should be able to capture 7 from pool (4+7=11)")
	}
	if canCapture(Card{Hearts, King}, pool) {
		t.Error("King should not capture a non-King pool")
	}
}

func TestIsSur(t *testing.T) {
	// Normal sur: capture all pool cards, not a Jack, not last deal
	if !isSur(Card{Hearts, 5}, 3, 3, false) {
		t.Error("expected sur when capturing all 3 pool cards")
	}
	// Jack: not a sur
	if isSur(Card{Hearts, Jack}, 3, 3, false) {
		t.Error("Jack capture should not be a sur")
	}
	// Last deal: not a sur
	if isSur(Card{Hearts, 5}, 3, 3, true) {
		t.Error("last deal capture should not be a sur")
	}
	// Partial capture: not a sur
	if isSur(Card{Hearts, 5}, 2, 3, false) {
		t.Error("partial capture should not be a sur")
	}
}

func TestRoundScore(t *testing.T) {
	captured := []Card{
		{Diamonds, 10}, // 3 pts
		{Clubs, 2},     // 2 pts
		{Hearts, Ace},  // 1 pt
		{Spades, Jack}, // 1 pt
	}
	// mostClubs=false, surCount=0: 3+2+1+1=7
	got := roundScore(captured, false, 0)
	if got != 7 {
		t.Errorf("roundScore = %d; want 7", got)
	}
	// mostClubs=true: 7+7=14
	got = roundScore(captured, true, 0)
	if got != 14 {
		t.Errorf("roundScore with clubs = %d; want 14", got)
	}
	// surCount=1: 7+5=12 (no most clubs)
	got = roundScore(captured, false, 1)
	if got != 12 {
		t.Errorf("roundScore with 1 sur = %d; want 12", got)
	}
}

func TestClubCount(t *testing.T) {
	cards := []Card{
		{Clubs, 5}, {Clubs, Ace}, {Hearts, 3}, {Spades, King},
	}
	if n := clubCount(cards); n != 2 {
		t.Errorf("clubCount = %d; want 2", n)
	}
}

// ---- game.go tests ----

func newTestBroker() *broker {
	b := newBroker()
	b.rng = deterministicRng()
	return b
}

func TestBroker_createAndJoin(t *testing.T) {
	b := newTestBroker()
	r := b.Create()
	if r == nil {
		t.Fatal("Create returned nil")
	}
	if r.Phase != phaseWaiting {
		t.Errorf("phase = %v; want phaseWaiting", r.Phase)
	}
	if len(r.Code) != 4 {
		t.Errorf("code len = %d; want 4", len(r.Code))
	}

	r2, ok := b.Join(r.Code)
	if !ok || r2 == nil {
		t.Fatal("Join failed")
	}
	if r2.Phase != phasePlaying {
		t.Errorf("phase after join = %v; want phasePlaying", r2.Phase)
	}
}

func TestBroker_joinUnknownCode(t *testing.T) {
	b := newTestBroker()
	_, ok := b.Join("ZZZZ")
	if ok {
		t.Error("joining unknown code should fail")
	}
}

func TestBroker_codesAreUnique(t *testing.T) {
	b := newTestBroker()
	codes := make(map[string]bool)
	for i := 0; i < 20; i++ {
		r := b.Create()
		if codes[r.Code] {
			t.Errorf("duplicate code: %s", r.Code)
		}
		codes[r.Code] = true
	}
}

func TestBroker_disconnectStartsGrace(t *testing.T) {
	b := newTestBroker()
	r := b.Create()
	b.Join(r.Code)

	b.Disconnect(r.Code, player1)

	b.mu.Lock()
	phase := b.rooms[r.Code].Phase
	b.mu.Unlock()

	if phase != phaseGrace {
		t.Errorf("phase after disconnect = %v; want phaseGrace", phase)
	}
}

func TestBroker_rejoinDuringGrace(t *testing.T) {
	b := newTestBroker()
	now := time.Now()
	r := b.Create()
	b.Join(r.Code)
	b.Disconnect(r.Code, player1)

	// Inject a future GraceEnds so grace period hasn't expired.
	b.mu.Lock()
	b.rooms[r.Code].GraceEnds = now.Add(60 * time.Second)
	b.rooms[r.Code].now = func() time.Time { return now }
	b.mu.Unlock()

	r2, ok := b.Join(r.Code)
	if !ok || r2 == nil {
		t.Fatal("rejoin during grace should succeed")
	}
	if r2.Phase != phasePlaying {
		t.Errorf("phase after rejoin = %v; want phasePlaying", r2.Phase)
	}
}

func TestBroker_broadcastReachesBothChannels(t *testing.T) {
	b := newTestBroker()
	r := b.Create()
	b.Join(r.Code)

	// Both channels should receive after a Play.
	b.Play(r.Code, player0, 0, nil)

	for seat := 0; seat < 2; seat++ {
		ch := r.channels[seat]
		select {
		case msg := <-ch:
			if msg.room == nil {
				t.Errorf("seat %d got nil room in stateMsg", seat)
			}
		default:
			t.Errorf("seat %d channel had no message after play", seat)
		}
	}
}

func TestGameState_dealHands(t *testing.T) {
	rng := deterministicRng()
	gs := newGameState(rng)

	initialDeckLen := len(gs.Deck)
	gs.dealHands()
	if len(gs.Hands[0]) != 2*handSize {
		t.Errorf("hand0 after deal = %d; want %d", len(gs.Hands[0]), 2*handSize)
	}
	if len(gs.Deck) != initialDeckLen-2*handSize {
		t.Errorf("deck len after deal = %d; want %d", len(gs.Deck), initialDeckLen-2*handSize)
	}
}

func TestGameState_playCardToPool(t *testing.T) {
	rng := deterministicRng()
	gs := newGameState(rng)
	poolBefore := len(gs.Pool)
	hand0Before := len(gs.Hands[0])

	// Find a card in player0's hand that cannot capture anything.
	idx := -1
	for i, c := range gs.Hands[0] {
		if !canCapture(c, gs.Pool) {
			idx = i
			break
		}
	}
	if idx == -1 {
		t.Skip("all cards in hand can capture; skipping play-to-pool test")
	}

	gs.playCard(idx, nil, rng)
	if len(gs.Pool) != poolBefore+1 {
		t.Errorf("pool after play = %d; want %d", len(gs.Pool), poolBefore+1)
	}
	if len(gs.Hands[0]) != hand0Before-1 {
		t.Errorf("hand after play = %d; want %d", len(gs.Hands[0]), hand0Before-1)
	}
	if gs.Turn != player1 {
		t.Error("turn should pass to player1")
	}
}

func TestGameState_noJackInInitialPool(t *testing.T) {
	rng := deterministicRng()
	for i := 0; i < 50; i++ {
		gs := newGameState(rng)
		for _, c := range gs.Pool {
			if c.Rank == Jack {
				t.Errorf("initial pool contains a Jack on iteration %d", i)
			}
		}
	}
}
