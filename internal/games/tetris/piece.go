package tetris

// Shape represents one rotation of a tetromino as a 4×4 boolean grid.
// true = occupied cell. The piece is always stored in the top-left of the grid.
type Shape [4][4]bool

// PieceType identifies one of the 7 standard tetrominoes.
type PieceType int

const (
	PieceI PieceType = iota
	PieceO
	PieceT
	PieceS
	PieceZ
	PieceJ
	PieceL
)

// pieces holds all 4 rotations (0°, 90°, 180°, 270°) for each piece type.
// Rotations are stored in clockwise order.
var pieces = [7][4]Shape{
	// I
	{
		{
			{false, false, false, false},
			{true, true, true, true},
			{false, false, false, false},
			{false, false, false, false},
		},
		{
			{false, false, true, false},
			{false, false, true, false},
			{false, false, true, false},
			{false, false, true, false},
		},
		{
			{false, false, false, false},
			{false, false, false, false},
			{true, true, true, true},
			{false, false, false, false},
		},
		{
			{false, true, false, false},
			{false, true, false, false},
			{false, true, false, false},
			{false, true, false, false},
		},
	},
	// O
	{
		{
			{false, true, true, false},
			{false, true, true, false},
			{false, false, false, false},
			{false, false, false, false},
		},
		{
			{false, true, true, false},
			{false, true, true, false},
			{false, false, false, false},
			{false, false, false, false},
		},
		{
			{false, true, true, false},
			{false, true, true, false},
			{false, false, false, false},
			{false, false, false, false},
		},
		{
			{false, true, true, false},
			{false, true, true, false},
			{false, false, false, false},
			{false, false, false, false},
		},
	},
	// T
	{
		{
			{false, true, false, false},
			{true, true, true, false},
			{false, false, false, false},
			{false, false, false, false},
		},
		{
			{false, true, false, false},
			{false, true, true, false},
			{false, true, false, false},
			{false, false, false, false},
		},
		{
			{false, false, false, false},
			{true, true, true, false},
			{false, true, false, false},
			{false, false, false, false},
		},
		{
			{false, true, false, false},
			{true, true, false, false},
			{false, true, false, false},
			{false, false, false, false},
		},
	},
	// S
	{
		{
			{false, true, true, false},
			{true, true, false, false},
			{false, false, false, false},
			{false, false, false, false},
		},
		{
			{false, true, false, false},
			{false, true, true, false},
			{false, false, true, false},
			{false, false, false, false},
		},
		{
			{false, false, false, false},
			{false, true, true, false},
			{true, true, false, false},
			{false, false, false, false},
		},
		{
			{true, false, false, false},
			{true, true, false, false},
			{false, true, false, false},
			{false, false, false, false},
		},
	},
	// Z
	{
		{
			{true, true, false, false},
			{false, true, true, false},
			{false, false, false, false},
			{false, false, false, false},
		},
		{
			{false, false, true, false},
			{false, true, true, false},
			{false, true, false, false},
			{false, false, false, false},
		},
		{
			{false, false, false, false},
			{true, true, false, false},
			{false, true, true, false},
			{false, false, false, false},
		},
		{
			{false, true, false, false},
			{true, true, false, false},
			{true, false, false, false},
			{false, false, false, false},
		},
	},
	// J
	{
		{
			{true, false, false, false},
			{true, true, true, false},
			{false, false, false, false},
			{false, false, false, false},
		},
		{
			{false, true, true, false},
			{false, true, false, false},
			{false, true, false, false},
			{false, false, false, false},
		},
		{
			{false, false, false, false},
			{true, true, true, false},
			{false, false, true, false},
			{false, false, false, false},
		},
		{
			{false, true, false, false},
			{false, true, false, false},
			{true, true, false, false},
			{false, false, false, false},
		},
	},
	// L
	{
		{
			{false, false, true, false},
			{true, true, true, false},
			{false, false, false, false},
			{false, false, false, false},
		},
		{
			{false, true, false, false},
			{false, true, false, false},
			{false, true, true, false},
			{false, false, false, false},
		},
		{
			{false, false, false, false},
			{true, true, true, false},
			{true, false, false, false},
			{false, false, false, false},
		},
		{
			{true, true, false, false},
			{false, true, false, false},
			{false, true, false, false},
			{false, false, false, false},
		},
	},
}

