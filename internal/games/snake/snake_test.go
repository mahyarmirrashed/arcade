package snake

import (
	"math/rand/v2"
	"slices"
	"testing"
	"time"
)

// testRand returns a deterministic random source so tests are repeatable.
func testRand() *rand.Rand {
	return rand.New(rand.NewPCG(1, 2))
}

// heading builds a living board whose snake is already moving in dir.
func heading(cols, rows int, dir direction, body ...point) board {
	return board{
		cols:  cols,
		rows:  rows,
		body:  body,
		dir:   dir,
		next:  dir,
		food:  point{-1, -1}, // off the board, so no test eats by accident
		alive: true,
	}
}

func TestNewBoard_startsCentredHeadingRight(t *testing.T) {
	b := newBoard(10, 8, testRand())

	want := []point{{5, 4}, {4, 4}, {3, 4}}
	if !slices.Equal(b.body, want) {
		t.Errorf("body = %v; want %v", b.body, want)
	}
	if b.dir != right || b.next != right {
		t.Errorf("dir, next = %v, %v; want right, right", b.dir, b.next)
	}
	if !b.alive || b.won || b.score != 0 {
		t.Errorf("alive, won, score = %v, %v, %d; want true, false, 0", b.alive, b.won, b.score)
	}
}

func TestStep_movesHeadOneCell(t *testing.T) {
	tests := []struct {
		name string
		dir  direction
		want point
	}{
		{"up", up, point{5, 4}},
		{"down", down, point{5, 6}},
		{"left", left, point{4, 5}},
		{"right", right, point{6, 5}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := heading(10, 10, tt.dir, point{5, 5})
			b.step(testRand())

			if got := b.body[0]; got != tt.want {
				t.Errorf("head = %v; want %v", got, tt.want)
			}
			if len(b.body) != 1 {
				t.Errorf("len(body) = %d; want 1", len(b.body))
			}
		})
	}
}

func TestStep_bodyFollowsHead(t *testing.T) {
	b := heading(10, 10, right, point{5, 5}, point{4, 5}, point{3, 5})
	b.step(testRand())

	want := []point{{6, 5}, {5, 5}, {4, 5}}
	if !slices.Equal(b.body, want) {
		t.Errorf("body = %v; want %v", b.body, want)
	}
}

func TestStep_collision(t *testing.T) {
	tests := []struct {
		name      string
		board     board
		wantAlive bool
	}{
		{"open space", heading(10, 10, right, point{5, 5}), true},
		{"top wall", heading(10, 10, up, point{5, 0}), false},
		{"bottom wall", heading(10, 10, down, point{5, 9}), false},
		{"left wall", heading(10, 10, left, point{0, 5}), false},
		{"right wall", heading(10, 10, right, point{9, 5}), false},
		{
			// Head at (5,5) moving up into (5,4), which the body still occupies.
			"own body",
			heading(10, 10, up,
				point{5, 5}, point{6, 5}, point{6, 4}, point{5, 4}, point{4, 4}),
			false,
		},
		{
			// A 2x2 loop: the head moves into the cell the tail is leaving.
			"cell the tail is leaving",
			heading(10, 10, up,
				point{5, 5}, point{6, 5}, point{6, 4}, point{5, 4}),
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := tt.board
			b.step(testRand())

			if b.alive != tt.wantAlive {
				t.Errorf("alive = %v; want %v", b.alive, tt.wantAlive)
			}
		})
	}
}

func TestStep_deadSnakeDoesNotMove(t *testing.T) {
	b := heading(10, 10, right, point{5, 5})
	b.alive = false
	b.step(testRand())

	if got := b.body[0]; got != (point{5, 5}) {
		t.Errorf("head = %v; want unchanged {5 5}", got)
	}
}

func TestTurn_ignoresReversal(t *testing.T) {
	tests := []struct {
		dir     direction
		reverse direction
	}{
		{up, down},
		{down, up},
		{left, right},
		{right, left},
	}
	for _, tt := range tests {
		t.Run(tt.dir.String(), func(t *testing.T) {
			b := heading(10, 10, tt.dir, point{5, 5}, point{5, 5}.add(tt.reverse.delta()))
			b.turn(tt.reverse)

			if b.next != tt.dir {
				t.Errorf("next = %v; want %v (reversal ignored)", b.next, tt.dir)
			}
		})
	}
}

