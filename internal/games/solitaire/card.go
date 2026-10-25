package solitaire

import (
	"math/rand"
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"

	"github.com/mahyarmirrashed/arcade/internal/ui"
)

// Suit represents one of the four card suits.
type Suit int

const (
	Hearts Suit = iota
	Diamonds
	Clubs
	Spades
)

// suitSymbol returns the Unicode symbol for the suit, as used in csol.
var suitSymbol = [4]string{
	Hearts:   "♥",
	Diamonds: "♦",
	Clubs:    "♣",
	Spades:   "♠",
}

// isRed reports whether the suit is red (Hearts or Diamonds).
func (s Suit) isRed() bool { return s == Hearts || s == Diamonds }

// Rank represents a card rank from Ace (1) to King (13).
type Rank int

const (
	Ace   Rank = 1
	Two   Rank = 2
	Three Rank = 3
	Four  Rank = 4
	Five  Rank = 5
	Six   Rank = 6
	Seven Rank = 7
	Eight Rank = 8
	Nine  Rank = 9
	Ten   Rank = 10
	Jack  Rank = 11
	Queen Rank = 12
	King  Rank = 13
)

// rankSymbol returns the display string for a rank.
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
	case Ten:
		return "10"
	default:
		return string(rune('0' + int(r)))
	}
}

// Card is a playing card.
type Card struct {
	Suit    Suit
	Rank    Rank
	FaceUp  bool
}

// color returns the lipgloss color for this card's suit.
func (c Card) color() lipgloss.Color {
	if c.Suit.isRed() {
		return ui.Spectrum[0] // red
	}
	return ui.ColorText // white/bright
}

// newDeck returns a shuffled 52-card deck.
func newDeck(rng *rand.Rand) []Card {
	deck := make([]Card, 0, 52)
	for s := Hearts; s <= Spades; s++ {
		for r := Ace; r <= King; r++ {
			deck = append(deck, Card{Suit: s, Rank: r})
		}
	}
	rng.Shuffle(len(deck), func(i, j int) {
		deck[i], deck[j] = deck[j], deck[i]
	})
	return deck
}

// Card rendering - csol corners-theme inspired:
//
//	┌A──♥┐   rank top-left, suit top-right, mirrored at bottom
//	│    │   (cardW=6, cardH=4, inner width=4)
//	│    │
//	└♥──A┘
//
// For two-char ranks (10):
//	┌10─♥┐
//	│    │
//	│    │
//	└♥─10┘
//
// Face-down:
//	┌────┐
//	│▒▒▒▒│
//	│▒▒▒▒│
//	└────┘

const (
	cardW     = 6 // terminal columns
	cardH     = 4 // terminal rows
	innerW    = 4 // characters between the two border pipes
	dash      = "─"
)

// cardBorderLine builds a border line with rank and suit embedded in the corners.
//
//	┌A──♥┐  (top, left=rank, right=suit)
//	└♥──A┘  (bottom, left=suit, right=rank)
func cardBorderLine(tl, tr, left, right string, styles ui.Styles, cardStyle lipgloss.Style, borderStyle lipgloss.Style) string {
	leftW := utf8.RuneCountInString(left)
	rightW := utf8.RuneCountInString(right)
	dashes := strings.Repeat(dash, innerW-leftW-rightW)
	return borderStyle.Render(tl) +
		cardStyle.Render(left) +
		borderStyle.Render(dashes) +
		cardStyle.Render(right) +
		borderStyle.Render(tr)
}

// bgCard is the background colour for both face-up and face-down cards.
var bgCard = lipgloss.Color("235") // dark grey

// renderCard returns a 6-wide x 4-tall string for the card.
// selected highlights the border.
func renderCard(c Card, selected bool, styles ui.Styles) string {
	borderColor := ui.ColorSubtle
	if selected {
		borderColor = ui.ColorText
	}

	if !c.FaceUp {
		borderStyle := styles.NewStyle().Foreground(borderColor).Background(bgCard)
		fill := styles.NewStyle().Foreground(ui.ColorMuted).Background(bgCard).Render("▒▒▒▒")
		midBack := borderStyle.Render("│") + fill + borderStyle.Render("│")
		return strings.Join([]string{
			borderStyle.Render("┌────┐"),
			midBack,
			midBack,
			borderStyle.Render("└────┘"),
		}, "\n")
	}

	borderStyle := styles.NewStyle().Foreground(borderColor).Background(bgCard)
	if selected {
		borderStyle = borderStyle.Bold(true)
	}

	midStyle := styles.NewStyle().Background(bgCard)
	mid := borderStyle.Render("│") + midStyle.Render(strings.Repeat(" ", innerW)) + borderStyle.Render("│")

	rank := c.Rank.String()
	suit := suitSymbol[c.Suit]
	cardStyle := styles.NewStyle().Foreground(c.color()).Background(bgCard)
	if selected {
		cardStyle = cardStyle.Bold(true)
	}

	topLine := cardBorderLine("┌", "┐", rank, suit, styles, cardStyle, borderStyle)
	botLine := cardBorderLine("└", "┘", suit, rank, styles, cardStyle, borderStyle)

	return strings.Join([]string{topLine, mid, mid, botLine}, "\n")
}

// renderEmpty returns a 6x4 empty pile placeholder.
func renderEmpty(styles ui.Styles) string {
	borderStyle := styles.NewStyle().Foreground(ui.ColorMuted)
	mid := borderStyle.Render("│") + strings.Repeat(" ", innerW) + borderStyle.Render("│")
	return strings.Join([]string{
		borderStyle.Render("┌────┐"),
		mid,
		mid,
		borderStyle.Render("└────┘"),
	}, "\n")
}

// renderBack returns a 6x4 face-down card (for the stock pile).
func renderBack(styles ui.Styles) string {
	borderStyle := styles.NewStyle().Foreground(ui.ColorSubtle).Background(bgCard)
	fill := styles.NewStyle().Foreground(ui.ColorMuted).Background(bgCard).Render("▒▒▒▒")
	midBack := borderStyle.Render("│") + fill + borderStyle.Render("│")
	return strings.Join([]string{
		borderStyle.Render("┌────┐"),
		midBack,
		midBack,
		borderStyle.Render("└────┘"),
	}, "\n")
}
