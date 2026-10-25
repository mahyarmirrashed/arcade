package solitaire

import (
	"math/rand"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/mahyarmirrashed/arcade/internal/game"
	"github.com/mahyarmirrashed/arcade/internal/ui"
)

func rng() *rand.Rand { return rand.New(rand.NewSource(42)) }

func newTestModel() *Model { return newModel(120, 40, ui.Styles{}) }

// ---- card.go tests ----

func TestRank_strings(t *testing.T) {
	tests := []struct{ rank Rank; want string }{
		{Ace, "A"}, {Two, "2"}, {Ten, "10"},
		{Jack, "J"}, {Queen, "Q"}, {King, "K"},
	}
	for _, tt := range tests {
		if got := tt.rank.String(); got != tt.want {
			t.Errorf("Rank(%d).String() = %q; want %q", tt.rank, got, tt.want)
		}
	}
}

func TestSuit_isRed(t *testing.T) {
	if !Hearts.isRed() { t.Error("Hearts should be red") }
	if !Diamonds.isRed() { t.Error("Diamonds should be red") }
	if Clubs.isRed() { t.Error("Clubs should not be red") }
	if Spades.isRed() { t.Error("Spades should not be red") }
}

func TestNewDeck_52Cards(t *testing.T) {
	deck := newDeck(rng())
	if len(deck) != 52 {
		t.Errorf("deck length = %d; want 52", len(deck))
	}
}

func TestNewDeck_allUnique(t *testing.T) {
	deck := newDeck(rng())
	seen := make(map[[2]int]bool)
	for _, c := range deck {
		key := [2]int{int(c.Suit), int(c.Rank)}
		if seen[key] {
			t.Errorf("duplicate card: %v %v", c.Suit, c.Rank)
		}
		seen[key] = true
	}
}

func TestRenderCard_faceUpContainsRankAndSuit(t *testing.T) {
	c := Card{Suit: Hearts, Rank: Ace, FaceUp: true}
	s := renderCard(c, false, ui.Styles{})
	if !strings.Contains(s, "A") {
		t.Error("rendered card should contain rank 'A'")
	}
	if !strings.Contains(s, "♥") {
		t.Error("rendered card should contain suit '♥'")
	}
}

func TestRenderCard_faceDown(t *testing.T) {
	c := Card{Suit: Hearts, Rank: Ace, FaceUp: false}
	s := renderCard(c, false, ui.Styles{})
	if strings.Contains(s, "A") {
		t.Error("face-down card should not reveal rank")
	}
	if strings.Contains(s, "♥") {
		t.Error("face-down card should not reveal suit")
	}
}

func TestRenderCard_10HasCorrectWidth(t *testing.T) {
	c := Card{Suit: Diamonds, Rank: Ten, FaceUp: true}
	s := renderCard(c, false, ui.Styles{})
	lines := strings.Split(s, "\n")
	if len(lines) != cardH {
		t.Errorf("card height = %d; want %d", len(lines), cardH)
	}
	// Top line should contain "10" and "♦"
	if !strings.Contains(lines[0], "10") {
		t.Errorf("top line %q should contain '10'", lines[0])
	}
	if !strings.Contains(lines[0], "♦") {
		t.Errorf("top line %q should contain '♦'", lines[0])
	}
}

func TestRenderCard_rankSuitMirroredAtBottom(t *testing.T) {
	c := Card{Suit: Spades, Rank: King, FaceUp: true}
	s := renderCard(c, false, ui.Styles{})
	lines := strings.Split(s, "\n")
	top := lines[0]
	bot := lines[cardH-1]
	// Top: ┌K──♠┐  bottom: └♠──K┘
	if !strings.Contains(top, "K") || !strings.Contains(top, "♠") {
		t.Errorf("top line %q missing rank/suit", top)
	}
	if !strings.Contains(bot, "K") || !strings.Contains(bot, "♠") {
		t.Errorf("bottom line %q missing rank/suit", bot)
	}
}