func TestTurn_acceptsPerpendicular(t *testing.T) {
	b := heading(10, 10, right, point{5, 5}, point{4, 5})
	b.turn(up)
	b.step(testRand())

	if got := b.body[0]; got != (point{5, 4}) {
		t.Errorf("head = %v; want {5 4}", got)
	}
}

func TestTurn_twoQuickPressesCannotReverse(t *testing.T) {
	// Moving right, the player presses up then left before the next tick.
	// Judged against the queued direction, left would be allowed and the
	// snake would turn back into its own neck.
	b := heading(10, 10, right, point{5, 5}, point{4, 5})
	b.turn(up)
	b.turn(left)

	if b.next != up {
		t.Errorf("next = %v; want up", b.next)
	}
	b.step(testRand())
	if !b.alive {
		t.Error("snake died; the reversal should have been ignored")
	}
}

func TestNewBoard_placesFoodOffTheSnake(t *testing.T) {
	for seed := range uint64(50) {
		b := newBoard(10, 8, rand.New(rand.NewPCG(seed, seed)))

		if slices.Contains(b.body, b.food) {
			t.Fatalf("seed %d: food %v placed on snake %v", seed, b.food, b.body)
		}
		if b.food.x < 0 || b.food.x >= b.cols || b.food.y < 0 || b.food.y >= b.rows {
			t.Fatalf("seed %d: food %v outside %dx%d board", seed, b.food, b.cols, b.rows)
		}
	}
}

func TestStep_eatingGrowsAndScores(t *testing.T) {
	b := heading(10, 10, right, point{5, 5}, point{4, 5}, point{3, 5})
	b.food = point{6, 5}
	b.step(testRand())

	want := []point{{6, 5}, {5, 5}, {4, 5}, {3, 5}}
	if !slices.Equal(b.body, want) {
		t.Errorf("body = %v; want %v (tail kept)", b.body, want)
	}
	if b.score != 1 {
		t.Errorf("score = %d; want 1", b.score)
	}
	if b.food == (point{6, 5}) || slices.Contains(b.body, b.food) {
		t.Errorf("food = %v; want a new cell off the snake", b.food)
	}
}

func TestPlaceFood_onlyEmptyCellChosen(t *testing.T) {
	// A 2x2 board with three cells taken leaves exactly one place for food.
	b := heading(2, 2, right, point{0, 0}, point{1, 0}, point{1, 1})
	b.placeFood(testRand())

	if b.food != (point{0, 1}) {
		t.Errorf("food = %v; want {0 1}", b.food)
	}
}

func TestStep_fillingTheBoardWins(t *testing.T) {
	// A 2x2 board: eating the last free cell leaves nowhere for food.
	b := heading(2, 2, left, point{1, 1}, point{1, 0}, point{0, 0})
	b.food = point{0, 1}
	b.step(testRand())

	if !b.won {
		t.Error("won = false; want true")
	}
	if !b.alive {
		t.Error("alive = false; want true")
	}
	if len(b.body) != 4 {
		t.Errorf("len(body) = %d; want 4", len(b.body))
	}

	b.step(testRand())
	if got := b.body[0]; got != (point{0, 1}) {
		t.Errorf("head = %v after winning; want it to stay at {0 1}", got)
	}
}

func TestInterval_speedsUpThenLevelsOff(t *testing.T) {
	tests := []struct {
		score int
		want  time.Duration
	}{
		{0, 150 * time.Millisecond},
		{1, 145 * time.Millisecond},
		{10, 100 * time.Millisecond},
		{18, 60 * time.Millisecond},
		{19, 60 * time.Millisecond},
		{1000, 60 * time.Millisecond},
	}
	for _, tt := range tests {
		if got := interval(tt.score); got != tt.want {
			t.Errorf("interval(%d) = %v; want %v", tt.score, got, tt.want)
		}
	}
}
