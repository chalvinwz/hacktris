// Package client implements the hacktris terminal UI and game loop. It runs the
// deterministic engine locally and, in multiplayer, exchanges board snapshots
// and garbage with the relay server.
package client

import (
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/chalvinwz/hacktris/internal/engine"
	"github.com/chalvinwz/hacktris/internal/protocol"
)

// frameRate drives the simulation; the engine's per-level gravity counter does
// the actual speed shaping, so a fixed 60 Hz tick is all the UI needs.
const frameRate = time.Second / 60

// boardEveryFrames throttles outgoing snapshots to ~15 Hz so opponents see a
// live board without flooding the link every frame.
const boardEveryFrames = 4

// Horizontal auto-shift tuning (frames at 60 Hz): dasDelayFrames is the pause
// after the first move before the piece starts sliding; arrFrames is how often
// it steps once sliding.
const (
	dasDelayFrames = 8
	arrFrames      = 2
)

// phase is the client's high-level state.
type phase int

const (
	phaseConnecting phase = iota // dialling the server
	phaseLobby                   // connected, waiting for the match to start
	phasePlaying                 // match in progress (also used for solo)
	phaseOver                    // match decided
	phaseError                   // connection failed or dropped
)

// tickMsg fires once per simulation frame.
type tickMsg time.Time

// oppView is the last-known state of an opponent, for the mini-board grid.
type oppView struct {
	name  string
	cells []byte // Width*Height row-major; nil until the first snapshot
	score int
	lines int
	level int
	alive bool
}

// Model is the root Bubble Tea model. With no server address it runs solo; with
// one it joins a networked match.
type Model struct {
	game  *engine.Game
	seed  int64
	frame int

	width  int
	height int

	// Networking. net is nil in solo play.
	serverAddr string
	name       string
	net        *protocol.Stream
	phase      phase
	errMsg     string

	myID       int
	players    []protocol.Player
	opponents  map[int]*oppView
	eliminated bool
	winnerID   int

	// Horizontal auto-shift (DAS/ARR), driven from held-key state. Only engaged
	// when the terminal reports key release events (Kitty keyboard protocol), so
	// held keys can be tracked and movement runs independently of rotation.
	keyEvents bool // terminal reports key press/release/repeat
	heldLeft  bool
	heldRight bool
	lastHoriz int // -1 or +1: the horizontal direction pressed most recently
	dasDir    int // direction currently charging (-1/0/+1)
	dasCharge int // frames the current direction has been held
}

// New builds a solo, time-seeded game.
func New() Model {
	seed := time.Now().UnixNano()
	return Model{
		game:  engine.NewGame(seed, 0),
		seed:  seed,
		phase: phasePlaying,
	}
}

// NewMultiplayer builds a client that will dial addr and join as name.
func NewMultiplayer(addr, name string) Model {
	if name == "" {
		name = "player"
	}
	return Model{
		serverAddr: addr,
		name:       name,
		phase:      phaseConnecting,
		opponents:  make(map[int]*oppView),
	}
}

// Init dials the server (multiplayer) or starts the frame ticker (solo).
func (m Model) Init() tea.Cmd {
	if m.serverAddr == "" {
		return tickCmd()
	}
	return connectCmd(m.serverAddr, m.name)
}

func tickCmd() tea.Cmd {
	return tea.Tick(frameRate, func(t time.Time) tea.Msg { return tickMsg(t) })
}

// solo reports whether this is a local, non-networked game.
func (m Model) solo() bool { return m.serverAddr == "" }
