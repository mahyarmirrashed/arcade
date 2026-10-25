package snake

import (
	"math/rand/v2"
	"slices"
	"time"
)

// Speed: each step waits interval(score), which shrinks by speedup per food
// eaten until it reaches minInterval.
const (
	baseInterval = 150 * time.Millisecond
	speedup      = 5 * time.Millisecond
	minInterval  = 60 * time.Millisecond
)

// interval is how long the snake waits between steps at the given score.
func interval(score int) time.Duration {
	return max(baseInterval-time.Duration(score)*speedup, minInterval)
}

// point is a cell on the board. x grows rightwards and y grows downwards.
type point struct{ x, y int }

func (p point) add(q point) point { return point{p.x + q.x, p.y + q.y} }

// direction is the way the snake is heading.
type direction int

const (
	up direction = iota
	down
	left
	right
)

func (d direction) String() string {
	switch d {
	case up:
		return "up"
	case down:
		return "down"
	case left:
		return "left"
	default:
		return "right"
	}
}

// delta is the one-cell offset of a move in direction d.
func (d direction) delta() point {
	switch d {
	case up:
		return point{0, -1}
	case down:
		return point{0, 1}
	case left:
		return point{-1, 0}
	default:
		return point{1, 0}
	}
}

func (d direction) opposite() direction {
	switch d {
	case up:
		return down
	case down:
		return up
	case left:
		return right
	default:
		return left
	}
}

// board is the pure game state. It has no Bubbletea dependencies so the rules
// can be tested directly.
type board struct {
	cols, rows int
	body       []point   // head first
	dir        direction // direction of the last move actually made
	next       direction // direction the next step will take
	food       point
	score      int
	alive      bool
	won        bool
}

// newBoard returns a board with a three-cell snake in the middle, heading right.
func newBoard(cols, rows int, rng *rand.Rand) board {
	head := point{cols / 2, rows / 2}
	b := board{
		cols:  cols,
		rows:  rows,
		body:  []point{head, head.add(point{-1, 0}), head.add(point{-2, 0})},
		dir:   right,
		next:  right,
		alive: true,
	}
	b.placeFood(rng)
	return b
}

// turn sets the direction of the next step. Reversing into the snake's own
// neck is ignored. The check is against the last move made, not the last key
// pressed, so two quick presses within one tick cannot turn the snake around.
func (b *board) turn(d direction) {
	if d == b.dir.opposite() {
		return
	}
	b.next = d
}

// step advances the snake one cell. Eating food grows the snake by keeping
// its tail; hitting a wall or the body ends the game.
func (b *board) step(rng *rand.Rand) {
	if !b.alive || b.won {
		return
	}
	b.dir = b.next
	head := b.body[0].add(b.dir.delta())

	if head.x < 0 || head.x >= b.cols || head.y < 0 || head.y >= b.rows {
		b.alive = false
		return
	}

	// Unless the snake is growing, the tail moves out of the way this step,
	// so its cell is free to enter.
	grow := head == b.food
	occupied := b.body
	if !grow {
		occupied = b.body[:len(b.body)-1]
	}
	if slices.Contains(occupied, head) {
		b.alive = false
		return
	}

	b.body = append([]point{head}, occupied...)
	if grow {
		b.score++
		b.placeFood(rng)
	}
}

// placeFood puts food on a random empty cell. With no empty cell left the
// board is full and the player has won.
func (b *board) placeFood(rng *rand.Rand) {
	var empty []point
	for y := range b.rows {
		for x := range b.cols {
			if p := (point{x, y}); !slices.Contains(b.body, p) {
				empty = append(empty, p)
			}
		}
	}
	if len(empty) == 0 {
		b.won = true
		return
	}
	b.food = empty[rng.IntN(len(empty))]
}
