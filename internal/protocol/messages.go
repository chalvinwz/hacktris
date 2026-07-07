// Package protocol defines the wire messages exchanged between hacktris clients
// and the relay server, encoded as newline-delimited JSON over TCP.
package protocol

// Type tags a message. A single flat Message struct carries every field; only
// those relevant to a given Type are populated.
type Type string

// Message type tags: client→server (join, board, garbage, eliminated) and
// server→client (start, playerout, result).
const (
	TypeJoin       Type = "join"       // client→server: request to enter the lobby
	TypeBoard      Type = "board"      // client→server: a board snapshot to share
	TypeGarbage    Type = "garbage"    // client→server: garbage for a random opponent
	TypeEliminated Type = "eliminated" // client→server: sender topped out

	TypeStart     Type = "start"     // server→client: match begins, shared seed + roster
	TypePlayerOut Type = "playerout" // server→client: a player was eliminated
	TypeResult    Type = "result"    // server→client: match over, winner announced
)

// Player identifies a participant. IDs start at 1 so they survive JSON
// omitempty.
type Player struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// Message is the flat envelope for every exchange. Field usage by Type:
//
//	join:       Name
//	start:      Seed, Players, You
//	board:      From (set by server), Cells, Score, Lines, Level
//	garbage:    Holes  (From set by server on relay)
//	eliminated: (none; sender known by connection)
//	playerout:  From
//	result:     WinnerID
type Message struct {
	Type Type `json:"t"`

	Name string `json:"name,omitempty"`

	Seed    int64    `json:"seed,omitempty"`
	Players []Player `json:"players,omitempty"`
	You     int      `json:"you,omitempty"` // the recipient's own player ID (start)

	From  int    `json:"from,omitempty"`  // originating player (server-stamped on relay)
	Cells []byte `json:"cells,omitempty"` // Width*Height row-major snapshot
	Score int    `json:"score,omitempty"`
	Lines int    `json:"lines,omitempty"`
	Level int    `json:"level,omitempty"`

	Holes []int `json:"holes,omitempty"` // garbage hole columns, one per row

	WinnerID int `json:"winner,omitempty"`
}
