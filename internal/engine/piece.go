package engine

// PieceType identifies one of the seven tetrominoes.
type PieceType int

// The seven tetrominoes, followed by pieceCount as a range sentinel.
const (
	PieceI PieceType = iota
	PieceO
	PieceT
	PieceS
	PieceZ
	PieceJ
	PieceL
	pieceCount
)

// Cell is a single board cell value. Zero is empty; other values encode the
// colour to draw. Piece types are stored as PieceType+1, garbage as Garbage.
type Cell uint8

// Reserved cell values; piece cells are stored as PieceType+1.
const (
	Empty   Cell = 0
	Garbage Cell = 8
)

// cell returns the stored Cell value for a piece type.
func (p PieceType) cell() Cell { return Cell(p) + 1 }

// CellValue returns the Cell a locked piece of this type writes, so the UI can
// colour the falling piece to match.
func (p PieceType) CellValue() Cell { return p.cell() }

// offset is a (col,row) position relative to a piece's 4x4 bounding box.
type offset struct{ dx, dy int }

// shapes holds the four rotation states for every piece. Each state lists the
// four occupied minos as offsets inside a 4x4 box whose top-left corner sits at
// the piece's (x,y) origin. Rotation is pure — no wall kicks — matching the
// classic NES feel: a blocked rotation simply fails.
var shapes = [pieceCount][4][4]offset{
	PieceI: {
		{{0, 1}, {1, 1}, {2, 1}, {3, 1}},
		{{2, 0}, {2, 1}, {2, 2}, {2, 3}},
		{{0, 2}, {1, 2}, {2, 2}, {3, 2}},
		{{1, 0}, {1, 1}, {1, 2}, {1, 3}},
	},
	PieceO: {
		{{1, 0}, {2, 0}, {1, 1}, {2, 1}},
		{{1, 0}, {2, 0}, {1, 1}, {2, 1}},
		{{1, 0}, {2, 0}, {1, 1}, {2, 1}},
		{{1, 0}, {2, 0}, {1, 1}, {2, 1}},
	},
	PieceT: {
		{{1, 0}, {0, 1}, {1, 1}, {2, 1}},
		{{1, 0}, {1, 1}, {2, 1}, {1, 2}},
		{{0, 1}, {1, 1}, {2, 1}, {1, 2}},
		{{1, 0}, {0, 1}, {1, 1}, {1, 2}},
	},
	PieceS: {
		{{1, 0}, {2, 0}, {0, 1}, {1, 1}},
		{{1, 0}, {1, 1}, {2, 1}, {2, 2}},
		{{1, 1}, {2, 1}, {0, 2}, {1, 2}},
		{{0, 0}, {0, 1}, {1, 1}, {1, 2}},
	},
	PieceZ: {
		{{0, 0}, {1, 0}, {1, 1}, {2, 1}},
		{{2, 0}, {1, 1}, {2, 1}, {1, 2}},
		{{0, 1}, {1, 1}, {1, 2}, {2, 2}},
		{{1, 0}, {0, 1}, {1, 1}, {0, 2}},
	},
	PieceJ: {
		{{0, 0}, {0, 1}, {1, 1}, {2, 1}},
		{{1, 0}, {2, 0}, {1, 1}, {1, 2}},
		{{0, 1}, {1, 1}, {2, 1}, {2, 2}},
		{{1, 0}, {1, 1}, {0, 2}, {1, 2}},
	},
	PieceL: {
		{{2, 0}, {0, 1}, {1, 1}, {2, 1}},
		{{1, 0}, {1, 1}, {1, 2}, {2, 2}},
		{{0, 1}, {1, 1}, {2, 1}, {0, 2}},
		{{0, 0}, {1, 0}, {1, 1}, {1, 2}},
	},
}

// PreviewCells returns a piece's minos in its spawn orientation as (x,y)
// offsets within a 4x4 box, for drawing the next-piece preview.
func PreviewCells(p PieceType) [4][2]int {
	var out [4][2]int
	for i, o := range shapes[p][0] {
		out[i] = [2]int{o.dx, o.dy}
	}
	return out
}

// piece is the falling tetromino: its type, rotation index (0-3) and the board
// position of its bounding box's top-left corner.
type piece struct {
	kind PieceType
	rot  int
	x, y int
}

// cells returns the four absolute board coordinates the piece occupies.
func (p piece) cells() [4][2]int {
	var out [4][2]int
	for i, o := range shapes[p.kind][p.rot] {
		out[i] = [2]int{p.x + o.dx, p.y + o.dy}
	}
	return out
}
