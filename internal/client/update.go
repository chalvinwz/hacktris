package client

import (
	tea "charm.land/bubbletea/v2"

	"github.com/chalvinwz/hacktris/internal/engine"
	"github.com/chalvinwz/hacktris/internal/protocol"
)

// Update routes window sizing, the frame ticker, networking events and keys.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil

	case tickMsg:
		return m.onTick()

	case connectedMsg:
		if msg.err != nil {
			m.phase = phaseError
			m.errMsg = msg.err.Error()
			return m, nil
		}
		m.net = msg.stream
		m.phase = phaseLobby
		return m, readNetCmd(m.net)

	case netMsg:
		if msg.err != nil {
			if m.phase != phaseOver {
				m.phase = phaseError
				m.errMsg = "disconnected: " + msg.err.Error()
			}
			return m, nil
		}
		extra := m.handleNet(msg.msg)
		return m, tea.Batch(readNetCmd(m.net), extra)

	case tea.KeyboardEnhancementsMsg:
		m.keyEvents = msg.SupportsEventTypes()
		return m, nil

	case tea.KeyReleaseMsg:
		return m.handleRelease(msg)

	case tea.KeyPressMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

// onTick advances the local simulation one frame and emits any resulting
// network traffic (garbage, top-out, throttled board snapshots).
func (m Model) onTick() (tea.Model, tea.Cmd) {
	if m.phase != phasePlaying || m.game == nil {
		return m, tickCmd()
	}
	m.frame++
	(&m).autoShift()
	ev := m.game.Tick()
	(&m).sendEvents(ev)
	if !m.solo() && m.frame%boardEveryFrames == 0 {
		_ = m.net.Send(m.boardMessage())
	}
	return m, tickCmd()
}

// autoShift applies delayed auto-shift: once a horizontal direction has been
// held past dasDelayFrames, the piece steps every arrFrames. It runs off the
// held-key state so it is independent of rotation, and only when the terminal
// reports key events. When both arrows are held, the most recent wins.
func (m *Model) autoShift() {
	if !m.keyEvents || m.game.GameOver {
		return
	}
	dir := 0
	switch {
	case m.heldLeft && m.heldRight:
		dir = m.lastHoriz
	case m.heldLeft:
		dir = -1
	case m.heldRight:
		dir = 1
	}
	if dir != m.dasDir { // direction changed (or released): recharge
		m.dasDir = dir
		m.dasCharge = 0
		return
	}
	if dir == 0 {
		return
	}
	m.dasCharge++
	if m.dasCharge >= dasDelayFrames && (m.dasCharge-dasDelayFrames)%arrFrames == 0 {
		if dir < 0 {
			m.game.MoveLeft()
		} else {
			m.game.MoveRight()
		}
	}
}

// handleRelease clears held-direction state (and disengages soft drop) when a
// key comes up. Only fires on terminals that report key releases.
func (m Model) handleRelease(msg tea.KeyReleaseMsg) (tea.Model, tea.Cmd) {
	if m.game == nil {
		return m, nil
	}
	switch msg.String() {
	case "left", "h":
		m.heldLeft = false
	case "right", "l":
		m.heldRight = false
	case "down", "j":
		m.game.SetSoftDrop(false)
	}
	return m, nil
}

// sendEvents forwards the network-visible consequences of a lock — garbage to
// route to an opponent, and this player topping out. Shared by gravity ticks
// (onTick) and hard drops (handleKey).
func (m *Model) sendEvents(ev engine.TickEvent) {
	if m.solo() {
		return
	}
	if len(ev.GarbageOut) > 0 {
		_ = m.net.Send(protocol.Message{Type: protocol.TypeGarbage, Holes: ev.GarbageOut})
	}
	if ev.ToppedOut && !m.eliminated {
		m.eliminated = true
		_ = m.net.Send(protocol.Message{Type: protocol.TypeEliminated})
	}
}

// handleNet applies one server message and returns any follow-up command.
func (m *Model) handleNet(msg protocol.Message) tea.Cmd {
	switch msg.Type {
	case protocol.TypeStart:
		m.seed = msg.Seed
		m.game = engine.NewGame(msg.Seed, 0)
		m.myID = msg.You
		m.players = msg.Players
		m.opponents = make(map[int]*oppView, len(msg.Players))
		for _, p := range msg.Players {
			if p.ID != m.myID {
				m.opponents[p.ID] = &oppView{name: p.Name, alive: true}
			}
		}
		m.phase = phasePlaying
		return tickCmd()

	case protocol.TypeBoard:
		if o := m.opponents[msg.From]; o != nil {
			o.cells = msg.Cells
			o.score, o.lines, o.level = msg.Score, msg.Lines, msg.Level
		}

	case protocol.TypeGarbage:
		if m.game != nil {
			m.game.AddGarbage(msg.Holes)
		}

	case protocol.TypePlayerOut:
		if o := m.opponents[msg.From]; o != nil {
			o.alive = false
		}

	case protocol.TypeResult:
		m.phase = phaseOver
		m.winnerID = msg.WinnerID
	}
	return nil
}

// boardMessage builds a snapshot of the local field (locked cells plus the
// active piece) for opponents to display.
func (m Model) boardMessage() protocol.Message {
	grid := m.composeGrid()
	cells := make([]byte, engine.Width*engine.Height)
	for y := 0; y < engine.Height; y++ {
		for x := 0; x < engine.Width; x++ {
			cells[y*engine.Width+x] = byte(grid[y][x])
		}
	}
	return protocol.Message{
		Type:  protocol.TypeBoard,
		Cells: cells,
		Score: m.game.Score,
		Lines: m.game.Lines,
		Level: m.game.Level,
	}
}

func (m Model) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "r":
		if m.solo() {
			m.seed = m.seed*6364136223846793005 + 1442695040888963407
			m.game = engine.NewGame(m.seed, 0)
		}
		return m, nil
	}

	if m.phase != phasePlaying || m.game == nil || m.game.GameOver {
		return m, nil
	}

	// With key events, held keys arrive as IsRepeat presses. Horizontal repeats
	// are driven by autoShift (DAS), and single-shot actions ignore repeats;
	// only soft drop welcomes them (as a fallback when hold isn't tracked).
	repeat := m.keyEvents && msg.Key().IsRepeat

	switch msg.String() {
	case "left", "h":
		if m.keyEvents {
			m.heldLeft, m.lastHoriz = true, -1
		}
		if !repeat {
			m.game.MoveLeft()
		}
	case "right", "l":
		if m.keyEvents {
			m.heldRight, m.lastHoriz = true, 1
		}
		if !repeat {
			m.game.MoveRight()
		}
	case "up", "x", "k":
		if !repeat {
			m.game.RotateCW()
		}
	case "z", "ctrl+z":
		if !repeat {
			m.game.RotateCCW()
		}
	case "down", "j":
		if m.keyEvents {
			m.game.SetSoftDrop(true) // held soft drop; released in handleRelease
		} else {
			m.game.SoftStep()
		}
	case " ", "space":
		if repeat {
			break
		}
		ev := m.game.HardDrop()
		(&m).sendEvents(ev)
		if !m.solo() {
			_ = m.net.Send(m.boardMessage()) // board jumps on a slam; push it now
		}
	case "c":
		if repeat {
			break
		}
		m.game.Hold()
		if !m.solo() {
			if m.game.GameOver && !m.eliminated {
				m.eliminated = true
				_ = m.net.Send(protocol.Message{Type: protocol.TypeEliminated})
			}
			_ = m.net.Send(m.boardMessage())
		}
	}
	return m, nil
}
