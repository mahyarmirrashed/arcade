package pasur

import "math/rand"

// Suit represents one of the four card suits.
type Suit int

const (
	Hearts Suit = iota
	Diamonds
	Clubs
	Spades
)

var suitSymbol = [4]string{
	Hearts:   "♥",
	Diamonds: "♦",
	Clubs:    "♣",
	Spades:   "♠",
}

func (s Suit) isRed() bool { return s == Hearts || s == Diamonds }

// Rank represents a card rank 1 (Ace) through 13 (King).
type Rank int

const (
	Ace   Rank = 1
	Jack  Rank = 11
	Queen Rank = 12
	King  Rank = 13
)

func (r Rank) String() string {
	switch r {
	case Ace:
		return "A"
	case Jack:
		return "J"
	case Queen:
		return "Q"
	case King:
		return "K"
	case 10:
		return "10"
	default:
		return string(rune('0' + int(r)))
	}
}

// pipValue returns the numerical value used for sum-to-11 captures.
// Face cards (J/Q/K) have no pip value for capture purposes.
func (r Rank) pipValue() int {
	if r >= Jack {
		return 0
	}
	return int(r)
}

// Card is a single playing card.
type Card struct {
	Suit Suit
	Rank Rank
}

func (c Card) String() string {
	return c.Rank.String() + suitSymbol[c.Suit]
}

// scoreValue returns the leaderboard point value of this card.
func (c Card) scoreValue() int {
	switch {
	case c.Rank == Jack:
		return 1
	case c.Rank == Ace:
		return 1
	case c == (Card{Diamonds, 10}):
		return 3
	case c == (Card{Clubs, 2}):
		return 2
	default:
		return 0
	}
}

// newDeck returns a shuffled standard 52-card deck.
func newDeck(rng *rand.Rand) []Card {
	deck := make([]Card, 0, 52)
	for s := Hearts; s <= Spades; s++ {
		for r := Ace; r <= King; r++ {
			deck = append(deck, Card{s, r})
		}
	}
	rng.Shuffle(len(deck), func(i, j int) {
		deck[i], deck[j] = deck[j], deck[i]
	})
	return deck
}
