# hacktris

Classic NES-style **Tetris in your terminal**, with online multiplayer versus
play. Drop pieces in a 10×20 well, clear lines, and rain garbage on everyone
else — last stack standing wins.

[![CI](https://github.com/chalvinwz/hacktris/actions/workflows/ci.yml/badge.svg)](https://github.com/chalvinwz/hacktris/actions/workflows/ci.yml)
[![Go 1.26](https://img.shields.io/badge/go-1.26.3-00ADD8?logo=go)](https://go.dev/dl/)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![PRs welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](CONTRIBUTING.md)

Built with the [Charm](https://charm.sh) stack — [Bubble Tea](https://github.com/charmbracelet/bubbletea)
for the TUI and [Lip Gloss](https://github.com/charmbracelet/lipgloss) for styling.

---

## Why

Tetris is the perfect terminal game: a tiny grid, a few keys, instant feedback.
hacktris keeps the **NES-style core** — fixed-pivot rotation, no wall kicks,
instant lock, the NES gravity and scoring curves — with a few modern comforts
layered on (hold, ghost piece, hard drop), and bolts on **N-player
battle-royale netplay** over a plain TCP relay. Run the solo game with one
command, or stand up a server and let a room full of friends fight.

## How it works

Each client runs its own deterministic game locally, so input is instant. The
server is a thin relay: it pairs players, hands every client the **same RNG
seed** (identical piece sequence = fair match), forwards **board snapshots** so
you can watch opponents, and routes **garbage** from whoever clears lines to a
random living opponent. Top out and you're eliminated; the last player alive
wins.

```
                 ┌────────────┐   garbage / snapshots   ┌────────────┐
   player A  ◄──►│  client A  │◄───────────────────────►│   relay    │
   (engine)      └────────────┘                          │  server    │
                 ┌────────────┐                          │ (rooms,    │
   player B  ◄──►│  client B  │◄────────────────────────►│  seed,     │
   (engine)      └────────────┘                          │  routing)  │
                       ⋮                                  └────────────┘
```

## Install

```sh
go install github.com/chalvinwz/hacktris/cmd/hacktris@latest
go install github.com/chalvinwz/hacktris/cmd/hacktris-server@latest
```

Or build from source:

```sh
make build      # -> bin/hacktris and bin/hacktris-server
```

## Play

**Solo:**

```sh
hacktris
```

**Multiplayer** — start a server (matches begin once enough players join), then
point clients at it:

```sh
# host: start a relay; a match starts as soon as 2 players are waiting
hacktris-server -addr :4321 -players 2

# each player:
hacktris -server HOST:4321 -name alice
hacktris -server HOST:4321 -name bob
```

Set `-players N` on the server for free-for-all rooms of any size; the room
starts once N players are waiting and ends when one remains.

## Controls

| Key            | Action       |
| -------------- | ------------ |
| `←` / `→`      | move         |
| `↑` / `x`      | rotate CW    |
| `z`            | rotate CCW   |
| `c`            | hold         |
| `↓`            | soft drop    |
| `space`        | hard drop    |
| `r`            | restart (solo) |
| `q` / `ctrl+c` | quit         |

## Rules

- **Board:** 10×20 well.
- **Rotation:** fixed-pivot SRS shapes, no wall kicks — a blocked rotation
  simply fails. Instant lock when a piece can't fall.
- **Modern comforts** on top of the NES core: **hold** (`c`, once per drop),
  a **ghost piece** showing where the current piece lands, and **hard drop**
  (`space`) that slams it down and locks — scored one point per cell.
- **Gravity & scoring:** NES NTSC speed curve and line-clear scoring; the level
  rises every ten lines.
- **Garbage:** clearing N lines sends `N-1` garbage rows to a random living
  opponent, applied when your next piece locks. Pushed off the top? You're out.

## Development

```sh
make check      # gofmt-check + vet + tests
make race       # tests under the race detector
make lint       # golangci-lint
make run        # solo game
make server     # relay server
```

Layout:

```
cmd/hacktris/         TUI client
cmd/hacktris-server/  relay server
internal/engine/      deterministic game core (no I/O)
internal/protocol/    wire messages + JSON codec
internal/client/      Bubble Tea model / update / view / net
internal/server/      lobby + per-match relay rooms
```

## License

[MIT](LICENSE) © 2026 chalvinwz
