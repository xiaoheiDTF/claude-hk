package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/backend"
)

func runBackend(args []string) {
	cfg := backend.DefaultConfig()

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case (arg == "--port" || arg == "-p") && i+1 < len(args):
			i++
			fmt.Sscanf(args[i], "%d", &cfg.Port)
		case strings.HasPrefix(arg, "--port="):
			fmt.Sscanf(arg[len("--port="):], "%d", &cfg.Port)
		case (arg == "--db" || arg == "-d") && i+1 < len(args):
			i++
			cfg.DBPath = args[i]
		case strings.HasPrefix(arg, "--db="):
			cfg.DBPath = arg[len("--db="):]
		case arg == "--host" && i+1 < len(args):
			i++
			cfg.Host = args[i]
		case strings.HasPrefix(arg, "--host="):
			cfg.Host = arg[len("--host="):]
		}
	}

	srv, err := backend.NewServer(cfg)
	if err != nil {
		log.Fatalf("create server: %v", err)
	}
	defer srv.Close()

	if err := srv.Start(); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
