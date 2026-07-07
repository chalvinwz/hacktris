package client

import (
	"strings"
	"testing"

	"github.com/chalvinwz/hacktris/internal/engine"
	"github.com/chalvinwz/hacktris/internal/protocol"
)

func startMsg() protocol.Message {
	return protocol.Message{
		Type: protocol.TypeStart,
		Seed: 777,
		You:  1,
		Players: []protocol.Player{
			{ID: 1, Name: "me"},
			{ID: 2, Name: "foe"},
		},
	}
}

func TestHandleStart(t *testing.T) {
	m := NewMultiplayer("x:1", "me")
	m.handleNet(startMsg())

	if m.phase != phasePlaying {
		t.Fatalf("phase = %d, want playing", m.phase)
	}
	if m.game == nil {
		t.Fatal("game not created on start")
	}
	if m.myID != 1 {
		t.Fatalf("myID = %d, want 1", m.myID)
	}
	if len(m.opponents) != 1 {
		t.Fatalf("opponents = %d, want 1 (self excluded)", len(m.opponents))
	}
	if o := m.opponents[2]; o == nil || o.name != "foe" || !o.alive {
		t.Fatalf("opponent 2 not set up correctly: %+v", o)
	}
}

func TestHandleBoardSnapshot(t *testing.T) {
	m := NewMultiplayer("x:1", "me")
	m.handleNet(startMsg())

	cells := make([]byte, engine.Width*engine.Height)
	cells[0] = byte(engine.Garbage)
	m.handleNet(protocol.Message{Type: protocol.TypeBoard, From: 2, Cells: cells, Score: 42, Lines: 3})

	o := m.opponents[2]
	if o.score != 42 || o.lines != 3 {
		t.Fatalf("opponent stats = (%d,%d), want (42,3)", o.score, o.lines)
	}
	if len(o.cells) != engine.Width*engine.Height {
		t.Fatalf("opponent cells len = %d", len(o.cells))
	}
}

func TestBoardMessageRoundTrip(t *testing.T) {
	m := NewMultiplayer("x:1", "me")
	m.handleNet(startMsg())

	msg := m.boardMessage()
	if msg.Type != protocol.TypeBoard {
		t.Fatalf("type = %q", msg.Type)
	}
	if len(msg.Cells) != engine.Width*engine.Height {
		t.Fatalf("cells len = %d, want %d", len(msg.Cells), engine.Width*engine.Height)
	}
}

func TestIncomingGarbageReachesBoard(t *testing.T) {
	m := NewMultiplayer("x:1", "me")
	m.handleNet(startMsg())

	// Queue four garbage rows; they apply when the active piece next locks.
	m.handleNet(protocol.Message{Type: protocol.TypeGarbage, Holes: []int{0, 1, 2, 3}})

	locked := false
	for i := 0; i < 5000 && !locked; i++ {
		if m.game.Tick().Locked {
			locked = true
		}
	}
	if !locked {
		t.Fatal("piece never locked")
	}

	found := false
	b := m.game.Board()
	for y := 0; y < engine.Height && !found; y++ {
		for x := 0; x < engine.Width; x++ {
			if b.At(x, y) == engine.Garbage {
				found = true
				break
			}
		}
	}
	if !found {
		t.Fatal("incoming garbage did not appear on the board after a lock")
	}
}

func TestResultBannerWin(t *testing.T) {
	m := NewMultiplayer("x:1", "me")
	m.handleNet(startMsg())
	m.handleNet(protocol.Message{Type: protocol.TypeResult, WinnerID: 1})

	if m.phase != phaseOver {
		t.Fatalf("phase = %d, want over", m.phase)
	}
	if b := m.resultBanner(); !strings.Contains(b, "YOU WIN") {
		t.Fatalf("banner = %q, want YOU WIN", b)
	}
}

func TestResultBannerLoss(t *testing.T) {
	m := NewMultiplayer("x:1", "me")
	m.handleNet(startMsg())
	m.handleNet(protocol.Message{Type: protocol.TypeResult, WinnerID: 2})

	if b := m.resultBanner(); !strings.Contains(b, "foe wins") {
		t.Fatalf("banner = %q, want \"foe wins\"", b)
	}
}
