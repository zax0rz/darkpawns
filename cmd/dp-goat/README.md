# Dp Goat CLI

Dark Pawns is a MUD (Multi-User Dungeon) with 10,057 rooms, 1,313 mobs,
and 854 objects. Commands are sent as JSON over a WebSocket connection.
This spec describes each game command as an endpoint for CLI generation.
The actual transport is WebSocket, not HTTP — the generated CLI will need
transport patches.

Printed by [@zax0rz](https://github.com/zax0rz).

## Install

The recommended path installs both the `dp-goat-pp-cli` binary and the `pp-dp-goat` agent skill (Claude Code, Codex, Cursor, Gemini CLI, GitHub Copilot, and other agents supported by the upstream [`skills`](https://github.com/vercel-labs/skills) CLI) in one shot:

```bash
npx -y @mvanhorn/printing-press install dp-goat
```

For CLI only (no skill):

```bash
npx -y @mvanhorn/printing-press install dp-goat --cli-only
```

For skill only — installs the skill into the same agents as the default command above, but skips the CLI binary (use this to update or reinstall just the skill):

```bash
npx -y @mvanhorn/printing-press install dp-goat --skill-only
```

To constrain the skill install to one or more specific agents (repeatable — agent names match the [`skills`](https://github.com/vercel-labs/skills) CLI):

```bash
npx -y @mvanhorn/printing-press install dp-goat --agent claude-code
npx -y @mvanhorn/printing-press install dp-goat --agent claude-code --agent codex
```

### Without Node

The generated install path is category-agnostic until this CLI is published. If `npx` is not available before publish, install Node or use the category-specific Go fallback from the public-library entry after publish.

### Pre-built binary

Download a pre-built binary for your platform from the [latest release](https://github.com/mvanhorn/printing-press-library/releases/tag/dp-goat-current). On macOS, clear the Gatekeeper quarantine: `xattr -d com.apple.quarantine <binary>`. On Unix, mark it executable: `chmod +x <binary>`.

<!-- pp-hermes-install-anchor -->
## Install for Hermes

From the Hermes CLI:

```bash
hermes skills install mvanhorn/printing-press-library/cli-skills/pp-dp-goat --force
```

Inside a Hermes chat session:

```bash
/skills install mvanhorn/printing-press-library/cli-skills/pp-dp-goat --force
```

## Install for OpenClaw

Tell your OpenClaw agent (copy this):

```
Install the pp-dp-goat skill from https://github.com/mvanhorn/printing-press-library/tree/main/cli-skills/pp-dp-goat. The skill defines how its required CLI can be installed.
```

## Use with Claude Desktop

This CLI ships an [MCPB](https://github.com/modelcontextprotocol/mcpb) bundle — Claude Desktop's standard format for one-click MCP extension installs (no JSON config required).

To install:

1. Download the `.mcpb` for your platform from the [latest release](https://github.com/mvanhorn/printing-press-library/releases/tag/dp-goat-current).
2. Double-click the `.mcpb` file. Claude Desktop opens and walks you through the install.

Requires Claude Desktop 1.0.0 or later. Pre-built bundles ship for macOS Apple Silicon (`darwin-arm64`) and Windows (`amd64`, `arm64`); for other platforms, use the manual config below.

<details>
<summary>Manual JSON config (advanced)</summary>

If you can't use the MCPB bundle (older Claude Desktop, unsupported platform), install the MCP binary and configure it manually.


Install the MCP binary from this CLI's published public-library entry or pre-built release.

Add to your Claude Desktop config (`~/Library/Application Support/Claude/claude_desktop_config.json`):

```json
{
  "mcpServers": {
    "dp-goat": {
      "command": "dp-goat-pp-mcp"
    }
  }
}
```

</details>

## Quick Start

### 1. Install

See [Install](#install) above.

### 2. Verify Setup

```bash
dp-goat-pp-cli doctor
```

This checks your configuration.

### 3. Try Your First Command

```bash
dp-goat-pp-cli get --item example-value
```

## Usage

Run `dp-goat-pp-cli --help` for the full command reference and flag list.

## Commands

### cast

Manage cast

- **`dp-goat-pp-cli cast`** - Cast a spell

### consider

Manage consider

- **`dp-goat-pp-cli consider`** - Assess combat difficulty of a target

### down

Manage down

- **`dp-goat-pp-cli down`** - Move down

### drink

Manage drink

- **`dp-goat-pp-cli drink`** - Drink from a container

### drop

Manage drop

- **`dp-goat-pp-cli drop`** - Drop an item

### east

Manage east

- **`dp-goat-pp-cli east`** - Move east

### eat

Manage eat

- **`dp-goat-pp-cli eat`** - Eat food

### flee

Manage flee

- **`dp-goat-pp-cli flee`** - Attempt to flee from combat. Costs some experience points.
May fail if the mob is much faster than you.

### get

Manage get

- **`dp-goat-pp-cli get`** - Pick up an item from the room or from a container.
Use 'get all' to pick up everything.

### give

Manage give

- **`dp-goat-pp-cli give`** - Give an item to someone

### inventory

Manage inventory

- **`dp-goat-pp-cli inventory`** - List carried items

### kill

Manage kill

- **`dp-goat-pp-cli kill`** - Initiate combat with a target. Requires a weapon or bare hands.
Use 'consider' first to assess difficulty.

### look

Manage look

- **`dp-goat-pp-cli look`** - Study your surroundings. Without arguments, shows the room.
With a target, examines that target specifically.

### north

Manage north

- **`dp-goat-pp-cli north`** - Move north

### say

Manage say

- **`dp-goat-pp-cli say`** - Say something to everyone in the room. All players and mobs
in the room will see your message.

### score

Manage score

- **`dp-goat-pp-cli score`** - Show character stats

### south

Manage south

- **`dp-goat-pp-cli south`** - Move south

### tell

Manage tell

- **`dp-goat-pp-cli tell`** - Send a private message to another player. Only the target
player will see the message, regardless of location.

### up

Manage up

- **`dp-goat-pp-cli up`** - Move up

### wear

Manage wear

- **`dp-goat-pp-cli wear`** - Equip armor or clothing

### west

Manage west

- **`dp-goat-pp-cli west`** - Move west

### who

Manage who

- **`dp-goat-pp-cli who`** - Show who is online

### wield

Manage wield

- **`dp-goat-pp-cli wield`** - Equip a weapon

### yell

Manage yell

- **`dp-goat-pp-cli yell`** - Shout to the entire zone


## Output Formats

```bash
# Human-readable table (default in terminal, JSON when piped)
dp-goat-pp-cli get --item example-value

# JSON for scripting and agents
dp-goat-pp-cli get --item example-value --json

# Filter to specific fields
dp-goat-pp-cli get --item example-value --json --select id,name,status

# Dry run — show the request without sending
dp-goat-pp-cli get --item example-value --dry-run

# Agent mode — JSON + compact + no prompts in one flag
dp-goat-pp-cli get --item example-value --agent
```

## Agent Usage

This CLI is designed for AI agent consumption:

- **Non-interactive** - never prompts, every input is a flag
- **Pipeable** - `--json` output to stdout, errors to stderr
- **Filterable** - `--select id,name` returns only fields you need
- **Previewable** - `--dry-run` shows the request without sending
- **Explicit retries** - add `--idempotent` to create retries when a no-op success is acceptable
- **Confirmable** - `--yes` for explicit confirmation of destructive actions
- **Piped input** - write commands can accept structured input when their help lists `--stdin`
- **Agent-safe by default** - no colors or formatting unless `--human-friendly` is set

Exit codes: `0` success, `2` usage error, `3` not found, `5` API error, `7` rate limited, `10` config error.

## Health Check

```bash
dp-goat-pp-cli doctor
```

Verifies configuration and connectivity to the API.

## Configuration

Config file: `~/.config/dark-pawns-mud-pp-cli/config.toml`

Static request headers can be configured under `headers`; per-command header overrides take precedence.

## Troubleshooting
**Not found errors (exit code 3)**
- Check the resource ID is correct
- Run the `list` command to see available items

---

Generated by [CLI Printing Press](https://github.com/mvanhorn/cli-printing-press)
