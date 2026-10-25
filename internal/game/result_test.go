package game_test

import (
	"testing"

	"github.com/mahyarmirrashed/arcade/internal/game"
)

// rankedGame is a minimal game.Ranked for tests.
type rankedGame struct{ stubGame }

func (rankedGame) Ranking() game.Ranking { return game.ByStreak }

func TestID_isLowercaseName(t *testing.T) {
	if got := game.ID(stubGame{name: "Snake"}); got != "snake" {
		t.Errorf("ID = %q; want %q", got, "snake")
	}
}

func TestFinished_reportsGameRankingAndResult(t *testing.T) {
	g := rankedGame{stubGame{name: "Wordle"}}
	res := game.Result{Won: true, Guesses: 4}

	msg, ok := game.Finished(g, res)().(game.ResultMsg)

	if !ok {
		t.Fatalf("Finished() produced %T; want game.ResultMsg", msg)
	}
	want := game.ResultMsg{Game: "wordle", Ranking: game.ByStreak, Result: res}
	if msg != want {
		t.Errorf("ResultMsg = %+v; want %+v", msg, want)
	}
}

func TestStanding_String(t *testing.T) {
	tests := []struct {
		name string
		s    game.Standing
		want string
	}{
		{"guest", game.Standing{}, "unranked"},
		{"not qualifying yet", game.Standing{Ranked: true, Of: 4}, "not ranked yet"},
		{"new best", game.Standing{Ranked: true, NewBest: true, Rank: 3, Of: 12}, "new best! #3/12"},
		{"same as before", game.Standing{Ranked: true, Rank: 5, Of: 12}, "#5/12"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.s.String(); got != tt.want {
				t.Errorf("String() = %q; want %q", got, tt.want)
			}
		})
	}
}

var _ game.Ranked = rankedGame{}
