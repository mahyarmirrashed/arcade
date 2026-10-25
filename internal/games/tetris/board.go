package tetris

const (
	boardRows = 20
	boardCols = 10
)

// cell represents a single cell on the board.
// zero value = empty.
type cell struct {
	filled    bool
	pieceType PieceType
}

// board is the 20×10 Tetris playfield.
// board[0] is the top row; board[19] is the bottom row.
type board [boardRows][boardCols]cell

// collides reports whether placing shape s with its top-left corner at
// (originRow, originCol) on b would overlap a filled cell or go out of bounds.
func collides(b board, s Shape, originRow, originCol int) bool {
	for _, pos := range Cells(s) {
		r := originRow + pos[0]
		c := originCol + pos[1]
		if r < 0 || r >= boardRows || c < 0 || c >= boardCols {
			return true
		}
		if b[r][c].filled {
			return true
		}
	}
	return false
}

// place stamps shape s with its top-left at (originRow, originCol) onto a
// copy of b and returns the new board. pt is stored in each filled cell.
func place(b board, s Shape, originRow, originCol int, pt PieceType) board {
	for _, pos := range Cells(s) {
		r := originRow + pos[0]
		c := originCol + pos[1]
		b[r][c] = cell{filled: true, pieceType: pt}
	}
	return b
}

// clearLines removes every fully filled row from b and returns the compacted
// board (empty rows added at the top) along with the number of lines cleared.
func clearLines(b board) (board, int) {
	cleared := 0
	var newRows [boardRows][boardCols]cell

	dest := boardRows - 1
	for src := boardRows - 1; src >= 0; src-- {
		if isFullRow(b[src]) {
			cleared++
			continue
		}
		newRows[dest] = b[src]
		dest--
	}
	// dest+1..0 remain zero-value (empty) - correct behaviour.
	return board(newRows), cleared
}

func isFullRow(row [boardCols]cell) bool {
	for _, c := range row {
		if !c.filled {
			return false
		}
	}
	return true
}

// score returns the points earned for clearing n lines at the given level,
// using the standard Tetris scoring:
//
//	1 line  → 100 × level
//	2 lines → 300 × level
//	3 lines → 500 × level
//	4 lines → 800 × level
func score(lines, level int) int {
	points := [5]int{0, 100, 300, 500, 800}
	if lines < 0 || lines > 4 {
		return 0
	}
	return points[lines] * level
}
