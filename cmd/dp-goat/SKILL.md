---
name: pp-dp-goat
description: "Printing Press CLI for Dp Goat. Dark Pawns is a MUD (Multi-User Dungeon) with 10,057 rooms, 1,313 mobs, and 854 objects. Commands are sent as JSON..."
author: "user"
license: "Apache-2.0"
argument-hint: "<command> [args] | install cli|mcp"
allowed-tools: "Read Bash"
metadata:
  openclaw:
    requires:
      bins:
        - dp-goat-pp-cli
---

# Dp Goat — Printing Press CLI

## Prerequisites: Install the CLI

This skill drives the `dp-goat-pp-cli` binary. **You must verify the CLI is installed before invoking any command from this skill.** If it is missing, install it first:

1. Install via the Printing Press installer:
   ```bash
   npx -y @mvanhorn/printing-press install dp-goat --cli-only
   ```
2. Verify: `dp-goat-pp-cli --version`
3. Ensure `$GOPATH/bin` (or `$HOME/go/bin`) is on `$PATH`.

If the `npx` install fails before this CLI has a public-library category, install Node or use the category-specific Go fallback after publish.

If `--version` reports "command not found" after install, the install step did not put the binary on `$PATH`. Do not proceed with skill commands until verification succeeds.

Dark Pawns is a MUD (Multi-User Dungeon) with 10,057 rooms, 1,313 mobs,
and 854 objects. Commands are sent as JSON over a WebSocket connection.
This spec describes each game command as an endpoint for CLI generation.
The actual transport is WebSocket, not HTTP — the generated CLI will need
transport patches.

## Command Reference

**cast** — Manage cast

- `dp-goat-pp-cli cast` — Cast a spell

**consider** — Manage consider

- `dp-goat-pp-cli consider` — Assess combat difficulty of a target

**down** — Manage down

- `dp-goat-pp-cli down` — Move down

**drink** — Manage drink

- `dp-goat-pp-cli drink` — Drink from a container

**drop** — Manage drop

- `dp-goat-pp-cli drop` — Drop an item

**east** — Manage east

- `dp-goat-pp-cli east` — Move east

**eat** — Manage eat

- `dp-goat-pp-cli eat` — Eat food

**flee** — Manage flee

- `dp-goat-pp-cli flee` — Attempt to flee from combat. Costs some experience points. May fail if the mob is much faster than you.

**get** — Manage get

- `dp-goat-pp-cli get` — Pick up an item from the room or from a container. Use 'get all' to pick up everything.

**give** — Manage give

- `dp-goat-pp-cli give` — Give an item to someone

**inventory** — Manage inventory

- `dp-goat-pp-cli inventory` — List carried items

**kill** — Manage kill

- `dp-goat-pp-cli kill` — Initiate combat with a target. Requires a weapon or bare hands. Use 'consider' first to assess difficulty.

**look** — Manage look

- `dp-goat-pp-cli look` — Study your surroundings. Without arguments, shows the room. With a target, examines that target specifically.

**north** — Manage north

- `dp-goat-pp-cli north` — Move north

**say** — Manage say

- `dp-goat-pp-cli say` — Say something to everyone in the room. All players and mobs in the room will see your message.

**score** — Manage score

- `dp-goat-pp-cli score` — Show character stats

**south** — Manage south

- `dp-goat-pp-cli south` — Move south

**tell** — Manage tell

- `dp-goat-pp-cli tell` — Send a private message to another player. Only the target player will see the message, regardless of location.

**up** — Manage up

- `dp-goat-pp-cli up` — Move up

**wear** — Manage wear

- `dp-goat-pp-cli wear` — Equip armor or clothing

**west** — Manage west

- `dp-goat-pp-cli west` — Move west

**who** — Manage who

- `dp-goat-pp-cli who` — Show who is online

**wield** — Manage wield

- `dp-goat-pp-cli wield` — Equip a weapon

**yell** — Manage yell

- `dp-goat-pp-cli yell` — Shout to the entire zone


### Finding the right command

When you know what you want to do but not which command does it, ask the CLI directly:

