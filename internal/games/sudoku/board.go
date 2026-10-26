package sudoku

import "math/rand"

const size = 9

// grid is a 9x9 Sudoku board. 0 = empty.
type grid [size][size]int

// isValidPlacement reports whether placing num at (row, col) is valid.
// It does not check whether the cell is already filled.
func isValidPlacement(g grid, row, col, num int) bool {
	// Check row.
	for c := 0; c < size; c++ {
		if g[row][c] == num {
			return false
		}
	}
	// Check column.
	for r := 0; r < size; r++ {
		if g[r][col] == num {
			return false
		}
	}
	// Check 3x3 box.
	boxR := (row / 3) * 3
	boxC := (col / 3) * 3
	for r := boxR; r < boxR+3; r++ {
		for c := boxC; c < boxC+3; c++ {
			if g[r][c] == num {
				return false
			}
		}
	}
	return true
}

// isSolved reports whether the grid is completely and correctly filled.
func isSolved(g grid) bool {
	for r := 0; r < size; r++ {
		for c := 0; c < size; c++ {
			if g[r][c] == 0 {
				return false
			}
			if !isValidPlacement(zeroAt(g, r, c), r, c, g[r][c]) {
				return false
			}
		}
	}
	return true
}

// zeroAt returns a copy of g with cell (r, c) set to 0.
// Used to check validity of an existing placement.
func zeroAt(g grid, r, c int) grid {
	g[r][c] = 0
	return g
}

// solve fills g with a valid Sudoku solution using backtracking.
// Returns true if successful.
func solve(g *grid) bool {
	for r := 0; r < size; r++ {
		for c := 0; c < size; c++ {
			if g[r][c] != 0 {
				continue
			}
			for num := 1; num <= 9; num++ {
				if isValidPlacement(*g, r, c, num) {
					g[r][c] = num
					if solve(g) {
						return true
					}
					g[r][c] = 0
				}
			}
			return false
		}
	}
	return true
}

// generate returns a solved grid with digits shuffled using rng.
func generate(rng *rand.Rand) grid {
	var g grid

	// Fill diagonal 3x3 boxes first (they are independent of each other).
	for box := 0; box < 3; box++ {
		digits := rng.Perm(9) // 0-8, we add 1 to get 1-9
		idx := 0
		for r := box * 3; r < box*3+3; r++ {
			for c := box * 3; c < box*3+3; c++ {
				g[r][c] = digits[idx] + 1
				idx++
			}
		}
	}

	// Solve the rest via backtracking.
	solve(&g)
	return g
}

// newPuzzle generates a puzzle by removing clues from a solved grid.
// cluesTarget controls difficulty: ~35 clues = medium, ~45 = easy, ~55 = very easy.
func newPuzzle(rng *rand.Rand, cluesTarget int) (puzzle grid, solution grid) {
	solution = generate(rng)
	puzzle = solution

	// Build a shuffled list of all 81 cells and remove them one by one.
	cells := rng.Perm(81)
	removed := 0
	for _, idx := range cells {
		if 81-removed <= cluesTarget {
			break
		}
		r, c := idx/9, idx%9
		backup := puzzle[r][c]
		puzzle[r][c] = 0

		// Verify the puzzle still has a unique solution.
		// We use a simple uniqueness check: count solutions up to 2.
		if countSolutions(puzzle, 2) != 1 {
			puzzle[r][c] = backup // revert if ambiguous
		} else {
			removed++
		}
	}

	return puzzle, solution
}

// countSolutions counts the number of solutions up to limit using backtracking.
func countSolutions(g grid, limit int) int {
	count := 0
	countHelper(&g, limit, &count)
	return count
}

func countHelper(g *grid, limit int, count *int) {
	if *count >= limit {
		return
	}
	for r := 0; r < size; r++ {
		for c := 0; c < size; c++ {
			if g[r][c] != 0 {
				continue
			}
			for num := 1; num <= 9; num++ {
				if isValidPlacement(*g, r, c, num) {
					g[r][c] = num
					countHelper(g, limit, count)
					g[r][c] = 0
				}
			}
			return
		}
	}
	*count++
}

// sameBox reports whether (r1,c1) and (r2,c2) are in the same 3x3 box.
func sameBox(r1, c1, r2, c2 int) bool {
	return (r1/3) == (r2/3) && (c1/3) == (c2/3)
}
