package server

import (
	"io"
	"log"
	"net"
	"testing"
	"time"

	"github.com/chalvinwz/hacktris/internal/protocol"
)

// quietLogger discards server log output during tests.
func quietLogger() *log.Logger { return log.New(io.Discard, "", 0) }

// dialJoin connects a client, sends Join and returns its stream + conn.
func dialJoin(t *testing.T, addr, name string) (*protocol.Stream, net.Conn) {
	t.Helper()
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	st := protocol.NewStream(conn)
	if err := st.Send(protocol.Message{Type: protocol.TypeJoin, Name: name}); err != nil {
		t.Fatalf("send join: %v", err)
	}
	return st, conn
}

// expect reads from the stream until a message of the wanted type arrives or the
// deadline passes.
func expect(t *testing.T, st *protocol.Stream, conn net.Conn, want protocol.Type) protocol.Message {
	t.Helper()
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	for {
		m, err := st.Recv()
		if err != nil {
			t.Fatalf("waiting for %q: %v", want, err)
		}
		if m.Type == want {
			return m
		}
	}
}

// A two-player match should start with a shared seed, relay garbage to the lone
// opponent, and crown the survivor when the other player tops out.
func TestTwoPlayerMatch(t *testing.T) {
	srv := New(2, quietLogger())
	addr, err := srv.Listen("127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer srv.Close()
	go srv.Serve()

	s1, c1 := dialJoin(t, addr.String(), "alice")
	defer c1.Close()
	s2, c2 := dialJoin(t, addr.String(), "bob")
	defer c2.Close()

	start1 := expect(t, s1, c1, protocol.TypeStart)
	start2 := expect(t, s2, c2, protocol.TypeStart)
	if start1.Seed != start2.Seed {
		t.Fatalf("seeds differ: %d vs %d", start1.Seed, start2.Seed)
	}
	if len(start1.Players) != 2 {
		t.Fatalf("roster size = %d, want 2", len(start1.Players))
	}
	// Joins race in the server, so identify players by the You field, not by
	// stream order.
	aliceID, bobID := start1.You, start2.You

	// Alice clears lines: garbage must land on Bob (the only opponent).
	if err := s1.Send(protocol.Message{Type: protocol.TypeGarbage, Holes: []int{3, 5}}); err != nil {
		t.Fatalf("send garbage: %v", err)
	}
	g := expect(t, s2, c2, protocol.TypeGarbage)
	if len(g.Holes) != 2 || g.Holes[0] != 3 || g.Holes[1] != 5 {
		t.Fatalf("garbage holes = %v, want [3 5]", g.Holes)
	}
	if g.From != aliceID {
		t.Fatalf("garbage From = %d, want %d", g.From, aliceID)
	}

	// Alice tops out: Bob should be declared the winner.
	if err := s1.Send(protocol.Message{Type: protocol.TypeEliminated}); err != nil {
		t.Fatalf("send eliminated: %v", err)
	}
	out := expect(t, s2, c2, protocol.TypePlayerOut)
	if out.From != aliceID {
		t.Fatalf("PlayerOut From = %d, want %d", out.From, aliceID)
	}
	res := expect(t, s2, c2, protocol.TypeResult)
	if res.WinnerID != bobID {
		t.Fatalf("winner = %d, want %d", res.WinnerID, bobID)
	}
}

// With three players, eliminating one should not end the match, but eliminating
// a second should crown the last survivor.
func TestThreePlayerLastStanding(t *testing.T) {
	srv := New(3, quietLogger())
	addr, err := srv.Listen("127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer srv.Close()
	go srv.Serve()

	s1, c1 := dialJoin(t, addr.String(), "a")
	defer c1.Close()
	s2, c2 := dialJoin(t, addr.String(), "b")
	defer c2.Close()
	s3, c3 := dialJoin(t, addr.String(), "c")
	defer c3.Close()

	expect(t, s1, c1, protocol.TypeStart)
	expect(t, s2, c2, protocol.TypeStart)
	start3 := expect(t, s3, c3, protocol.TypeStart)
	survivorID := start3.You // s3 is the player we never eliminate

	// One player out: match continues (no result yet, just a PlayerOut).
	s1.Send(protocol.Message{Type: protocol.TypeEliminated})
	expect(t, s3, c3, protocol.TypePlayerOut)

	// Second player out: the survivor on s3 should win.
	s2.Send(protocol.Message{Type: protocol.TypeEliminated})
	res := expect(t, s3, c3, protocol.TypeResult)
	if res.WinnerID != survivorID {
		t.Fatalf("winner = %d, want %d", res.WinnerID, survivorID)
	}
}
