package engine

// SoftDropFrames is the gravity delay while soft dropping: the piece falls fast
// but still steps one row at a time so the player keeps control.
const SoftDropFrames = 2

// spawnX places a piece's 4x4 box so the tetromino lands centred on a 10-wide
// well.
const spawnX = 3

// TickEvent reports what happened during a single Tick so the caller (the
// network client) can react: forward garbage, refresh displays, end the match.
type TickEvent struct {
	Locked       bool  // the active piece locked this tick
	LinesCleared int   // rows cleared on that lock
	GarbageOut   []int // hole columns for garbage to send to an opponent
	ToppedOut    bool  // the player just topped out (game over)
}

// Game is a single player's deterministic Tetris simulation. It performs no
// I/O: the client drives it with input methods and Tick, and renders from the
// exported state.
type Game struct {
	board      Board
	rng        *Rng // drives the piece sequence; identical across players
	garbageRng *Rng // drives garbage hole columns; kept separate so clearing
	// lines never perturbs the shared piece sequence
	cur     piece
	next    PieceType
	startLv int

	Score int
	Lines int
	Level int

	gravity  int  // frames accumulated toward the next drop
	soft     bool // soft drop currently engaged
	GameOver bool

	held     PieceType // the piece parked in the hold slot
	hasHeld  bool      // whether the hold slot is occupied
	holdUsed bool      // hold already used this drop (reset on lock)

	incoming [][]int // queued garbage hole-sets, applied on the next lock
}

// NewGame creates a game seeded for a starting level. The seed must match across
// players for a fair, identical piece sequence.
func NewGame(seed int64, startLevel int) *Game {
	g := &Game{
		rng:        NewRng(seed),
		garbageRng: NewRng(seed ^ 0x5deece66d),
		startLv:    startLevel,
		Level:      startLevel,
	}
	g.next = g.rng.NextPiece()
	g.spawn()
	return g
}

// Board exposes the grid for rendering.
func (g *Game) Board() *Board { return &g.board }

// Current returns the falling piece's type and absolute cells, for rendering.
func (g *Game) Current() (PieceType, [4][2]int) { return g.cur.kind, g.cur.cells() }

// Next returns the upcoming piece type, for the preview box.
func (g *Game) Next() PieceType { return g.next }

// Queue returns the next n upcoming pieces (starting with Next) without
// advancing the real generator, for a multi-piece preview. It peeks on a copy
// of the RNG so the live piece sequence is untouched.
func (g *Game) Queue(n int) []PieceType {
	if n <= 0 {
		return nil
	}
	out := make([]PieceType, 0, n)
	out = append(out, g.next)
	r := *g.rng // copy state; NextPiece mutates only the copy
	for i := 1; i < n; i++ {
		out = append(out, r.NextPiece())
	}
	return out
}

// Held returns the piece in the hold slot and whether the slot is occupied.
func (g *Game) Held() (PieceType, bool) { return g.held, g.hasHeld }

// CanHold reports whether a hold is currently allowed (not yet used this drop).
func (g *Game) CanHold() bool { return !g.holdUsed && !g.GameOver }

// Hold swaps the active piece with the hold slot. The first hold parks the
// current piece and pulls the next one; later holds swap the two. Only one hold
// is allowed per drop — the lock resets it — so it can't be spammed.
func (g *Game) Hold() {
	if g.GameOver || g.holdUsed {
		return
	}
	g.holdUsed = true
	if !g.hasHeld {
		g.held = g.cur.kind
		g.hasHeld = true
		g.spawn() // bring the next piece in; may top out
		return
	}
	swap := g.held
	g.held = g.cur.kind
	g.cur = piece{kind: swap, rot: 0, x: spawnX, y: 0}
	g.gravity = 0
	if g.board.collides(g.cur) {
		g.GameOver = true
	}
}

// GhostCells returns where the active piece would rest if dropped straight down,
// for the landing-preview (ghost) overlay.
func (g *Game) GhostCells() [4][2]int {
	p := g.cur
	for {
		q := p
		q.y++
		if g.board.collides(q) {
			break
		}
		p = q
	}
	return p.cells()
}

// spawn places the next piece at the top-centre and pulls a new preview. A
// fresh piece that immediately collides means the player has topped out.
func (g *Game) spawn() bool {
	g.cur = piece{kind: g.next, rot: 0, x: spawnX, y: 0}
	g.next = g.rng.NextPiece()
	g.gravity = 0
	if g.board.collides(g.cur) {
		g.GameOver = true
		return false
	}
	return true
}

