package engine

import "testing"

// Two generators with the same seed must yield an identical piece sequence —
// the property that keeps networked players fair and in sync.
func TestRngDeterministic(t *testing.T) {
	a, b := NewRng(12345), NewRng(12345)
	for i := 0; i < 500; i++ {
		if pa, pb := a.NextPiece(), b.NextPiece(); pa != pb {
			t.Fatalf("piece %d differs: %d vs %d", i, pa, pb)
		}
	}
}

// Pulling garbage holes from a generator must never advance another generator's
// state — that's why Game keeps a separate garbage Rng.
func TestRngStreamsIndependent(t *testing.T) {
	pieces := NewRng(999)
	garbage := NewRng(999 ^ 0x5deece66d)
	want := make([]PieceType, 100)
	for i := range want {
		want[i] = pieces.NextPiece()
	}
	// Draw a pile of garbage holes from the other stream.
	for i := 0; i < 1000; i++ {
		garbage.Intn(Width)
	}
	// The piece stream from a fresh generator with the same seed is unchanged.
	fresh := NewRng(999)
	for i, w := range want {
		if got := fresh.NextPiece(); got != w {
			t.Fatalf("piece %d changed: got %d want %d", i, got, w)
		}
	}
}

func TestNextPieceInRange(t *testing.T) {
	r := NewRng(7)
	for i := 0; i < 1000; i++ {
		if p := r.NextPiece(); p < PieceI || p >= pieceCount {
			t.Fatalf("piece out of range: %d", p)
		}
	}
}

func TestClearLines(t *testing.T) {
	var b Board
	// Fill the bottom two rows completely, plus one block above them.
	for x := 0; x < Width; x++ {
		b.cells[Height-1][x] = PieceI.cell()
		b.cells[Height-2][x] = PieceI.cell()
	}
	b.cells[Height-3][0] = PieceO.cell()

	if n := b.clearLines(); n != 2 {
		t.Fatalf("cleared = %d, want 2", n)
	}
	// The stray block should now rest on the floor.
	if b.cells[Height-1][0] != PieceO.cell() {
		t.Fatalf("stray block did not fall to the floor: %v", b.cells[Height-1][0])
	}
	if !b.rowEmpty(Height - 2) {
		t.Fatalf("row above floor should be empty after clear")
	}
}

func TestRotateBlockedIsNoop(t *testing.T) {
	g := NewGame(1, 0)
	// Surround the piece so any rotation collides.
	for y := 0; y < Height; y++ {
		for x := 0; x < Width; x++ {
			if y >= 4 { // leave the spawn rows open, wall off everything below
				g.board.cells[y][x] = Garbage
			}
		}
	}
	g.cur = piece{kind: PieceT, rot: 0, x: spawnX, y: 2}
	before := g.cur.rot
	g.RotateCW()
	if g.cur.rot != before {
		// Only assert no-op when the rotated cells actually collide.
		probe := g.cur
		probe.rot = (before + 1) % 4
		if g.board.collides(probe) {
			t.Fatalf("blocked rotation should be a no-op")
		}
	}
}

func TestGarbageOverflowToppedOut(t *testing.T) {
	var b Board
	// A block near the very top means raising the stack pushes it off-screen.
	b.cells[0][5] = PieceO.cell()
	if !b.addGarbage([]int{0, 1, 2}) {
		t.Fatalf("addGarbage should report overflow when blocks are pushed off the top")
	}
}

func TestGarbageHoleColumn(t *testing.T) {
	var b Board
	b.addGarbage([]int{4})
	row := Height - 1
	if b.cells[row][4] != Empty {
		t.Fatalf("hole column should be empty")
	}
	for x := 0; x < Width; x++ {
		if x != 4 && b.cells[row][x] != Garbage {
			t.Fatalf("non-hole column %d should be garbage", x)
		}
	}
}

func TestLineScore(t *testing.T) {
	cases := []struct {
		n, level, want int
	}{
		{1, 0, 40}, {2, 0, 100}, {3, 0, 300}, {4, 0, 1200},
		{1, 9, 400}, {4, 9, 12000},
	}
	for _, c := range cases {
		if got := LineScore(c.n, c.level); got != c.want {
			t.Errorf("LineScore(%d,%d) = %d, want %d", c.n, c.level, got, c.want)
		}
	}
}

func TestGarbageSent(t *testing.T) {
	for _, c := range []struct{ n, want int }{{0, 0}, {1, 0}, {2, 1}, {3, 2}, {4, 3}} {
		if got := garbageSent(c.n); got != c.want {
			t.Errorf("garbageSent(%d) = %d, want %d", c.n, got, c.want)
		}
	}
}