```bash
dp-goat-pp-cli which "<capability in your own words>"
```

`which` resolves a natural-language capability query to the best matching command from this CLI's curated feature index. Exit code `0` means at least one match; exit code `2` means no confident match — fall back to `--help` or use a narrower query.

## Auth Setup

No authentication required.

Run `dp-goat-pp-cli doctor` to verify setup.

## Agent Mode

Add `--agent` to any command. Expands to: `--json --compact --no-input --no-color --yes`.

- **Pipeable** — JSON on stdout, errors on stderr
- **Filterable** — `--select` keeps a subset of fields. Dotted paths descend into nested structures; arrays traverse element-wise. Critical for keeping context small on verbose APIs:

  ```bash
  dp-goat-pp-cli get --item example-value --agent --select id,name,status
  ```
- **Previewable** — `--dry-run` shows the request without sending
- **Non-interactive** — never prompts, every input is a flag
- **Explicit retries** — use `--idempotent` only when an already-existing create should count as success

## Agent Feedback

When you (or the agent) notice something off about this CLI, record it:

```
dp-goat-pp-cli feedback "the --since flag is inclusive but docs say exclusive"
dp-goat-pp-cli feedback --stdin < notes.txt
dp-goat-pp-cli feedback list --json --limit 10
```

Entries are stored locally at `~/.dp-goat-pp-cli/feedback.jsonl`. They are never POSTed unless `DP_GOAT_FEEDBACK_ENDPOINT` is set AND either `--send` is passed or `DP_GOAT_FEEDBACK_AUTO_SEND=true`. Default behavior is local-only.

Write what *surprised* you, not a bug report. Short, specific, one line: that is the part that compounds.

## Output Delivery

Every command accepts `--deliver <sink>`. The output goes to the named sink in addition to (or instead of) stdout, so agents can route command results without hand-piping. Three sinks are supported:

| Sink | Effect |
|------|--------|
| `stdout` | Default; write to stdout only |
| `file:<path>` | Atomically write output to `<path>` (tmp + rename) |
| `webhook:<url>` | POST the output body to the URL (`application/json` or `application/x-ndjson` when `--compact`) |

Unknown schemes are refused with a structured error naming the supported set. Webhook failures return non-zero and log the URL + HTTP status on stderr.

## Named Profiles

A profile is a saved set of flag values, reused across invocations. Use it when a scheduled agent calls the same command every run with the same configuration - HeyGen's "Beacon" pattern.

```
dp-goat-pp-cli profile save briefing --json
dp-goat-pp-cli --profile briefing get --item example-value
dp-goat-pp-cli profile list --json
dp-goat-pp-cli profile show briefing
dp-goat-pp-cli profile delete briefing --yes
```

Explicit flags always win over profile values; profile values win over defaults. `agent-context` lists all available profiles under `available_profiles` so introspecting agents discover them at runtime.

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 2 | Usage error (wrong arguments) |
| 3 | Resource not found |
| 5 | API error (upstream issue) |
| 7 | Rate limited (wait and retry) |
| 10 | Config error |

## Argument Parsing

Parse `$ARGUMENTS`:

1. **Empty, `help`, or `--help`** → show `dp-goat-pp-cli --help` output
2. **Starts with `install`** → ends with `mcp` → MCP installation; otherwise → see Prerequisites above
3. **Anything else** → Direct Use (execute as CLI command with `--agent`)

## MCP Server Installation

Install the MCP binary from this CLI's published public-library entry or pre-built release, then register it:

```bash
claude mcp add dp-goat-pp-mcp -- dp-goat-pp-mcp
```

Verify: `claude mcp list`

## Direct Use

1. Check if installed: `which dp-goat-pp-cli`
   If not found, offer to install (see Prerequisites at the top of this skill).
2. Match the user query to the best command from the Unique Capabilities and Command Reference above.
3. Execute with the `--agent` flag:
   ```bash
   dp-goat-pp-cli <command> [subcommand] [args] --agent
   ```
4. If ambiguous, drill into subcommand help: `dp-goat-pp-cli <command> --help`.
