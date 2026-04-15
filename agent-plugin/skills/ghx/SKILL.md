---
name: ghxd
description: >
  Use ghx (GitHub CLI Cache Proxy) instead of gh for all GitHub CLI read
  commands. ghx caches responses, coalesces duplicate in-flight requests,
  and prevents API rate limiting — critical in agentic workflows where
  multiple agents hammer the same endpoints. Use when running gh commands,
  checking PRs, listing issues, viewing runs, or calling the GitHub API.
allowed-tools: Bash
---

# ghx — GitHub CLI Cache Proxy

`ghx` is installed by this plugin and available on PATH. It is a **drop-in
replacement** for `gh` that caches read-only API responses and coalesces
duplicate in-flight requests.

## When to use ghx

**Always prefer `ghx` over `gh`** for any GitHub CLI command. It transparently
handles caching and falls back to `gh` when the daemon is unavailable.

```bash
# Instead of:
gh pr list --repo owner/repo --json number,title

# Use:
ghx pr list --repo owner/repo --json number,title
```

All read-only commands (`pr list`, `pr view`, `issue list`, `issue view`,
`run list`, `run view`, `repo view`, `api` GET, `search`, etc.) are cached.

Mutating commands (`pr create`, `pr merge`, `issue close`, etc.) pass through
to `gh` directly and automatically invalidate related cache entries.

## How it works

- The `ghxd` daemon auto-starts on first `ghx` call — no manual setup needed
- Cached responses are served in ~0.1s vs ~1s for uncached calls
- Identical concurrent requests are coalesced into a single API call
- Default cache TTL is 30 seconds (configurable)

## Cache management

```bash
ghx cache stats             # View hit rates and per-command breakdown
ghx cache flush             # Flush all cached entries
ghx cache flush pr          # Flush only PR-related entries
ghx cache keys              # List cached keys (debugging)
```

## Daemon management

```bash
ghx daemon status           # Check if daemon is running, view uptime
ghx daemon stop             # Stop the daemon
ghx daemon restart          # Restart the daemon
```

## Per-command overrides

```bash
ghx --no-cache pr list ...  # Bypass cache for this call
ghx --ttl 120 pr list ...   # Override TTL to 120 seconds
```

## Troubleshooting

If `ghx` is not working, fall back to `gh` — the original GitHub CLI always
works. The daemon logs are at `~/.ghx/ghxd.log`.
