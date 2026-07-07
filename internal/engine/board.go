// Package engine implements the deterministic, I/O-free classic NES-style
// Tetris core: the board, pieces, seeded randomiser, gravity/scoring tables and
// the per-frame tick. Clients drive it with input methods and render from its
// exported state.
package engine

// Board dimensions follow the classic 10x20 well.
const (
	Width  = 10
	Height = 20
)

// Board is the playfield grid, indexed [row][col] with row 0 at the top.
type Board struct {
	cells [Height][Width]Cell
}

// At returns the cell at (x,y) for rendering.
func (b *Board) At(x, y int) Cell { return b.cells[y][x] }

// occupied reports whether (x,y) blocks a piece. Out-of-bounds to the sides and
// below the floor count as occupied; above the ceiling (y<0) is free so a piece
// may straddle the top at spawn.
func (b *Board) occupied(x, y int) bool {
	if x < 0 || x >= Width || y >= Height {
		return true
	}
	if y < 0 {
		return false
	}
	return b.cells[y][x] != Empty
}

// collides reports whether any of the piece's cells overlaps a blocked space.
func (b *Board) collides(p piece) bool {
	for _, c := range p.cells() {
		if b.occupied(c[0], c[1]) {
			return true
		}
	}
	return false
}

// lock writes the piece's cells into the grid. Cells above the ceiling are
// dropped; the caller detects that separately as a top-out.
func (b *Board) lock(p piece) {
	for _, c := range p.cells() {
		x, y := c[0], c[1]
		if y >= 0 && y < Height && x >= 0 && x < Width {
			b.cells[y][x] = p.kind.cell()
		}
	}
}

// clearLines removes every full row, collapses the stack downward and returns
// the number of rows cleared.
func (b *Board) clearLines() int {
	dst := Height - 1
	cleared := 0
	for src := Height - 1; src >= 0; src-- {
		if b.rowFull(src) {
			cleared++
			continue
		}
		if dst != src {
			b.cells[dst] = b.cells[src]
		}
		dst--
	}
	for dst >= 0 {
		b.cells[dst] = [Width]Cell{}
		dst--
	}
	return cleared
}

// addGarbage raises the stack by len(holes) rows and inserts that many garbage
// lines at the bottom, each full except for its hole column. It reports whether
// any existing block was pushed off the top of the well (a top-out).
func (b *Board) addGarbage(holes []int) bool {
	n := len(holes)
	if n == 0 {
		return false
	}
	overflow := false
	for y := 0; y < n && y < Height; y++ {
		if !b.rowEmpty(y) {
			overflow = true
		}
	}
	// Shift every row up by n.
	for y := 0; y < Height; y++ {
		if src := y + n; src < Height {
			b.cells[y] = b.cells[src]
		} else {
			b.cells[y] = [Width]Cell{}
		}
	}
	// Fill the bottom n rows with garbage.
	for i := 0; i < n; i++ {
		row := Height - n + i
		if row < 0 {
			continue
		}
		var r [Width]Cell
		for x := 0; x < Width; x++ {
			r[x] = Garbage
		}
		if hole := holes[i]; hole >= 0 && hole < Width {
			r[hole] = Empty
		}
		b.cells[row] = r
	}
	return overflow
}

func (b *Board) rowFull(y int) bool {
	for x := 0; x < Width; x++ {
		if b.cells[y][x] == Empty {
			return false
		}
	}
	return true
}

func (b *Board) rowEmpty(y int) bool {
	for x := 0; x < Width; x++ {
		if b.cells[y][x] != Empty {
			return false
		}
	}
	return true
}
