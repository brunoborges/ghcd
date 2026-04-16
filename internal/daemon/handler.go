package daemon

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/brunoborges/ghx/internal/allowlist"
	"github.com/brunoborges/ghx/internal/cache"
	"github.com/brunoborges/ghx/internal/config"
	execctx "github.com/brunoborges/ghx/internal/context"
	"github.com/brunoborges/ghx/internal/executor"
	"github.com/brunoborges/ghx/internal/metrics"
	"github.com/brunoborges/ghx/internal/protocol"
)

// Handler processes incoming requests from clients.
type Handler struct {
	cfg        *config.Config
	cache      *cache.Cache
	classifier *allowlist.Classifier
	stats      *metrics.Stats

	// singleflight: one in-flight request per cache key
	mu       sync.Mutex
	inflight map[string]*call
}

type call struct {
	wg  sync.WaitGroup
	res *executor.Result
}

func NewHandler(cfg *config.Config, c *cache.Cache, cl *allowlist.Classifier, s *metrics.Stats) *Handler {
	h := &Handler{
		cfg:        cfg,
		cache:      c,
		classifier: cl,
		stats:      s,
		inflight:   make(map[string]*call),
	}
	c.OnEvict(func(key string) {
		s.RecordEviction()
	})
	return h
}

// Handle processes a single client request and returns a response.
func (h *Handler) Handle(req *protocol.Request) *protocol.Response {
	switch req.Type {
	case protocol.TypeStats:
		return h.handleStats()
	case protocol.TypeFlush:
		return h.handleFlush(req)
	case protocol.TypeKeys:
		return h.handleKeys()
	case protocol.TypeShutdown:
		return &protocol.Response{Stdout: []byte("shutting down\n")}
	default:
		return h.handleExec(req)
	}
}

func (h *Handler) handleExec(req *protocol.Request) *protocol.Response {
	start := time.Now()

	classification := h.classifier.Classify(req.Args)
	cmdKey := classification.CmdKey
	if cmdKey == "" {
		cmdKey = strings.Join(req.Args, "_")
		if len(cmdKey) > 40 {
			cmdKey = cmdKey[:40]
		}
	}

	// Non-cacheable: execute directly via daemon (captures output)
	if classification.Type == allowlist.Passthrough || req.NoCache {
		result := executor.Execute(context.Background(), h.cfg.GHPath, req.Args, req.WorkDir)
		latency := time.Since(start).Seconds() * 1000
		h.stats.Record(cmdKey, "", metrics.ResultPassthrough, latency)

		return &protocol.Response{
			Stdout:   result.Stdout,
			Stderr:   result.Stderr,
			ExitCode: result.ExitCode,
		}
	}

	// Mutation: execute directly, then invalidate
	if classification.Type == allowlist.Mutation {
		result := executor.Execute(context.Background(), h.cfg.GHPath, req.Args, req.WorkDir)
		latency := time.Since(start).Seconds() * 1000
		h.stats.Record(cmdKey, "", metrics.ResultPassthrough, latency)

		if classification.Resource != allowlist.ResourceUnknown {
			count := h.cache.InvalidateNamespace(req.Context.Host, req.Context.Repo, classification.Resource)
			h.stats.RecordInvalidation(count)
		}

		return &protocol.Response{
			Stdout:   result.Stdout,
			Stderr:   result.Stderr,
			ExitCode: result.ExitCode,
		}
	}

	// Cacheable: check cache → singleflight → execute → store
	cacheKey := execctx.CacheKey(req.Context, req.Args)

	// Check cache
	if entry := h.cache.Get(cacheKey); entry != nil {
		latency := time.Since(start).Seconds() * 1000
		h.stats.Record(cmdKey, cacheKey, metrics.ResultHit, latency)
		return &protocol.Response{
			Stdout:   entry.Stdout,
			Stderr:   entry.Stderr,
			ExitCode: entry.ExitCode,
			Cached:   true,
		}
	}

	// Singleflight: coalesce concurrent requests for the same key
	result, coalesced := h.doSingleflight(cacheKey, req)
	latency := time.Since(start).Seconds() * 1000

	if coalesced {
		h.stats.Record(cmdKey, cacheKey, metrics.ResultCoalesced, latency)
	} else {
		h.stats.Record(cmdKey, cacheKey, metrics.ResultMiss, latency)
	}

	// Store in cache
	ttl := h.cfg.CommandTTL(cmdKey)
	if req.TTLOverride > 0 {
		ttl = time.Duration(req.TTLOverride) * time.Second
	}

	h.cache.Set(&cache.Entry{
		Key:      cacheKey,
		Stdout:   result.Stdout,
		Stderr:   result.Stderr,
		ExitCode: result.ExitCode,
		CachedAt: time.Now(),
		TTL:      ttl,
		Resource: classification.Resource,
		Host:     req.Context.Host,
		Repo:     req.Context.Repo,
	})

	return &protocol.Response{
		Stdout:   result.Stdout,
		Stderr:   result.Stderr,
		ExitCode: result.ExitCode,
	}
}

func (h *Handler) doSingleflight(key string, req *protocol.Request) (*executor.Result, bool) {
	h.mu.Lock()
	if c, ok := h.inflight[key]; ok {
		h.mu.Unlock()
		c.wg.Wait()
		return c.res, true // coalesced
	}

	c := &call{}
	c.wg.Add(1)
	h.inflight[key] = c
	h.mu.Unlock()

	c.res = executor.Execute(context.Background(), h.cfg.GHPath, req.Args, req.WorkDir)
	c.wg.Done()

	h.mu.Lock()
	delete(h.inflight, key)
	h.mu.Unlock()

	return c.res, false
}

func (h *Handler) handleStats() *protocol.Response {
	snap := h.stats.Snapshot(h.cache.Size(), h.cfg.MaxCacheEntries)
	data, _ := json.Marshal(snap)
	return &protocol.Response{Stdout: data}
}

func (h *Handler) handleFlush(req *protocol.Request) *protocol.Response {
	var resource allowlist.ResourceType
	if len(req.Args) > 0 {
		resource = allowlist.ResourceType(req.Args[0])
	}
	count := h.cache.Flush(resource)
	msg := []byte(fmt.Sprintf("flushed %d entries\n", count))
	return &protocol.Response{Stdout: msg}
}

func (h *Handler) handleKeys() *protocol.Response {
	keys := h.cache.Keys()
	data, _ := json.Marshal(keys)
	return &protocol.Response{Stdout: data}
}