// ---- rules.go tests ----

func emptyPile(k pileKind) pile { return pile{kind: k} }

func TestCanPlaceOnFoundation_aceOnEmpty(t *testing.T) {
	f := emptyPile(kindFoundation)
	c := Card{Suit: Hearts, Rank: Ace, FaceUp: true}
	if !canPlaceOnFoundation(&f, c) {
		t.Error("ace should be placeable on empty foundation")
	}
}

func TestCanPlaceOnFoundation_twoOnAce(t *testing.T) {
	f := pile{kind: kindFoundation, cards: []Card{{Suit: Hearts, Rank: Ace, FaceUp: true}}}
	c := Card{Suit: Hearts, Rank: Two, FaceUp: true}
	if !canPlaceOnFoundation(&f, c) {
		t.Error("two of hearts should be placeable on ace of hearts")
	}
}

func TestCanPlaceOnFoundation_wrongSuit(t *testing.T) {
	f := pile{kind: kindFoundation, cards: []Card{{Suit: Hearts, Rank: Ace, FaceUp: true}}}
	c := Card{Suit: Spades, Rank: Two, FaceUp: true}
	if canPlaceOnFoundation(&f, c) {
		t.Error("wrong suit should not be placeable")
	}
}

func TestCanPlaceOnFoundation_notNextRank(t *testing.T) {
	f := pile{kind: kindFoundation, cards: []Card{{Suit: Hearts, Rank: Ace, FaceUp: true}}}
	c := Card{Suit: Hearts, Rank: Three, FaceUp: true}
	if canPlaceOnFoundation(&f, c) {
		t.Error("skipping a rank should not be allowed")
	}
}

func TestCanPlaceOnTableau_kingOnEmpty(t *testing.T) {
	t1 := emptyPile(kindTableau)
	c := Card{Suit: Hearts, Rank: King, FaceUp: true}
	if !canPlaceOnTableau(&t1, c) {
		t.Error("king should be placeable on empty tableau")
	}
}

func TestCanPlaceOnTableau_nonKingOnEmpty(t *testing.T) {
	t1 := emptyPile(kindTableau)
	c := Card{Suit: Hearts, Rank: Queen, FaceUp: true}
	if canPlaceOnTableau(&t1, c) {
		t.Error("non-king should not be placeable on empty tableau")
	}
}

func TestCanPlaceOnTableau_alternatingColor(t *testing.T) {
	t1 := pile{kind: kindTableau, cards: []Card{{Suit: Hearts, Rank: King, FaceUp: true}}}
	// Black queen on red king - valid
	c := Card{Suit: Spades, Rank: Queen, FaceUp: true}
	if !canPlaceOnTableau(&t1, c) {
		t.Error("black queen should be placeable on red king")
	}
	// Red queen on red king - invalid
	c2 := Card{Suit: Diamonds, Rank: Queen, FaceUp: true}
	if canPlaceOnTableau(&t1, c2) {
		t.Error("red queen should not be placeable on red king")
	}
}

func TestCanPlaceSequence_valid(t *testing.T) {
	t1 := pile{kind: kindTableau, cards: []Card{{Suit: Hearts, Rank: King, FaceUp: true}}}
	seq := []Card{
		{Suit: Spades, Rank: Queen, FaceUp: true},
		{Suit: Hearts, Rank: Jack, FaceUp: true},
	}
	if !canPlaceSequence(&t1, seq, 0) {
		t.Error("valid sequence should be placeable")
	}
}

func TestCanPlaceSequence_invalidSequence(t *testing.T) {
	t1 := emptyPile(kindTableau)
	seq := []Card{
		{Suit: Hearts, Rank: King, FaceUp: true},
		{Suit: Diamonds, Rank: Jack, FaceUp: true}, // skips a rank
	}
	if canPlaceSequence(&t1, seq, 0) {
		t.Error("invalid sequence (skipped rank) should not be placeable")
	}
}

