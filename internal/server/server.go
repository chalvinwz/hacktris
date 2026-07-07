// Package server implements the hacktris relay: it groups players into rooms,
// hands out a shared RNG seed and forwards board snapshots and garbage between
// them. Game simulation lives entirely on the clients.
package server

import (
	"log"
	"math/rand"
	"net"
	"sync"
	"time"

	"github.com/chalvinwz/hacktris/internal/protocol"
)

// player is one connected participant.
type player struct {
	id     int
	name   string
	stream *protocol.Stream
}

// Server accepts connections and fills rooms of roomSize players.
type Server struct {
	roomSize int
	log      *log.Logger

	ln net.Listener

	mu      sync.Mutex
	rng     *rand.Rand // guarded by mu; used for seeds only
	nextID  int
	waiting []*player
}

// New creates a server that starts a match once roomSize players are waiting.
// roomSize is clamped to a minimum of 2.
func New(roomSize int, logger *log.Logger) *Server {
	if roomSize < 2 {
		roomSize = 2
	}
	if logger == nil {
		logger = log.Default()
	}
	return &Server{
		roomSize: roomSize,
		log:      logger,
		rng:      rand.New(rand.NewSource(time.Now().UnixNano())),
		nextID:   1,
	}
}

// Listen binds the server to addr and returns the resolved address.
func (s *Server) Listen(addr string) (net.Addr, error) {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	s.ln = ln
	return ln.Addr(), nil
}

// Serve runs the accept loop until the listener is closed.
func (s *Server) Serve() error {
	for {
		conn, err := s.ln.Accept()
		if err != nil {
			return err
		}
		go s.handleConn(conn)
	}
}

// Close stops accepting new connections.
func (s *Server) Close() error {
	if s.ln == nil {
		return nil
	}
	return s.ln.Close()
}

// handleConn reads the join handshake, then enqueues the player. The room takes
// over reading once the match starts.
func (s *Server) handleConn(conn net.Conn) {
	st := protocol.NewStream(conn)
	msg, err := st.Recv()
	if err != nil || msg.Type != protocol.TypeJoin {
		conn.Close()
		return
	}
	s.mu.Lock()
	p := &player{id: s.nextID, name: msg.Name, stream: st}
	s.nextID++
	s.waiting = append(s.waiting, p)
	s.log.Printf("player %d (%q) joined; %d waiting", p.id, p.name, len(s.waiting))

	if len(s.waiting) < s.roomSize {
		s.mu.Unlock()
		return
	}
	group := s.waiting[:s.roomSize]
	s.waiting = s.waiting[s.roomSize:]
	seed := s.rng.Int63()
	s.mu.Unlock()

	r := newRoom(group, seed, s.log)
	s.log.Printf("starting room: %d players, seed %d", len(group), seed)
	go r.run()
}
