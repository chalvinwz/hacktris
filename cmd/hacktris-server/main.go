// Command hacktris-server relays matches between hacktris clients.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/chalvinwz/hacktris/internal/server"
)

// version is set at build time via -ldflags "-X main.version=…".
var version = "dev"

func main() {
	addr := flag.String("addr", ":4321", "listen address")
	players := flag.Int("players", 2, "players per match (minimum 2)")
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Println("hacktris-server", version)
		return
	}

	logger := log.New(os.Stdout, "", log.LstdFlags)
	srv := server.New(*players, logger)
	listenAddr, err := srv.Listen(*addr)
	if err != nil {
		logger.Fatalf("listen: %v", err)
	}
	logger.Printf("hacktris-server listening on %s, %d players per match", listenAddr, *players)
	if err := srv.Serve(); err != nil {
		logger.Fatalf("serve: %v", err)
	}
}
