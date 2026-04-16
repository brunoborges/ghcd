# ghx — Agentic CLI Plugin

A plugin for [Claude Code](https://code.claude.com/docs/en/plugins) and [GitHub Copilot CLI](https://docs.github.com/en/copilot/concepts/agents/copilot-cli/about-cli-plugins) that installs and configures [ghxd](https://github.com/brunoborges/ghx) — a caching proxy for the GitHub CLI (`gh`).

Compatible with any agentic CLI runner that supports the [Claude Code Plugin](https://code.claude.com/docs/en/plugins) format.

## What it does

When enabled, this plugin:

1. **Installs `ghx` and `ghxd`** automatically on first use (lazy install)
2. **Adds `ghx` to PATH** so agents can use it as a drop-in replacement for `gh`
3. **Teaches Claude and Copilot to prefer `ghx`** over `gh` via a built-in skill, so all GitHub CLI calls go through the caching proxy

This eliminates redundant API calls, prevents rate limiting, and dramatically speeds up repeated `gh` commands in agentic workflows.

## Install

### Claude Code & GitHub Copilot CLI

```bash
# Add the marketplace (one-time)
/plugin marketplace add brunoborges/agent-plugins

# Install the plugin
/plugin install ghx@agent-plugins
```

### Local development / testing

```bash
claude --plugin-dir ./agent-plugin
```

## How it works

### Lazy binary installation

The plugin ships wrapper scripts in `bin/` that are automatically added to PATH. On first invocation, the wrapper downloads and installs the real `ghx` and `ghxd` binaries to the plugin's persistent data directory (`${CLAUDE_PLUGIN_DATA}/bin`).

### Skill: automatic `ghx` preference

The plugin includes a skill that instructs Claude to use `ghx` instead of `gh` for all GitHub CLI commands. Claude and Copilot CLI load this skill automatically when relevant — no manual invocation needed.

You can also invoke it explicitly:

```
/ghx:ghx
```

### Cache behavior

```
First call:   ghx pr list ...   → ~1.1s (cache miss, calls gh)
Second call:  ghx pr list ...   → ~0.01s (cache hit, instant)
After TTL:    ghx pr list ...   → ~1.0s (TTL expired, fresh call)
```

## Plugin structure

```
agent-plugin/
├── .claude-plugin/
│   └── plugin.json      # Plugin manifest
├── bin/
│   ├── ghx              # Wrapper: lazy-installs, then delegates to real ghx
│   └── ghxd             # Wrapper: lazy-installs, then delegates to real ghxd
├── scripts/
│   └── install.sh       # Downloads and installs ghx/ghxd binaries
├── skills/
│   └── ghxd/
│       └── SKILL.md     # Teaches agents to use ghx instead of gh
└── README.md
```

## Requirements

- macOS or Windows/Linux (amd64 or arm64)
- `curl` and `tar` available in PATH
- `gh` (GitHub CLI) is **optional** — if not installed, `ghx` auto-downloads it on first use, and deploys a shim `gh` command. However, `gh auth login` must be run before using any authenticated commands.

## Learn more

- [Claude Code Plugins documentation](https://code.claude.com/docs/en/plugins)
- [GitHub Copilot CLI Plugins documentation](https://docs.github.com/en/copilot/concepts/agents/copilot-cli/about-cli-plugins)
- [ghx project](https://github.com/brunoborges/ghx)

## License

[MIT](../LICENSE)
