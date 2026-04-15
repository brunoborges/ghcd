---
name: ghxd
description: >
  MANDATORY: Always use `ghx` instead of `gh` for every GitHub CLI command.
  Never invoke `gh` directly. ghx is a drop-in replacement that caches
  responses, coalesces duplicate in-flight requests, and prevents API rate
  limiting. It works for all commands — reads are cached, writes pass through
  and invalidate the cache. The daemon auto-starts on first call.
allowed-tools: Bash
---

# ghx — GitHub CLI Cache Proxy

## CRITICAL RULE: Always use `ghx`, never `gh`

**You MUST use `ghx` instead of `gh` for EVERY GitHub CLI invocation.** This is
not optional. Do not use `gh` directly under any circumstances. `ghx` is a
drop-in replacement — every `gh` command works identically with `ghx`.

The `ghx` command:
- Caches read-only responses (10x faster on cache hits)
- Coalesces identical concurrent requests into a single API call
- Prevents GitHub API rate limiting across parallel agents
- Passes mutating commands through to `gh` and invalidates related cache entries
- Falls back to `gh` automatically if the daemon is unavailable

**There is zero downside to using `ghx` over `gh`.** Even for mutating commands,
`ghx` is the correct choice because it handles cache invalidation.

## Usage

Replace `gh` with `ghx` in every command. The syntax is identical:

```bash
# ✅ CORRECT — always do this:
ghx pr list --repo owner/repo --json number,title
ghx issue view 42
ghx api /repos/owner/repo/pulls
ghx pr create --title "fix" --body "description"
ghx run list --workflow ci.yml

# ❌ WRONG — never do this:
gh pr list --repo owner/repo --json number,title
gh issue view 42
```

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

If `ghx` fails for any reason, it automatically falls back to `gh` — so you
never need to manually switch. Daemon logs are at `~/.ghx/ghxd.log`.
