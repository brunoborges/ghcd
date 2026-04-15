package main

import (
	"fmt"
	"log"
	"os"

	"github.com/brunoborges/ghx/internal/config"
	"github.com/brunoborges/ghx/internal/daemon"
)

var version = "dev"

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Printf("warning: config load: %v (using defaults)", err)
	}

	// Setup logging
	log.SetPrefix("ghxd: ")
	log.SetFlags(log.Ldate | log.Ltime | log.Lmsgprefix)

	if len(os.Args) > 1 && (os.Args[1] == "--version" || os.Args[1] == "-v") {
		fmt.Printf("ghxd version %s\n", version)
		os.Exit(0)
	}

	if len(os.Args) > 1 && os.Args[1] == "--help" {
		fmt.Println("ghxd — GitHub CLI Cache Proxy Daemon")
		fmt.Printf("Version: %s\n", version)
		fmt.Println()
		fmt.Println("Usage: ghxd [--help]")
		fmt.Println()
		fmt.Printf("  Socket:    %s\n", cfg.SocketPath)
		fmt.Printf("  Dashboard: http://127.0.0.1:%d/\n", cfg.DashboardPort)
		fmt.Printf("  PID file:  %s\n", cfg.PIDFile)
		os.Exit(0)
	}

	srv := daemon.NewServer(cfg, version)
	if err := srv.Run(); err != nil {
		log.Fatalf("fatal: %v", err)
	}
}
