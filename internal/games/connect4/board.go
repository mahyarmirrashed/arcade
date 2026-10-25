package connect4

const (
	cols    = 7
	rows    = 6
	winRun  = 4
)

// cell is the content of a board cell.
type cell int

const (
	empty cell = iota
	p0         // player 0's piece
	p1         // player 1's piece
)

// board is the 6x7 Connect 4 grid.
// board[row][col]; row 0 is the top, row rows-1 is the bottom.
type board [rows][cols]cell

// drop places a piece for the given player in col, falling to the lowest
// empty row. Returns the row it landed in, or -1 if the column is full.
func drop(b board, col int, piece cell) (board, int) {
	for r := rows - 1; r >= 0; r-- {
		if b[r][col] == empty {
			b[r][col] = piece
			return b, r
		}
	}
	return b, -1
}

// colFull reports whether column col has no empty cells.
func colFull(b board, col int) bool {
	return b[0][col] != empty
}

// boardFull reports whether all cells are filled.
func boardFull(b board) bool {
	for c := 0; c < cols; c++ {
		if !colFull(b, c) {
			return false
		}
	}
	return true
}

// checkWin reports whether the last piece placed at (row, col) wins the game.
func checkWin(b board, row, col int) bool {
	piece := b[row][col]
	if piece == empty {
		return false
	}

	// Four directions: horizontal, vertical, diagonal /, diagonal \.
	dirs := [4][2]int{{0, 1}, {1, 0}, {1, 1}, {1, -1}}
	for _, d := range dirs {
		if runLength(b, row, col, d[0], d[1], piece) >= winRun {
			return true
		}
	}
	return false
}

// runLength counts consecutive pieces in both directions along (dr, dc).
func runLength(b board, row, col, dr, dc int, piece cell) int {
	count := 1
	for _, sign := range []int{1, -1} {
		for i := 1; i < winRun; i++ {
			r := row + sign*i*dr
			c := col + sign*i*dc
			if r < 0 || r >= rows || c < 0 || c >= cols || b[r][c] != piece {
				break
			}
			count++
		}
	}
	return count
}

// winningCells returns all cells that form the winning run, or nil if no win.
func winningCells(b board, row, col int) [][2]int {
	piece := b[row][col]
	if piece == empty {
		return nil
	}
	dirs := [4][2]int{{0, 1}, {1, 0}, {1, 1}, {1, -1}}
	for _, d := range dirs {
		cells := runCells(b, row, col, d[0], d[1], piece)
		if len(cells) >= winRun {
			return cells
		}
	}
	return nil
}

func runCells(b board, row, col, dr, dc int, piece cell) [][2]int {
	cells := [][2]int{{row, col}}
	for _, sign := range []int{1, -1} {
		for i := 1; i < winRun+1; i++ {
			r := row + sign*i*dr
			c := col + sign*i*dc
			if r < 0 || r >= rows || c < 0 || c >= cols || b[r][c] != piece {
				break
			}
			cells = append(cells, [2]int{r, c})
		}
	}
	return cells
}
