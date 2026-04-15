package main

import (
	"fmt"
	"log"
	"os"

	"github.com/brunoborges/ghcd/internal/config"
	"github.com/brunoborges/ghcd/internal/daemon"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Printf("warning: config load: %v (using defaults)", err)
	}

	// Setup logging
	log.SetPrefix("ghcd: ")
	log.SetFlags(log.Ldate | log.Ltime | log.Lmsgprefix)

	if len(os.Args) > 1 && os.Args[1] == "--help" {
		fmt.Println("ghcd — GitHub CLI Cache Proxy Daemon")
		fmt.Println()
		fmt.Println("Usage: ghcd [--help]")
		fmt.Println()
		fmt.Printf("  Socket:    %s\n", cfg.SocketPath)
		fmt.Printf("  Dashboard: http://127.0.0.1:%d/\n", cfg.DashboardPort)
		fmt.Printf("  PID file:  %s\n", cfg.PIDFile)
		os.Exit(0)
	}

	srv := daemon.NewServer(cfg)
	if err := srv.Run(); err != nil {
		log.Fatalf("fatal: %v", err)
	}
}