func TestIsComplete_false(t *testing.T) {
	var foundations [4]pile
	if isComplete(foundations) {
		t.Error("empty foundations should not be complete")
	}
}

func TestIsComplete_true(t *testing.T) {
	var foundations [4]pile
	for i := range foundations {
		foundations[i].cards = make([]Card, 13)
	}
	if !isComplete(foundations) {
		t.Error("full foundations should be complete")
	}
}

func TestAutoFoundation_movesAce(t *testing.T) {
	src := pile{kind: kindTableau, cards: []Card{
		{Suit: Hearts, Rank: Ace, FaceUp: true},
	}}
	var foundations [4]pile
	idx := autoFoundation(&src, &foundations)
	if idx < 0 {
		t.Error("ace should auto-move to foundation")
	}
	if len(src.cards) != 0 {
		t.Error("source should be empty after auto-move")
	}
}

// ---- model tests ----

func TestModel_dealCorrectCardCount(t *testing.T) {
	m := newTestModel()
	total := 0
	for i := 0; i < 7; i++ {
		total += len(m.tableau[i].cards)
	}
	total += len(m.stock.cards)
	if total != 52 {
		t.Errorf("total cards = %d; want 52", total)
	}
}

func TestModel_tableauTopCardFaceUp(t *testing.T) {
	m := newTestModel()
	for i := 0; i < 7; i++ {
		top := m.tableau[i].top()
		if top == nil || !top.FaceUp {
			t.Errorf("tableau[%d] top card should be face up", i)
		}
	}
}

func TestModel_tableauHiddenCardsBelow(t *testing.T) {
	m := newTestModel()
	for i := 1; i < 7; i++ {
		p := &m.tableau[i]
		for j := 0; j < len(p.cards)-1; j++ {
			if p.cards[j].FaceUp {
				t.Errorf("tableau[%d] card %d should be face down", i, j)
			}
		}
	}
}

func TestModel_cursorMovement(t *testing.T) {
	m := newTestModel()
	m.cursor = 3
	m.handleKey(tea.KeyMsg{Type: tea.KeyLeft})
	if m.cursor != 2 {
		t.Errorf("cursor after left = %d; want 2", m.cursor)
	}
	m.handleKey(tea.KeyMsg{Type: tea.KeyRight})
	m.handleKey(tea.KeyMsg{Type: tea.KeyRight})
	if m.cursor != 4 {
		t.Errorf("cursor after right twice = %d; want 4", m.cursor)
	}
}

func TestModel_cursorBounded(t *testing.T) {
	m := newTestModel()
	m.cursor = 0
	m.handleKey(tea.KeyMsg{Type: tea.KeyLeft})
	if m.cursor != 0 {
		t.Error("cursor should not go below 0")
	}
	m.cursor = cursorMax
	m.handleKey(tea.KeyMsg{Type: tea.KeyRight})
	if m.cursor != cursorMax {
		t.Errorf("cursor should not exceed %d", cursorMax)
	}
}

func TestModel_drawFromStock(t *testing.T) {
	m := newTestModel()
	stockBefore := len(m.stock.cards)
	wasteBefore := len(m.waste.cards)
	m.draw()
	if len(m.stock.cards) != stockBefore-1 {
		t.Error("stock should decrease by 1 after draw")
	}
	if len(m.waste.cards) != wasteBefore+1 {
		t.Error("waste should increase by 1 after draw")
	}
	if !m.waste.top().FaceUp {
		t.Error("drawn card should be face up")
	}
}

func TestModel_drawRecyclesWaste(t *testing.T) {
	m := newTestModel()
	// Empty stock by drawing all cards.
	for len(m.stock.cards) > 0 {
		m.draw()
	}
	wasteBefore := len(m.waste.cards)
	m.draw() // should recycle
	if len(m.stock.cards) != wasteBefore {
		t.Errorf("recycled stock size = %d; want %d", len(m.stock.cards), wasteBefore)
	}
	if len(m.waste.cards) != 0 {
		t.Error("waste should be empty after recycle")
	}
}

