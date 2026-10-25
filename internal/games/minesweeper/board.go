package minesweeper

import "math/rand"

const (
	defaultCols  = 16
	defaultRows  = 16
	defaultMines = 40
)

// cellState describes what the player can see for a cell.
type cellState int

const (
	cellHidden  cellState = iota // not yet revealed
	cellRevealed                 // revealed; shows neighbour count
	cellFlagged                  // player has placed a flag
)

// cell is a single tile on the board.
type cell struct {
	mine      bool
	state     cellState
	neighbours int // count of adjacent mines (0-8); valid only when !mine
}

// board holds the full game state.
type board struct {
	cells  [][]cell
	cols   int
	rows   int
	mines  int
	seeded bool // false until the first reveal (mines placed then)

	revealed int // count of non-mine cells revealed
	flagged  int // count of flagged cells
}

// newBoard returns an unseeded board of the given dimensions.
// Mines are placed on the first reveal so the opening cell is always safe.
func newBoard(cols, rows, mines int) board {
	cells := make([][]cell, rows)
	for r := range cells {
		cells[r] = make([]cell, cols)
	}
	return board{cells: cells, cols: cols, rows: rows, mines: mines}
}

// seed places mines randomly, ensuring (safeR, safeC) and its neighbours are
// mine-free, then computes neighbour counts.
func (b *board) seed(safeR, safeC int, rng *rand.Rand) {
	safe := make(map[[2]int]bool)
	for dr := -1; dr <= 1; dr++ {
		for dc := -1; dc <= 1; dc++ {
			r, c := safeR+dr, safeC+dc
			if r >= 0 && r < b.rows && c >= 0 && c < b.cols {
				safe[[2]int{r, c}] = true
			}
		}
	}

	placed := 0
	for placed < b.mines {
		r := rng.Intn(b.rows)
		c := rng.Intn(b.cols)
		if !b.cells[r][c].mine && !safe[[2]int{r, c}] {
			b.cells[r][c].mine = true
			placed++
		}
	}

	// Compute neighbour counts.
	for r := 0; r < b.rows; r++ {
		for c := 0; c < b.cols; c++ {
			if b.cells[r][c].mine {
				continue
			}
			count := 0
			eachNeighbour(b.cols, b.rows, r, c, func(nr, nc int) {
				if b.cells[nr][nc].mine {
					count++
				}
			})
			b.cells[r][c].neighbours = count
		}
	}
	b.seeded = true
}

// reveal reveals the cell at (r, c). If it's a mine, returns true (game over).
// If it has zero neighbours, flood-fills outward.
func (b *board) reveal(r, c int, rng *rand.Rand) (hitMine bool) {
	if !b.seeded {
		b.seed(r, c, rng)
	}

	cl := &b.cells[r][c]
	if cl.state == cellRevealed || cl.state == cellFlagged {
		return false
	}
	if cl.mine {
		cl.state = cellRevealed
		return true
	}

	b.floodReveal(r, c)
	return false
}

// floodReveal recursively reveals empty cells outward from (r, c).
func (b *board) floodReveal(r, c int) {
	cl := &b.cells[r][c]
	if cl.state == cellRevealed || cl.state == cellFlagged || cl.mine {
		return
	}
	cl.state = cellRevealed
	b.revealed++

	if cl.neighbours == 0 {
		eachNeighbour(b.cols, b.rows, r, c, func(nr, nc int) {
			b.floodReveal(nr, nc)
		})
	}
}

// toggleFlag toggles a flag on an unrevealed cell.
func (b *board) toggleFlag(r, c int) {
	cl := &b.cells[r][c]
	switch cl.state {
	case cellHidden:
		cl.state = cellFlagged
		b.flagged++
	case cellFlagged:
		cl.state = cellHidden
		b.flagged--
	}
}

// hasWon reports whether all non-mine cells have been revealed.
func (b *board) hasWon() bool {
	return b.revealed == b.cols*b.rows-b.mines
}

// revealAll reveals all cells (called on game over to show the board).
func (b *board) revealAll() {
	for r := 0; r < b.rows; r++ {
		for c := 0; c < b.cols; c++ {
			b.cells[r][c].state = cellRevealed
		}
	}
}

// eachNeighbour calls f for every valid neighbour of (r, c).
func eachNeighbour(cols, rows, r, c int, f func(nr, nc int)) {
	for dr := -1; dr <= 1; dr++ {
		for dc := -1; dc <= 1; dc++ {
			if dr == 0 && dc == 0 {
				continue
			}
			nr, nc := r+dr, c+dc
			if nr >= 0 && nr < rows && nc >= 0 && nc < cols {
				f(nr, nc)
			}
		}
	}
}
