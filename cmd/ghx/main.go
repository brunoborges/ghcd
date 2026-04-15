package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/brunoborges/ghx/internal/client"
	"github.com/brunoborges/ghx/internal/config"
	execctx "github.com/brunoborges/ghx/internal/context"
	"github.com/brunoborges/ghx/internal/protocol"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "ghx: warning: config load: %v\n", err)
	}

	args := os.Args[1:]
	if len(args) == 0 {
		execDirect(cfg.GHPath, nil)
		return
	}

	// Handle ghx-specific subcommands
	switch args[0] {
	case "daemon":
		handleDaemon(cfg, args[1:])
		return
	case "cache":
		handleCache(cfg, args[1:])
		return
	}

	// Parse ghx flags (before the gh args)
	noCache := os.Getenv("GHX_NO_CACHE") == "1"
	ttlOverride := 0
	var ghArgs []string

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--no-cache":
			noCache = true
		case "--ttl":
			if i+1 < len(args) {
				i++
				if v, err := strconv.Atoi(args[i]); err == nil {
					ttlOverride = v
				}
			}
		default:
			ghArgs = append(ghArgs, args[i:]...)
			i = len(args) // break
		}
	}

	if len(ghArgs) == 0 {
		execDirect(cfg.GHPath, nil)
		return
	}

	// Resolve execution context
	ctx := execctx.Resolve(cfg.GHPath)

	// Connect to daemon
	cl := client.New(cfg.SocketPath)

	// Auto-start daemon if not running
	if !cl.IsRunning() {
		if cfg.AutoStart {
			if err := startDaemon(cfg); err != nil {
				fmt.Fprintf(os.Stderr, "ghx: daemon auto-start failed: %v (falling back to direct gh)\n", err)
				execDirect(cfg.GHPath, ghArgs)
				return
			}
			// Wait for daemon to be ready
			for i := 0; i < 20; i++ {
				if cl.IsRunning() {
					break
				}
				time.Sleep(100 * time.Millisecond)
			}
			if !cl.IsRunning() {
				fmt.Fprintf(os.Stderr, "ghx: daemon failed to start in time (falling back to direct gh)\n")
				execDirect(cfg.GHPath, ghArgs)
				return
			}
		} else {
			execDirect(cfg.GHPath, ghArgs)
			return
		}
	}

	// Send request to daemon
	req := &protocol.Request{
		Type:        protocol.TypeExec,
		Args:        ghArgs,
		Context:     ctx,
		NoCache:     noCache,
		TTLOverride: ttlOverride,
	}

	resp, err := cl.Send(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ghx: daemon error: %v (falling back to direct gh)\n", err)
		execDirect(cfg.GHPath, ghArgs)
		return
	}

	if len(resp.Stdout) > 0 {
		os.Stdout.Write(resp.Stdout)
	}
	if len(resp.Stderr) > 0 {
		os.Stderr.Write(resp.Stderr)
	}
	os.Exit(resp.ExitCode)
}