// MoveLeft shifts the piece one column left when space allows.
func (g *Game) MoveLeft() { g.shift(-1) }

// MoveRight shifts the piece one column right when space allows.
func (g *Game) MoveRight() { g.shift(1) }

func (g *Game) shift(dx int) {
	if g.GameOver {
		return
	}
	p := g.cur
	p.x += dx
	if !g.board.collides(p) {
		g.cur = p
	}
}

// RotateCW turns the piece clockwise if the target position is clear.
func (g *Game) RotateCW() { g.rotate(1) }

// RotateCCW turns the piece counter-clockwise if the target position is clear.
func (g *Game) RotateCCW() { g.rotate(-1) }

func (g *Game) rotate(dir int) {
	if g.GameOver {
		return
	}
	p := g.cur
	p.rot = (p.rot + dir + 4) % 4
	if !g.board.collides(p) { // no wall kicks: blocked rotation is ignored
		g.cur = p
	}
}

// SetSoftDrop engages or releases accelerated gravity (held soft drop, for
// terminals that report key releases).
func (g *Game) SetSoftDrop(on bool) { g.soft = on }

// SoftStep drops the piece one row immediately if it can, awarding the NES
// soft-drop point. It never locks — the regular gravity tick handles that — so
// it suits terminals that report key presses (with auto-repeat) but not
// releases.
func (g *Game) SoftStep() {
	if g.GameOver {
		return
	}
	p := g.cur
	p.y++
	if !g.board.collides(p) {
		g.cur = p
		g.Score++
		g.gravity = 0
	}
}

// HardDrop drops the piece straight to the bottom, awarding one point per cell
// fallen (the Guideline scores a hard drop like a soft drop), then locks it
// immediately. It reports what happened via the same TickEvent as a gravity
// lock, so the caller can forward garbage and detect a top-out.
func (g *Game) HardDrop() TickEvent {
	if g.GameOver {
		return TickEvent{}
	}
	cells := 0
	p := g.cur
	for {
		q := p
		q.y++
		if g.board.collides(q) {
			break
		}
		p = q
		cells++
	}
	g.cur = p
	g.Score += cells
	g.gravity = 0
	return g.lockdown()
}

// AddGarbage queues garbage rows (one hole column each) to apply the next time
// the active piece locks.
func (g *Game) AddGarbage(holes []int) {
	if len(holes) == 0 {
		return
	}
	g.incoming = append(g.incoming, append([]int(nil), holes...))
}

// Tick advances the simulation by one frame and reports what happened.
func (g *Game) Tick() TickEvent {
	if g.GameOver {
		return TickEvent{}
	}
	g.gravity++
	threshold := FramesPerRow(g.Level)
	if g.soft && SoftDropFrames < threshold {
		threshold = SoftDropFrames
	}
	if g.gravity < threshold {
		return TickEvent{}
	}
	g.gravity = 0

	// Try to fall one row.
	p := g.cur
	p.y++
	if !g.board.collides(p) {
		g.cur = p
		if g.soft {
			g.Score++ // NES soft-drop bonus
		}
		return TickEvent{}
	}

	// Cannot fall: lock, clear, take garbage, spawn.
	return g.lockdown()
}

func (g *Game) lockdown() TickEvent {
	ev := TickEvent{Locked: true}
	g.board.lock(g.cur)

	if cleared := g.board.clearLines(); cleared > 0 {
		g.Score += LineScore(cleared, g.Level)
		g.Lines += cleared
		g.Level = g.startLv + LevelFor(g.Lines)
		ev.LinesCleared = cleared
		for i := 0; i < garbageSent(cleared); i++ {
			ev.GarbageOut = append(ev.GarbageOut, g.garbageRng.Intn(Width))
		}
	}

	// Apply queued incoming garbage now that the piece has settled.
	for _, holes := range g.incoming {
		if g.board.addGarbage(holes) {
			g.GameOver = true
			ev.ToppedOut = true
		}
	}
	g.incoming = g.incoming[:0]
	if g.GameOver {
		return ev
	}

	if !g.spawn() {
		ev.ToppedOut = true
	}
	g.holdUsed = false // a fresh piece is in play; hold is available again
	return ev
}
