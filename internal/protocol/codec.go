package protocol

import (
	"encoding/json"
	"io"
	"sync"
)

// Stream is a bidirectional message channel over a connection. Recv is meant to
// be called from a single reader goroutine; Send is safe for concurrent use.
type Stream struct {
	dec *json.Decoder
	enc *json.Encoder
	mu  sync.Mutex
}

// NewStream wraps a connection (or any ReadWriter) for message exchange.
func NewStream(rw io.ReadWriter) *Stream {
	return &Stream{
		dec: json.NewDecoder(rw),
		enc: json.NewEncoder(rw),
	}
}

// Send writes one message followed by a newline. It is safe to call from
// multiple goroutines.
func (s *Stream) Send(m Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.enc.Encode(m) // Encode appends '\n', framing the stream
}

// Recv reads the next message from the stream.
func (s *Stream) Recv() (Message, error) {
	var m Message
	err := s.dec.Decode(&m)
	return m, err
}
