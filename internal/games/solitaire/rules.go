package solitaire

// pileKind identifies the type of a pile.
type pileKind int

const (
	kindTableau    pileKind = iota
	kindFoundation          // one per suit, Ace up to King
	kindStock               // face-down draw pile
	kindWaste               // face-up discard from stock
)

// pile is a stack of cards.
type pile struct {
	kind  pileKind
	cards []Card
}

// top returns the top card of the pile, or nil if empty.
func (p *pile) top() *Card {
	if len(p.cards) == 0 {
		return nil
	}
	return &p.cards[len(p.cards)-1]
}

// canPlaceOnFoundation reports whether card c can be placed on foundation f.
// Foundation rule: starts with Ace, builds up in the same suit.
func canPlaceOnFoundation(f *pile, c Card) bool {
	if !c.FaceUp {
		return false
	}
	top := f.top()
	if top == nil {
		return c.Rank == Ace
	}
	return c.Suit == top.Suit && c.Rank == top.Rank+1
}

// canPlaceOnTableau reports whether card c can be placed on tableau t.
// Tableau rule: starts with King on empty, builds down in alternating colour.
func canPlaceOnTableau(t *pile, c Card) bool {
	if !c.FaceUp {
		return false
	}
	top := t.top()
	if top == nil {
		return c.Rank == King
	}
	if !top.FaceUp {
		return false
	}
	return c.Suit.isRed() != top.Suit.isRed() && c.Rank == top.Rank-1
}

// canPlaceSequence reports whether the sequence starting at cards[from] can be
// placed on tableau t. The whole sequence must be valid (alternating colour,
// descending rank) and canPlaceOnTableau must hold for cards[from].
func canPlaceSequence(t *pile, cards []Card, from int) bool {
	if from >= len(cards) {
		return false
	}
	if !canPlaceOnTableau(t, cards[from]) {
		return false
	}
	// Verify the sequence itself is valid.
	for i := from; i < len(cards)-1; i++ {
		a, b := cards[i], cards[i+1]
		if !a.FaceUp || !b.FaceUp {
			return false
		}
		if a.Suit.isRed() == b.Suit.isRed() {
			return false
		}
		if a.Rank != b.Rank+1 {
			return false
		}
	}
	return true
}

// isComplete reports whether all four foundations hold 13 cards (game won).
func isComplete(foundations [4]pile) bool {
	for i := range foundations {
		if len(foundations[i].cards) != 13 {
			return false
		}
	}
	return true
}

// autoFoundation tries to move the top card of src to any foundation.
// Returns the foundation index it moved to, or -1 if no move was possible.
func autoFoundation(src *pile, foundations *[4]pile) int {
	top := src.top()
	if top == nil || !top.FaceUp {
		return -1
	}
	for i := range foundations {
		if canPlaceOnFoundation(&foundations[i], *top) {
			foundations[i].cards = append(foundations[i].cards, *top)
			src.cards = src.cards[:len(src.cards)-1]
			return i
		}
	}
	return -1
}
