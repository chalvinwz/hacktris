// Command hacktris is a classic NES-style Tetris for the terminal, with online
// multiplayer versus play.
package main

import (
	"flag"
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"

	"github.com/chalvinwz/hacktris/internal/client"
)

// version is set at build time via -ldflags "-X main.version=…".
var version = "dev"

func main() {
	server := flag.String("server", "", "relay server address (host:port); omit for solo play")
	name := flag.String("name", "", "player name shown to opponents")
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Println("hacktris", version)
		return
	}

	var model tea.Model
	if *server != "" {
		model = client.NewMultiplayer(*server, *name)
	} else {
		model = client.New()
	}

	if _, err := tea.NewProgram(model).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
