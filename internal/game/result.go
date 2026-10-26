package game

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// Ranking is how a game's results are ranked on the leaderboard.
type Ranking int

const (
	// ByScore ranks by highest score.
	ByScore Ranking = iota
	// ByStreak ranks by longest run of consecutive wins, then by fewest
	// guesses per win.
	ByStreak
	// ByTime ranks by fastest win.
	ByTime
)

// Ranked is implemented by games that appear on the leaderboard.
type Ranked interface {
	Game

	// Ranking returns how the game's results are ranked.
	Ranking() Ranking
}

// ID is the name a game's results are stored under: its name in lowercase.
func ID(g Game) string { return strings.ToLower(g.Name()) }

// Result is the outcome of a finished game.
type Result struct {
	Score    int           // points, for ByScore games
	Won      bool          // whether the player won, for ByStreak and ByTime games
	Guesses  int           // guesses a win took, for ByStreak games
	Duration time.Duration // how long a win took, for ByTime games
}

// ResultMsg is sent by a game when it finishes. The root model records it on
// the leaderboard and replies with a StandingMsg.
type ResultMsg struct {
	Game    string
	Ranking Ranking
	Result  Result
}

// Finished returns the command a ranked game sends when it ends.
func Finished(g Ranked, r Result) tea.Cmd {
	msg := ResultMsg{Game: ID(g), Ranking: g.Ranking(), Result: r}
	return func() tea.Msg { return msg }
}

// Standing is where a result placed on the leaderboard.
type Standing struct {
	Ranked  bool // false for guests and unnamed players, whose results aren't kept
	NewBest bool // the result beat the player's previous best
	Rank    int  // the player's position, from 1; 0 if they don't qualify yet
	Of      int  // how many players qualify
	Best    int  // the player's all-time best score (ByScore games only)
}

// StandingMsg tells a game where its result placed.
type StandingMsg struct{ Standing Standing }

// String describes the standing briefly enough for a game-over screen.
func (s Standing) String() string {
	switch {
	case !s.Ranked:
		return "unranked"
	case s.Rank == 0:
		return "not ranked yet"
	case s.NewBest:
		return fmt.Sprintf("new best! #%d/%d", s.Rank, s.Of)
	default:
		return fmt.Sprintf("#%d/%d", s.Rank, s.Of)
	}
}
