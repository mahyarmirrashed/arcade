// Package _2048 implements the 2048 sliding-tile game for the arcade.
package _2048

import "math/rand"

const size = 4

// grid is the 4×4 board. grid[row][col]; zero = empty.
type grid [size][size]int

// direction represents a slide direction.
type direction int

const (
	dirUp    direction = iota
	dirDown
	dirLeft
	dirRight
)

// rotationsFor maps each direction to the number of 90° CW rotations needed
// so that the intended slide becomes a slide-left operation.
//
//	left  → 0  (no rotation needed)
//	up    → 3  (CCW once: col 0 top→bottom becomes row 0 left→right)
//	right → 2  (180°: row reverses)
//	down  → 1  (CW once: col 0 bottom→top becomes row 0 left→right)
var rotationsFor = [4]int{
	dirUp:    3,
	dirDown:  1,
	dirLeft:  0,
	dirRight: 2,
}

// slide slides and merges all tiles in the given direction, returning the new
// grid, the points earned, and whether any tile moved.
func slide(g grid, d direction) (grid, int, bool) {
	n := rotationsFor[d]
	rotated := rotateN(g, n)
	merged, points := slideLeft(rotated)
	result := rotateN(merged, (4-n)%4)
	moved := result != g
	return result, points, moved
}

// slideLeft slides all rows to the left, merging equal adjacent tiles.
func slideLeft(g grid) (grid, int) {
	points := 0
	for r := 0; r < size; r++ {
		row := compact(g[r])
		row, p := merge(row)
		row = compact(row)
		points += p
		g[r] = row
	}
	return g, points
}

// compact removes zeros, packing non-zero tiles to the left.
func compact(row [size]int) [size]int {
	var out [size]int
	i := 0
	for _, v := range row {
		if v != 0 {
			out[i] = v
			i++
		}
	}
	return out
}

// merge merges adjacent equal tiles left-to-right in a compacted row.
// Each tile may only participate in one merge per move.
func merge(row [size]int) ([size]int, int) {
	points := 0
	for i := 0; i < size-1; i++ {
		if row[i] != 0 && row[i] == row[i+1] {
			row[i] *= 2
			points += row[i]
			for j := i + 1; j < size-1; j++ {
				row[j] = row[j+1]
			}
			row[size-1] = 0
		}
	}
	return row, points
}

// rotateN rotates the grid 90° clockwise n times.
func rotateN(g grid, n int) grid {
	for i := 0; i < n; i++ {
		g = rotate90(g)
	}
	return g
}

// rotate90 rotates the grid 90° clockwise once.
func rotate90(g grid) grid {
	var out grid
	for r := 0; r < size; r++ {
		for c := 0; c < size; c++ {
			out[c][size-1-r] = g[r][c]
		}
	}
	return out
}

// spawn places a new tile (90% = 2, 10% = 4) on a random empty cell.
// Returns the new grid and false if no empty cells exist.
func spawn(g grid, rng *rand.Rand) (grid, bool) {
	var empty [][2]int
	for r := 0; r < size; r++ {
		for c := 0; c < size; c++ {
			if g[r][c] == 0 {
				empty = append(empty, [2]int{r, c})
			}
		}
	}
	if len(empty) == 0 {
		return g, false
	}
	pos := empty[rng.Intn(len(empty))]
	value := 2
	if rng.Intn(10) == 0 {
		value = 4
	}
	g[pos[0]][pos[1]] = value
	return g, true
}

// hasWon reports whether any tile has reached 2048.
func hasWon(g grid) bool {
	for r := 0; r < size; r++ {
		for c := 0; c < size; c++ {
			if g[r][c] >= 2048 {
				return true
			}
		}
	}
	return false
}

// hasMovesLeft reports whether any move is still possible.
func hasMovesLeft(g grid) bool {
	for r := 0; r < size; r++ {
		for c := 0; c < size; c++ {
			if g[r][c] == 0 {
				return true
			}
			if c+1 < size && g[r][c] == g[r][c+1] {
				return true
			}
			if r+1 < size && g[r][c] == g[r+1][c] {
				return true
			}
		}
	}
	return false
}
