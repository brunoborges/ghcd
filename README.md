# ghx — GitHub CLI Cache Proxy

<p align="center">
  <img src="ghx-dashboard.png" alt="ghx Dashboard" width="700">
</p>

A caching proxy for the [GitHub CLI (`gh`)](https://cli.github.com/) that eliminates redundant API calls, prevents rate limiting, and dramatically speeds up repeated commands.

**Built for AI agent workflows** where multiple agents (Copilot CLI, coding agents, MCP servers) hammer the same `gh` commands simultaneously.

## Highlights

- 🚀 **10x faster** cached responses (~0.1s vs ~1s)
- 🔄 **Singleflight coalescing** — 5 agents asking the same thing = 1 API call
- 🎯 **Allowlist-based** — only caches known-safe read-only commands
- 🧹 **Auto-invalidation** — mutations flush related cache entries
- 📊 **Web dashboard** — real-time hit rates, per-command stats, request log
- 🔌 **Drop-in replacement** — just use `ghx` instead of `gh`

## Install

### Homebrew (recommended)

```bash
brew tap brunoborges/tap
brew install ghxd
```

### Quick install script

```bash
curl -fsSL https://raw.githubusercontent.com/brunoborges/ghx/main/install.sh | bash
```

This detects your OS and architecture, downloads the latest release, and installs `ghx` and `ghxd` to `/usr/local/bin`. To install elsewhere:

```bash
curl -fsSL https://raw.githubusercontent.com/brunoborges/ghx/main/install.sh | INSTALL_DIR=~/.local/bin bash
```

### Manual download

Download the latest release for your platform from [GitHub Releases](https://github.com/brunoborges/ghx/releases):

```bash
# macOS (Apple Silicon)
curl -fsSL https://github.com/brunoborges/ghx/releases/latest/download/ghx-darwin-arm64.tar.gz | tar xz
sudo cp ghx ghxd /usr/local/bin/

# Linux (x64)
curl -fsSL https://github.com/brunoborges/ghx/releases/latest/download/ghx-linux-amd64.tar.gz | tar xz
sudo cp ghx ghxd /usr/local/bin/

# Linux (arm64)
curl -fsSL https://github.com/brunoborges/ghx/releases/latest/download/ghx-linux-arm64.tar.gz | tar xz
sudo cp ghx ghxd /usr/local/bin/
```

### Build from source

```bash
git clone https://github.com/brunoborges/ghx.git
cd ghxd
make build
# Binaries are in bin/ghx and bin/ghxd
sudo cp bin/ghx bin/ghxd /usr/local/bin/
```

### Agents Plugin (Claude Code & Copilot CLI)

If you use [Claude Code](https://code.claude.com/docs/en/plugins) or [GitHub Copilot CLI](https://docs.github.com/en/copilot/concepts/agents/copilot-cli/about-cli-plugins), install the plugin and your agent will automatically prefer `ghx` over `gh`:

```bash
# Add the marketplace (one-time)
/plugin marketplace add brunoborges/agent-plugins

# Install the plugin
/plugin install ghx@agent-plugins
```

> **Local development / testing:** `claude --plugin-dir ./agent-plugin`

The plugin:
- **Lazy-installs** `ghx` and `ghxd` binaries on first use
- **Adds `ghx` to PATH** so agents use it automatically
- **Includes a skill** that teaches agents to prefer `ghx` for all GitHub CLI calls

See the [plugin README](agent-plugin/README.md) for details. Plugin releases are available on the [Releases page](https://github.com/brunoborges/ghx/releases) with the `plugin-v*` tag.

## Usage

Use `ghx` exactly like `gh` — the daemon starts automatically on first use:

```bash
# These are cached (read-only commands)
ghx pr list --repo owner/repo --json number,title
ghx issue view 42 --json title,state
ghx api /repos/owner/repo --jq '.stargazers_count'
ghx run list --repo owner/repo

# These pass through directly to gh (mutations)
ghx pr create --title "My PR" --body "Description"
ghx issue close 42
```

### Cache behavior

```
First call:   ghx pr list ...   → 1.1s (cache miss, calls gh)
Second call:  ghx pr list ...   → 0.1s (cache hit, instant)
After 30s:    ghx pr list ...   → 1.0s (TTL expired, fresh call)
```

### Per-command options

```bash
ghx --no-cache pr list ...     # Bypass cache for this call
ghx --ttl 120 pr list ...      # Override TTL to 120 seconds
GHX_NO_CACHE=1 ghx pr list ... # Same via env var
GHX_TTL=60 ghx pr list ...     # Same via env var
```

## Daemon Management

```bash
ghx daemon start          # Start in foreground
ghx daemon start -d       # Start detached (background)
ghx daemon stop           # Graceful shutdown
ghx daemon status         # Show uptime and cache stats
ghx daemon restart        # Stop + start
```

The daemon auto-starts on first `ghx` call. If the daemon can't start, `ghx` falls back to running `gh` directly — it never blocks you.

## Cache Management

```bash
ghx cache stats           # Show hit rates and per-command breakdown
ghx cache flush           # Flush all entries
ghx cache flush pr        # Flush PR-related entries only
ghx cache keys            # List cached keys (debugging)
```

### Example stats output

```
Uptime:          2h 34m
Total Requests:  1,247
Cache Hits:      891 (71.4%)
Cache Misses:    203 (16.3%)
Passthrough:     153 (12.3%)
Coalesced:       87
Cache Size:      142 / 1000 entries

Top Commands:
  pr list                  412 hits / 48 misses  (89.6%)
  issue view               198 hits / 32 misses  (86.1%)
  pr view                  143 hits / 67 misses  (68.1%)
  api get                   88 hits / 31 misses  (73.9%)
```

## Web Dashboard

When the daemon is running, a live dashboard is available at:

```
http://localhost:9847/
```

It shows:
- **Real-time hit rate** and request counters
- **Per-command breakdown** with hit/miss rates and average latency
- **Request log** — live tail of recent requests with cache result and timing

The dashboard auto-refreshes every 2 seconds. No external dependencies — it's a single HTML page embedded in the binary.

### JSON API

The dashboard data is also available as JSON for scripting:

```bash
curl http://localhost:9847/api/stats | jq .
curl http://localhost:9847/api/log?limit=50 | jq .
curl http://localhost:9847/api/ttl-analysis | jq .
```

## What Gets Cached

Only explicitly allowlisted read-only commands are cached:

| Command | Cached |
|---------|--------|
| `gh pr list/view/status/checks/diff` | ✅ |
| `gh issue list/view/status` | ✅ |
| `gh repo view/list` | ✅ |
| `gh run list/view` | ✅ |
| `gh workflow list/view` | ✅ |
| `gh release list/view` | ✅ |
| `gh search repos/issues/prs/commits/code` | ✅ |
| `gh api` (GET only) | ✅ |
| `gh label list` | ✅ |
| `gh gist list/view` | ✅ |
| `gh project list/view` | ✅ |
| `gh cache list` | ✅ |
| `gh ruleset list/view/check` | ✅ |
| `gh org list` | ✅ |
| `gh pr create/merge/close/edit` | ❌ (mutation → invalidates PR cache) |
| `gh issue create/edit/delete/close` | ❌ (mutation → invalidates issue cache) |
| `gh auth/config/codespace/secret` | ❌ (always passthrough) |

Mutations automatically invalidate related cache entries. For example, `gh pr merge 42` flushes all cached PR entries for that repo.

### Custom Commands

You can add your own commands to the allowlist via `~/.ghx/config.yaml`:

```yaml
additional_cacheable:
  - "gh status"
  - "gh variable list"
  - "gh secret list"
```

Each entry should be the full command prefix (e.g., `"gh status"` for a single-word subcommand, or `"gh variable list"` for two-word). Custom commands are classified with `ResourceUnknown` — they participate in caching but won't be invalidated by mutation detection. To apply changes, restart the daemon: `ghxd --restart`.

## Configuration

Configuration file: `~/.ghx/config.yaml`

```yaml
# Default TTL for all cached commands (default: 30s)
ttl: 30s

# Per-command TTL overrides
ttl_overrides:
  pr_list: 60s
  pr_view: 30s
  issue_list: 60s
  run_list: 15s

# Max cached entries before LRU eviction (default: 1000)
max_cache_entries: 1000

# Dashboard HTTP port (default: 9847)
dashboard_port: 9847

# Auto-start daemon on first ghx call (default: true)
auto_start: true

# Additional commands to cache
additional_cacheable:
  - "gh status"

# Path to gh binary (defaults to "gh" from PATH)
# gh_path: /usr/local/bin/gh
```

## Architecture

```
┌─────────┐  ┌─────────┐  ┌─────────┐
│ Agent 1 │  │ Agent 2 │  │ Agent 3 │
│ (ghx)   │  │ (ghx)   │  │ (ghx)   │
└────┬────┘  └────┬────┘  └────┬────┘
     │            │            │
     └────────────┼────────────┘
                  │ Unix Domain Socket
           ┌──────┴──────┐
           │    ghxd     │
           │  (daemon)   │
           ├─────────────┤
           │ Cache (LRU) │
           │ Singleflight│
           │ Metrics     │
           │ Dashboard   │
           └──────┬──────┘
                  │ exec
              ┌───┴───┐
              │  gh   │
              └───────┘
```

**Key design decisions:**
- **Allowlist, not denylist** — only known-safe commands are cached
- **Context-aware cache keys** — includes repo, branch, host, and auth token hash to prevent cross-context collisions
- **Singleflight** — concurrent identical requests share a single `gh` execution
- **Coarse invalidation** — mutations flush the entire resource namespace (all PR cache for that repo)
- **Graceful fallback** — if daemon is down or fails, `ghx` runs `gh` directly

## Security

- Unix socket with `0600` permissions (owner-only access)
- Auth tokens are never stored — only a SHA256 fingerprint is used in cache keys
- Dashboard binds to `127.0.0.1` only (not accessible from network)
- In-memory cache only (lost on daemon restart)
- Each user runs their own isolated daemon

## Development

```bash
# Build
make build

# Run tests
make test

# Clean
make clean
```

## License

[MIT](LICENSE)
