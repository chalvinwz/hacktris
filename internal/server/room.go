package server

import (
	"log"
	"math/rand"

	"github.com/chalvinwz/hacktris/internal/protocol"
)

// room runs one match. All shared state is owned by the single run goroutine;
// per-player read pumps feed it through inbox, so no locks are needed here.
type room struct {
	players []*player
	seed    int64
	rng     *rand.Rand // seeded from seed; picks garbage targets
	log     *log.Logger

	alive map[int]bool
	inbox chan inbound
	done  chan struct{}
}

// inbound is one message (or read error) from a player.
type inbound struct {
	from *player
	msg  protocol.Message
	err  error
}

func newRoom(players []*player, seed int64, logger *log.Logger) *room {
	r := &room{
		players: players,
		seed:    seed,
		rng:     rand.New(rand.NewSource(seed)),
		log:     logger,
		alive:   make(map[int]bool, len(players)),
		inbox:   make(chan inbound, len(players)*4),
		done:    make(chan struct{}),
	}
	for _, p := range players {
		r.alive[p.id] = true
	}
	return r
}

// run announces the start, pumps every player and processes messages until one
// player remains.
func (r *room) run() {
	roster := make([]protocol.Player, len(r.players))
	for i, p := range r.players {
		roster[i] = protocol.Player{ID: p.id, Name: p.name}
	}
	for _, p := range r.players {
		// Each client learns its own ID via You so it can render self vs.
		// opponents and recognise itself as the winner.
		_ = p.stream.Send(protocol.Message{
			Type:    protocol.TypeStart,
			Seed:    r.seed,
			Players: roster,
			You:     p.id,
		})
	}

	for _, p := range r.players {
		go r.readPump(p)
	}

	for in := range r.inbox {
		if r.handle(in) {
			close(r.done)
			return
		}
	}
}

// readPump forwards a player's messages to the room until the connection drops
// or the match ends.
func (r *room) readPump(p *player) {
	for {
		m, err := p.stream.Recv()
		select {
		case r.inbox <- inbound{from: p, msg: m, err: err}:
		case <-r.done:
			return
		}
		if err != nil {
			return
		}
	}
}

// handle processes one inbound message and reports whether the match is over.
func (r *room) handle(in inbound) bool {
	if in.err != nil {
		return r.eliminate(in.from)
	}
	switch in.msg.Type {
	case protocol.TypeBoard:
		in.msg.From = in.from.id
		r.broadcastExcept(in.from.id, in.msg)
	case protocol.TypeGarbage:
		r.routeGarbage(in.from, in.msg.Holes)
	case protocol.TypeEliminated:
		return r.eliminate(in.from)
	}
	return false
}

// routeGarbage forwards garbage to one random living opponent.
func (r *room) routeGarbage(from *player, holes []int) {
	if len(holes) == 0 {
		return
	}
	var targets []*player
	for _, p := range r.players {
		if p.id != from.id && r.alive[p.id] {
			targets = append(targets, p)
		}
	}
	if len(targets) == 0 {
		return
	}
	t := targets[r.rng.Intn(len(targets))]
	_ = t.stream.Send(protocol.Message{Type: protocol.TypeGarbage, From: from.id, Holes: holes})
}

// eliminate marks a player out, tells everyone and checks for a winner.
func (r *room) eliminate(p *player) bool {
	if !r.alive[p.id] {
		return false
	}
	r.alive[p.id] = false
	r.log.Printf("player %d eliminated", p.id)
	r.broadcast(protocol.Message{Type: protocol.TypePlayerOut, From: p.id})
	return r.checkWinner()
}

// checkWinner announces the result when at most one player remains.
func (r *room) checkWinner() bool {
	var alive []int
	for id, ok := range r.alive {
		if ok {
			alive = append(alive, id)
		}
	}
	if len(alive) > 1 {
		return false
	}
	winner := 0
	if len(alive) == 1 {
		winner = alive[0]
	}
	r.broadcast(protocol.Message{Type: protocol.TypeResult, WinnerID: winner})
	r.log.Printf("match over; winner %d", winner)
	return true
}

func (r *room) broadcast(m protocol.Message) {
	for _, p := range r.players {
		_ = p.stream.Send(m)
	}
}

func (r *room) broadcastExcept(id int, m protocol.Message) {
	for _, p := range r.players {
		if p.id != id {
			_ = p.stream.Send(m)
		}
	}
}
