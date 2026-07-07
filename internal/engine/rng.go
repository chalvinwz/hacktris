package engine

// Rng is a small deterministic generator. The same seed yields the same stream
// on every platform, which keeps networked players' piece sequences in sync.
type Rng struct {
	state uint64
	last  PieceType
}

// NewRng seeds a generator. A zero seed is remapped so the stream never stalls.
func NewRng(seed int64) *Rng {
	s := uint64(seed)
	if s == 0 {
		s = 0x9e3779b97f4a7c15
	}
	return &Rng{state: s, last: pieceCount}
}

// next advances the xorshift64 state and returns the raw value.
func (r *Rng) next() uint64 {
	r.state ^= r.state << 13
	r.state ^= r.state >> 7
	r.state ^= r.state << 17
	return r.state
}

// Intn returns a value in [0,n). n<=0 yields 0.
func (r *Rng) Intn(n int) int {
	if n <= 0 {
		return 0
	}
	return int(r.next() % uint64(n))
}

// NextPiece returns the next tetromino, mimicking the NES randomiser: roll one
// of eight slots and, if it repeats the previous piece or lands on the spare
// slot, reroll once across the seven pieces. This trims — but never fully
// removes — immediate repeats.
func (r *Rng) NextPiece() PieceType {
	roll := r.Intn(int(pieceCount) + 1) // 0..7
	if roll == int(pieceCount) || PieceType(roll) == r.last {
		roll = r.Intn(int(pieceCount)) // 0..6
	}
	p := PieceType(roll)
	r.last = p
	return p
}