func handleDaemon(cfg *config.Config, args []string) {
	if len(args) == 0 {
		fmt.Println("Usage: ghx daemon <start|stop|status|restart>")
		os.Exit(1)
	}

	switch args[0] {
	case "start":
		detach := false
		for _, a := range args[1:] {
			if a == "-d" || a == "--detach" {
				detach = true
			}
		}
		if detach {
			if err := startDaemon(cfg); err != nil {
				fmt.Fprintf(os.Stderr, "ghx: failed to start daemon: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("ghx: daemon started (socket: %s, dashboard: http://127.0.0.1:%d/)\n", cfg.SocketPath, cfg.DashboardPort)
		} else {
			// Foreground mode — exec the daemon directly
			ghxdPath, err := findGHXD()
			if err != nil {
				fmt.Fprintf(os.Stderr, "ghx: %v\n", err)
				os.Exit(1)
			}
			execReplace(ghxdPath, []string{"ghxd"}, os.Environ())
		}

	case "stop":
		cl := client.New(cfg.SocketPath)
		if !cl.IsRunning() {
			fmt.Println("ghx: daemon is not running")
			return
		}
		resp, err := cl.Send(&protocol.Request{Type: protocol.TypeShutdown})
		if err != nil {
			fmt.Fprintf(os.Stderr, "ghx: stop failed: %v\n", err)
			os.Exit(1)
		}
		os.Stdout.Write(resp.Stdout)

	case "status":
		cl := client.New(cfg.SocketPath)
		if !cl.IsRunning() {
			fmt.Println("ghx: daemon is not running")
			os.Exit(1)
		}
		resp, err := cl.Send(&protocol.Request{Type: protocol.TypeStats})
		if err != nil {
			fmt.Fprintf(os.Stderr, "ghx: %v\n", err)
			os.Exit(1)
		}
		printFormattedStats(resp.Stdout)

	case "restart":
		handleDaemon(cfg, []string{"stop"})
		time.Sleep(500 * time.Millisecond)
		handleDaemon(cfg, []string{"start", "-d"})

	default:
		fmt.Fprintf(os.Stderr, "ghx: unknown daemon command: %s\n", args[0])
		os.Exit(1)
	}
}

func handleCache(cfg *config.Config, args []string) {
	if len(args) == 0 {
		fmt.Println("Usage: ghx cache <stats|flush|keys>")
		os.Exit(1)
	}

	cl := client.New(cfg.SocketPath)
	if !cl.IsRunning() {
		fmt.Println("ghx: daemon is not running")
		os.Exit(1)
	}

	switch args[0] {
	case "stats":
		resp, err := cl.Send(&protocol.Request{Type: protocol.TypeStats})
		if err != nil {
			fmt.Fprintf(os.Stderr, "ghx: %v\n", err)
			os.Exit(1)
		}
		printFormattedStats(resp.Stdout)

	case "flush":
		flushArgs := args[1:]
		resp, err := cl.Send(&protocol.Request{Type: protocol.TypeFlush, Args: flushArgs})
		if err != nil {
			fmt.Fprintf(os.Stderr, "ghx: %v\n", err)
			os.Exit(1)
		}
		os.Stdout.Write(resp.Stdout)

	case "keys":
		resp, err := cl.Send(&protocol.Request{Type: protocol.TypeKeys})
		if err != nil {
			fmt.Fprintf(os.Stderr, "ghx: %v\n", err)
			os.Exit(1)
		}
		var keys []string
		json.Unmarshal(resp.Stdout, &keys)
		for _, k := range keys {
			fmt.Println(k)
		}

	default:
		fmt.Fprintf(os.Stderr, "ghx: unknown cache command: %s\n", args[0])
		os.Exit(1)
	}
}

func printFormattedStats(data []byte) {
	var stats struct {
		Uptime       string                     `json:"uptime"`
		Total        int64                      `json:"total"`
		Hits         int64                      `json:"hits"`
		Misses       int64                      `json:"misses"`
		Passthrough  int64                      `json:"passthrough"`
		Coalesced    int64                      `json:"coalesced"`
		HitRate      float64                    `json:"hit_rate"`
		CacheSize    int                        `json:"cache_size"`
		MaxCacheSize int                        `json:"max_cache_size"`
		Commands     map[string]json.RawMessage `json:"commands"`
	}
	if err := json.Unmarshal(data, &stats); err != nil {
		os.Stdout.Write(data)
		return
	}

	fmt.Printf("Uptime:          %s\n", stats.Uptime)
	fmt.Printf("Total Requests:  %d\n", stats.Total)
	fmt.Printf("Cache Hits:      %d (%.1f%%)\n", stats.Hits, stats.HitRate)
	fmt.Printf("Cache Misses:    %d\n", stats.Misses)
	fmt.Printf("Passthrough:     %d\n", stats.Passthrough)
	fmt.Printf("Coalesced:       %d\n", stats.Coalesced)
	fmt.Printf("Cache Size:      %d / %d entries\n", stats.CacheSize, stats.MaxCacheSize)

	if len(stats.Commands) > 0 {
		fmt.Println("\nTop Commands:")
		for name, raw := range stats.Commands {
			var cmd struct {
				Hits   int64 `json:"hits"`
				Misses int64 `json:"misses"`
			}
			json.Unmarshal(raw, &cmd)
			total := cmd.Hits + cmd.Misses
			rate := 0.0
			if total > 0 {
				rate = float64(cmd.Hits) / float64(total) * 100
			}
			fmt.Printf("  %-24s %d hits / %d misses  (%.1f%%)\n",
				strings.ReplaceAll(name, "_", " "), cmd.Hits, cmd.Misses, rate)
		}
	}
}

// findGHXD locates the ghxd binary.
func findGHXD() (string, error) {
	suffix := ""
	if runtime.GOOS == "windows" {
		suffix = ".exe"
	}

	// Check next to the ghx binary
	exe, err := os.Executable()
	if err == nil {
		dir := filepath.Dir(exe)
		candidate := filepath.Join(dir, "ghxd"+suffix)
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}

	// Check PATH
	path, err := exec.LookPath("ghxd" + suffix)
	if err != nil {
		return "", fmt.Errorf("ghxd not found (install it next to ghx or in PATH)")
	}
	return path, nil
}