// Rotation returns the Shape for the given piece type and rotation index (0–3).
func Rotation(pt PieceType, rot int) Shape {
	return pieces[pt][rot&3]
}

// Cells returns the list of occupied (row, col) positions within the 4×4 grid
// for the given shape. Row 0 is the top.
func Cells(s Shape) [][2]int {
	var out [][2]int
	for r := 0; r < 4; r++ {
		for c := 0; c < 4; c++ {
			if s[r][c] {
				out = append(out, [2]int{r, c})
			}
		}
	}
	return out
}

// srsOffsets holds the SRS wall-kick offset tests for J, L, S, T, Z pieces.
// Index: [fromRotation][toRotation (CW)] -> list of (col, row) offset tests.
// Using the standard SRS table (col delta, row delta).
var srsOffsets = [4][4][][2]int{
	// from 0
	{
		{}, // 0→0 unused
		{{0, 0}, {-1, 0}, {-1, 1}, {0, -2}, {-1, -2}},  // 0→1 (CW)
		{},                                               // 0→2 unused
		{{0, 0}, {1, 0}, {1, 1}, {0, -2}, {1, -2}},      // 0→3 (CCW)
	},
	// from 1
	{
		{{0, 0}, {1, 0}, {1, -1}, {0, 2}, {1, 2}},  // 1→0 (CCW)
		{},
		{{0, 0}, {1, 0}, {1, -1}, {0, 2}, {1, 2}},  // 1→2 (CW)
		{},
	},
	// from 2
	{
		{},
		{{0, 0}, {-1, 0}, {-1, 1}, {0, -2}, {-1, -2}}, // 2→1 (CCW)
		{},
		{{0, 0}, {1, 0}, {1, 1}, {0, -2}, {1, -2}},     // 2→3 (CW)
	},
	// from 3
	{
		{{0, 0}, {-1, 0}, {-1, -1}, {0, 2}, {-1, 2}},  // 3→0 (CW)
		{},
		{{0, 0}, {-1, 0}, {-1, -1}, {0, 2}, {-1, 2}},  // 3→2 (CCW)
		{},
	},
}

// srsOffsetsI holds the SRS wall-kick offset tests specifically for the I piece.
var srsOffsetsI = [4][4][][2]int{
	{
		{},
		{{0, 0}, {-2, 0}, {1, 0}, {-2, -1}, {1, 2}},   // 0→1 (CW)
		{},
		{{0, 0}, {-1, 0}, {2, 0}, {-1, 2}, {2, -1}},   // 0→3 (CCW)
	},
	{
		{{0, 0}, {2, 0}, {-1, 0}, {2, 1}, {-1, -2}},   // 1→0 (CCW)
		{},
		{{0, 0}, {-1, 0}, {2, 0}, {-1, 2}, {2, -1}},   // 1→2 (CW)
		{},
	},
	{
		{},
		{{0, 0}, {1, 0}, {-2, 0}, {1, -2}, {-2, 1}},   // 2→1 (CCW)
		{},
		{{0, 0}, {2, 0}, {-1, 0}, {2, 1}, {-1, -2}},   // 2→3 (CW)
	},
	{
		{{0, 0}, {1, 0}, {-2, 0}, {1, -2}, {-2, 1}},   // 3→0 (CW)
		{},
		{{0, 0}, {-2, 0}, {1, 0}, {-2, -1}, {1, 2}},   // 3→2 (CCW)
		{},
	},
}

// WallKickOffsets returns the SRS offset tests for rotating piece pt
// from rotation fromRot to toRot.
func WallKickOffsets(pt PieceType, fromRot, toRot int) [][2]int {
	if pt == PieceI {
		return srsOffsetsI[fromRot&3][toRot&3]
	}
	return srsOffsets[fromRot&3][toRot&3]
}