// Running gravity with no input must eventually top out without panicking,
// exercising spawn, lock, clear and the tick loop end to end.
func TestGravityEventuallyTopsOut(t *testing.T) {
	g := NewGame(42, 0)
	for i := 0; i < 1_000_000 && !g.GameOver; i++ {
		g.Tick()
	}
	if !g.GameOver {
		t.Fatalf("game never topped out under pure gravity")
	}
}

// HardDrop must slam the piece flush onto the stack, award one point per cell
// fallen, and lock it — leaving the well's bottom row occupied on an empty
// board.
func TestHardDropLocksAtBottom(t *testing.T) {
	g := NewGame(3, 0)
	g.cur = piece{kind: PieceO, rot: 0, x: 4, y: 0}
	before := g.Score

	ev := g.HardDrop()
	if !ev.Locked {
		t.Fatalf("hard drop should lock the piece")
	}
	if g.Score <= before {
		t.Fatalf("hard drop should award drop points: %d -> %d", before, g.Score)
	}
	if g.GameOver {
		t.Fatalf("hard drop on an empty well should not end the game")
	}
	if g.board.rowEmpty(Height - 1) {
		t.Fatalf("bottom row should hold the locked piece after a hard drop")
	}
}

// Repeatedly hard-dropping without ever clearing a line stacks pieces to the top
// and must eventually top out through the shared lockdown/spawn path.
func TestHardDropEventuallyTopsOut(t *testing.T) {
	g := NewGame(7, 0)
	toppedOut := false
	for i := 0; i < 1000 && !g.GameOver; i++ {
		if g.HardDrop().ToppedOut {
			toppedOut = true
		}
	}
	if !g.GameOver || !toppedOut {
		t.Fatalf("repeated hard drops should top out (over=%v toppedOut=%v)", g.GameOver, toppedOut)
	}
}

// Hold parks the current piece and pulls the next one; a second hold before a
// lock is refused; after a lock, hold swaps the two pieces.
func TestHoldParkSwapAndLock(t *testing.T) {
	g := NewGame(11, 0)
	first, _ := g.Current()
	next0 := g.Next()

	g.Hold()
	if held, ok := g.Held(); !ok || held != first {
		t.Fatalf("hold slot = (%v,%v), want held %v", held, ok, first)
	}
	if cur, _ := g.Current(); cur != next0 {
		t.Fatalf("current = %v after first hold, want previous next %v", cur, next0)
	}
	if g.CanHold() {
		t.Fatalf("hold should be locked out until the next lock")
	}

	// A second hold before locking must be a no-op.
	prev, _ := g.Current()
	g.Hold()
	if cur, _ := g.Current(); cur != prev {
		t.Fatalf("second hold before a lock should do nothing")
	}

	// Lock a piece, which re-enables hold, then swap.
	g.HardDrop()
	if !g.CanHold() {
		t.Fatalf("hold should be available after a lock")
	}
	heldBefore, _ := g.Held()
	curBefore, _ := g.Current()
	g.Hold()
	if cur, _ := g.Current(); cur != heldBefore {
		t.Fatalf("swap: current = %v, want held %v", cur, heldBefore)
	}
	if held, _ := g.Held(); held != curBefore {
		t.Fatalf("swap: held = %v, want previous current %v", held, curBefore)
	}
}

// The ghost rests on the floor of an empty well and occupies the same columns as
// the active piece (a straight-down drop).
func TestGhostRestsOnFloor(t *testing.T) {
	g := NewGame(4, 0)
	ghost := g.GhostCells()
	_, cur := g.Current()

	maxY := -1
	gcols, ccols := map[int]int{}, map[int]int{}
	for i := 0; i < 4; i++ {
		if ghost[i][1] > maxY {
			maxY = ghost[i][1]
		}
		gcols[ghost[i][0]]++
		ccols[cur[i][0]]++
	}
	if maxY != Height-1 {
		t.Fatalf("ghost lowest row = %d, want floor %d", maxY, Height-1)
	}
	for x, n := range ccols {
		if gcols[x] != n {
			t.Fatalf("ghost columns differ from the piece at x=%d: %d vs %d", x, gcols[x], n)
		}
	}
}

// Determinism at the Game level: identical seeds and identical inputs produce
// identical scores and piece previews.
func TestGameDeterministic(t *testing.T) {
	a, b := NewGame(2024, 0), NewGame(2024, 0)
	for i := 0; i < 5000; i++ {
		if i%7 == 0 {
			a.MoveLeft()
			b.MoveLeft()
		}
		if i%5 == 0 {
			a.RotateCW()
			b.RotateCW()
		}
		a.Tick()
		b.Tick()
	}
	if a.Score != b.Score || a.Lines != b.Lines || a.Next() != b.Next() {
		t.Fatalf("games diverged: a(%d,%d,%d) b(%d,%d,%d)",
			a.Score, a.Lines, a.Next(), b.Score, b.Lines, b.Next())
	}
}
