package client

import (
	"net"

	tea "charm.land/bubbletea/v2"

	"github.com/chalvinwz/hacktris/internal/protocol"
)

// connectedMsg reports the outcome of dialling and joining the server.
type connectedMsg struct {
	stream *protocol.Stream
	err    error
}

// netMsg carries one inbound server message (or a read error).
type netMsg struct {
	msg protocol.Message
	err error
}

// connectCmd dials the server and sends the join handshake.
func connectCmd(addr, name string) tea.Cmd {
	return func() tea.Msg {
		conn, err := net.Dial("tcp", addr)
		if err != nil {
			return connectedMsg{err: err}
		}
		st := protocol.NewStream(conn)
		if err := st.Send(protocol.Message{Type: protocol.TypeJoin, Name: name}); err != nil {
			return connectedMsg{err: err}
		}
		return connectedMsg{stream: st}
	}
}

// readNetCmd blocks on the next server message, returning it as a tea.Msg. The
// Update loop re-arms it after each message to keep reading.
func readNetCmd(st *protocol.Stream) tea.Cmd {
	return func() tea.Msg {
		m, err := st.Recv()
		return netMsg{msg: m, err: err}
	}
}