func TestModel_viewRendersWithoutPanic(t *testing.T) {
	m := newTestModel()
	_ = m.View()
}

func TestModel_viewRendersWon(t *testing.T) {
	m := newTestModel()
	m.state = stateWon
	_ = m.View()
}

func TestModel_restartOnWon(t *testing.T) {
	m := newTestModel()
	m.state = stateWon
	cmd := m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	if cmd == nil {
		t.Fatal("expected restart command")
	}
	msg := cmd()
	rm, ok := msg.(restartMsg)
	if !ok {
		t.Fatalf("expected restartMsg, got %T", msg)
	}
	if rm.m.state != statePlaying {
		t.Error("restarted model should be playing")
	}
}

// stoppedClock returns a clock reading t0 that the test moves by hand.
func stoppedClock(m *Model) *time.Time {
	t0 := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	m.now = func() time.Time { return t0 }
	return &t0
}

// oneCardFromWinning leaves only the king of spades, face up on the waste,
// for the foundations to take.
func oneCardFromWinning(m *Model) {
	for s := Hearts; s <= Spades; s++ {
		m.foundations[s] = pile{kind: kindFoundation}
		for r := Ace; r <= King; r++ {
			if s == Spades && r == King {
				break
			}
			m.foundations[s].cards = append(m.foundations[s].cards, Card{Suit: s, Rank: r, FaceUp: true})
		}
	}
	for i := range m.tableau {
		m.tableau[i] = pile{kind: kindTableau}
	}
	m.stock = pile{kind: kindStock}
	m.waste = pile{kind: kindWaste, cards: []Card{{Suit: Spades, Rank: King, FaceUp: true}}}
	m.cursor = cursorWaste
}

var autoKey = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")}

func TestModel_winReportsTimeFromFirstMove(t *testing.T) {
	m := newTestModel()
	clock := stoppedClock(m)
	*clock = clock.Add(time.Hour) // time before the first move doesn't count
	m.handleKey(tea.KeyMsg{Type: tea.KeyRight})
	oneCardFromWinning(m)
	*clock = clock.Add(3 * time.Minute)

	cmd := m.handleKey(autoKey)

	want := game.ResultMsg{Game: "solitaire", Ranking: game.ByTime,
		Result: game.Result{Won: true, Duration: 3 * time.Minute}}
	if m.state != stateWon || cmd == nil || cmd() != want {
		t.Errorf("state %v, cmd() = %v; want stateWon and %#v", m.state, cmd, want)
	}
}

func TestModel_resultIsReportedOnce(t *testing.T) {
	m := newTestModel()
	stoppedClock(m)
	oneCardFromWinning(m)
	m.handleKey(autoKey)

	if cmd := m.handleKey(autoKey); cmd != nil {
		t.Errorf("a key after winning returned %v; want nil", cmd)
	}
}

func TestModel_standingAppearsAtGameOver(t *testing.T) {
	m := newTestModel()
	m.state = stateWon

	m.Update(game.StandingMsg{Standing: game.Standing{Ranked: true, NewBest: true, Rank: 1, Of: 4}})

	if v := m.View(); !strings.Contains(v, "new best! #1/4") {
		t.Errorf("win screen doesn't show the standing:\n%s", v)
	}
}

func TestModel_tooSmallTerminalShowsNoticeAndIgnoresKeys(t *testing.T) {
	m := newTestModel()
	cursor := m.cursor

	m.Update(tea.WindowSizeMsg{Width: 30, Height: 10})
	m.Update(tea.KeyMsg{Type: tea.KeyRight})

	if v := m.View(); !strings.Contains(v, "terminal too small") {
		t.Errorf("30x10 view does not say the terminal is too small:\n%s", v)
	}
	if m.cursor != cursor || !m.started.IsZero() {
		t.Error("a key acted on the game while the player couldn't see it")
	}
}
