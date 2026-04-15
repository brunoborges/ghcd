# ghcd — Claude Code Plugin

A [Claude Code Plugin](https://code.claude.com/docs/en/plugins) that installs and configures [ghcd](https://github.com/brunoborges/ghcd) — a caching proxy for the GitHub CLI (`gh`).

Also compatible with [GitHub Copilot CLI](https://github.com/github/copilot-cli) and other agentic CLI runners that support the Claude Plugin format.

## What it does

When enabled, this plugin:

1. **Installs `ghc` and `ghcd`** automatically on first use (lazy install)
2. **Adds `ghc` to PATH** so agents can use it as a drop-in replacement for `gh`
3. **Teaches Claude to prefer `ghc`** over `gh` via a built-in skill, so all GitHub CLI calls go through the caching proxy

This eliminates redundant API calls, prevents rate limiting, and dramatically speeds up repeated `gh` commands in agentic workflows.

## Install

### From a marketplace

```
/plugin install <marketplace-url>
```

### Local development / testing

```bash
claude --plugin-dir ./agents-plugin
```

Or from the repo root:

```bash
claude --plugin-dir ./agents-plugin
```

## How it works

### Lazy binary installation

The plugin ships wrapper scripts in `bin/` that are automatically added to PATH. On first invocation, the wrapper downloads and installs the real `ghc` and `ghcd` binaries to the plugin's persistent data directory (`${CLAUDE_PLUGIN_DATA}/bin`).

To pin a specific version:

```bash
GHCD_VERSION=v1.0.0 ghc pr list
```

### Skill: automatic `ghc` preference

The plugin includes a skill that instructs Claude to use `ghc` instead of `gh` for all GitHub CLI commands. Claude loads this skill automatically when relevant — no manual invocation needed.

You can also invoke it explicitly:

```
/ghcd:ghcd
```

### Cache behavior

```
First call:   ghc pr list ...   → ~1.1s (cache miss, calls gh)
Second call:  ghc pr list ...   → ~0.1s (cache hit, instant)
After TTL:    ghc pr list ...   → ~1.0s (TTL expired, fresh call)
```

## Plugin structure

```
agents-plugin/
├── .claude-plugin/
│   └── plugin.json      # Plugin manifest
├── bin/
│   ├── ghc              # Wrapper: lazy-installs, then delegates to real ghc
│   └── ghcd             # Wrapper: lazy-installs, then delegates to real ghcd
├── scripts/
│   └── install.sh       # Downloads and installs ghc/ghcd binaries
├── skills/
│   └── ghcd/
│       └── SKILL.md     # Teaches agents to use ghc instead of gh
└── README.md
```

## Requirements

- `gh` (GitHub CLI) must be installed and authenticated
- macOS or Linux (amd64 or arm64)
- `curl` and `tar` available in PATH

## License

[MIT](../LICENSE)
