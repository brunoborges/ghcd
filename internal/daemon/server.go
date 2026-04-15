package daemon

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/brunoborges/ghx/internal/allowlist"
	"github.com/brunoborges/ghx/internal/cache"
	"github.com/brunoborges/ghx/internal/config"
	"github.com/brunoborges/ghx/internal/dashboard"
	"github.com/brunoborges/ghx/internal/metrics"
	"github.com/brunoborges/ghx/internal/protocol"
)

// Server is the ghxd daemon.
type Server struct {
	cfg     *config.Config
	cache   *cache.Cache
	stats   *metrics.Stats
	handler *Handler
	ln      net.Listener
	httpSrv *http.Server
	done    chan struct{}
	wg      sync.WaitGroup
	version string
}

// NewServer creates a new daemon server.
func NewServer(cfg *config.Config, version string) *Server {
	c := cache.New(cfg.MaxCacheEntries)
	stats := metrics.New()
	classifier := allowlist.NewClassifier(cfg.AdditionalCache)
	handler := NewHandler(cfg, c, classifier, stats)

	return &Server{
		cfg:     cfg,
		cache:   c,
		stats:   stats,
		handler: handler,
		done:    make(chan struct{}),
		version: version,
	}
}

// Run starts the daemon and blocks until shutdown.
func (s *Server) Run() error {
	// Ensure socket directory exists
	socketDir := filepath.Dir(s.cfg.SocketPath)
	if err := os.MkdirAll(socketDir, 0700); err != nil {
		return fmt.Errorf("create socket dir: %w", err)
	}

	// Remove stale socket
	os.Remove(s.cfg.SocketPath)

	// Start Unix socket listener
	ln, err := net.Listen("unix", s.cfg.SocketPath)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	s.ln = ln

	// Set socket permissions
	if err := os.Chmod(s.cfg.SocketPath, 0600); err != nil {
		ln.Close()
		return fmt.Errorf("chmod socket: %w", err)
	}

	// Write PID file
	if err := s.writePIDFile(); err != nil {
		ln.Close()
		return fmt.Errorf("write pid: %w", err)
	}
	defer s.removePIDFile()

	// Start HTTP server for dashboard
	s.startHTTP()

	// Handle signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		select {
		case <-sigCh:
			log.Println("received shutdown signal")
			s.Shutdown()
		case <-s.done:
		}
	}()

	log.Printf("ghxd started (socket: %s, dashboard: http://127.0.0.1:%d/)\n", s.cfg.SocketPath, s.cfg.DashboardPort)

	// Accept connections
	for {
		conn, err := ln.Accept()
		if err != nil {
			select {
			case <-s.done:
				s.wg.Wait()
				return nil
			default:
				log.Printf("accept error: %v", err)
				continue
			}
		}

		s.wg.Add(1)
		go s.handleConn(conn)
	}
}

func (s *Server) handleConn(conn net.Conn) {
	defer s.wg.Done()
	defer conn.Close()

	conn.SetDeadline(time.Now().Add(60 * time.Second))

	var req protocol.Request
	if err := protocol.ReadMessage(conn, &req); err != nil {
		// Don't log EOF — it's just a health check probe from IsRunning()
		if err.Error() != "read length: EOF" {
			log.Printf("read request: %v", err)
		}
		return
	}

	resp := s.handler.Handle(&req)

	// Check for shutdown request
	if req.Type == protocol.TypeShutdown {
		protocol.WriteMessage(conn, resp)
		go s.Shutdown()
		return
	}

	if err := protocol.WriteMessage(conn, resp); err != nil {
		log.Printf("write response: %v", err)
	}
}

func (s *Server) startHTTP() {
	mux := http.NewServeMux()

	// Dashboard
	mux.HandleFunc("/", dashboard.Handler())

	// JSON API
	mux.HandleFunc("/api/stats", func(w http.ResponseWriter, r *http.Request) {
		snap := s.stats.Snapshot(s.cache.Size(), s.cfg.MaxCacheEntries)
		resp := struct {
			Version string `json:"version"`
			metrics.Snapshot
		}{s.version, snap}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		json.NewEncoder(w).Encode(resp)
	})

	mux.HandleFunc("/api/log", func(w http.ResponseWriter, r *http.Request) {
		const maxLogLimit = 1000
		limit := 200
		if v := r.URL.Query().Get("limit"); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n > 0 {
				if n > maxLogLimit {
					limit = maxLogLimit
				} else {
					limit = n
				}
			}
		}
		entries := s.stats.Log(limit)
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		json.NewEncoder(w).Encode(entries)
	})

	mux.HandleFunc("/api/ttl-analysis", func(w http.ResponseWriter, r *http.Request) {
		recs := s.stats.TTLAnalysis()
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		json.NewEncoder(w).Encode(recs)
	})

	// Mutating endpoints — POST only, origin-validated
	mux.HandleFunc("/api/flush", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		resource := allowlist.ResourceType(r.URL.Query().Get("resource"))
		count := s.cache.Flush(resource)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"flushed": count})
	})

	mux.HandleFunc("/api/shutdown", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "shutting_down"})
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
		go func() {
			time.Sleep(100 * time.Millisecond)
			s.Shutdown()
		}()
	})

	s.httpSrv = &http.Server{
		Addr:    fmt.Sprintf("127.0.0.1:%d", s.cfg.DashboardPort),
		Handler: mux,
	}

	go func() {
		if err := s.httpSrv.ListenAndServe(); err != http.ErrServerClosed {
			log.Printf("HTTP server error: %v", err)
		}
	}()
}

// Shutdown gracefully stops the daemon.
func (s *Server) Shutdown() {
	select {
	case <-s.done:
		return // already shutting down
	default:
		close(s.done)
	}

	// Stop HTTP server
	if s.httpSrv != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		s.httpSrv.Shutdown(ctx)
	}

	// Stop accepting connections
	if s.ln != nil {
		s.ln.Close()
	}

	log.Println("ghxd shutdown complete")
}

func (s *Server) writePIDFile() error {
	dir := filepath.Dir(s.cfg.PIDFile)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	return os.WriteFile(s.cfg.PIDFile, []byte(strconv.Itoa(os.Getpid())), 0600)
}

func (s *Server) removePIDFile() {
	os.Remove(s.cfg.PIDFile)
	os.Remove(s.cfg.SocketPath)
}
