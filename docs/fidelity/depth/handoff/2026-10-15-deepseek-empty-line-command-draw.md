# DeepSeek — a bare RETURN is a command (empty-line draw parity)

Date: 2026-10-15 · Branch: `deepseek/draw-parity-empty-line` (off `origin/main`
`f1bf62126`) · Rules: R1, R3a, R3d, R5e, R5g, R5h

## The finding

`command_interpreter` draws before it decides the line is empty
(`src/interpreter.c:889-895`):

```c
void command_interpreter(struct char_data *ch, char *argument)
{
  ...
  if (!number(0,3))
    REMOVE_BIT_AR(AFF_FLAGS(ch), AFF_HIDE);

  /* just drop to next line for hitting CR */
  skip_spaces(&argument);
  if (!*argument)
    return;
```

`comm.c` hands it *every* dequeued line at `CON_PLAYING`, including a bare
RETURN — `write_to_q` queues an empty line and `get_from_q` returns it
(`src/comm.c:2008`, `:1232-1249`), then `command_interpreter(d->character, comm)`
(`src/comm.c:628`). Only the three earlier arms divert a line: `d->str`
(`:615-616`), `d->showstr_count` (`:617-618`) and `d->connected != CON_PLAYING`
(`:619-620`).

Go returned before the draw. `pkg/session/terminal.go`'s empty-line arm only
refreshed the prompt, and `executeCommandRaw`'s `if cmdStr == "" { return nil }`
sat *above* the `commandNumber(0, 3)` call, so a bare RETURN, a whitespace-only
line, and an alias whose replacement is empty all burned zero draws where C
burns one.

## The fix

- `pkg/session/terminal.go`: the `case line == "":` arm now routes the line
  through `terminalCommand("", nil, rawLine)` — the same path every other line
  takes, so the wait-state queue, the idle reset and the dequeue point match C's
  — and then refreshes the prompt.
- `pkg/session/commands.go`: the leading `commandNumber(0, 3)` draw moved above
  the empty-command return (still after alias expansion, which C runs first at
  `src/comm.c:625-627`), and the alias branch no longer returns early when its
  expansion resolves to an empty command. The empty return now sits where C's
  `if (!*argument) return;` does.

## Proof (R5h)

`pkg/session/command_empty_line_draw_test.go` — three tests, each verified
failing on the reverted production shape (copy-based revert, no `git stash`):

| Test | Asserts | Reverted shape |
|---|---|---|
| `TestExecuteCommandEmptyLineConsumesLeadingDraw` (bare return, whitespace only) | exactly one `Number(0,3)` and a stream advanced by one | fails: `draws = [], want [{from:0 to:3}]` |
| `TestTerminalLineBareReturnConsumesLeadingDraw` | the transport arm reaches the interpreter | fails: `bare RETURN command draws = []` |
| `TestTerminalLineBareReturnClearsHide` | seed 5's zero roll clears `AFF_HIDE` | fails: `AFF_HIDE survived a bare RETURN` |

## Rows cleared (seed-1 draw-parity corpus, `finish-baseline-draw-parity`)

| Scenario | Before (baseline `results.tsv` / serial repro) | After |
|---|---|---|
| `prompt-playing-depth` | `stream=4058 c=Number(0,3) = 0 go=(log ends) c-step=probe step 3: go-step=after the last recorded step` | `draw-parity MATCH entries=4060 stream=4060` |
| `prompt-infobar-status-depth` | `stream=4081 c=Number(0,3) = 0 go=(log ends) c-step=probe step 26: inactive` | `draw-parity MATCH entries=4097 stream=4097` |
| `socials-command-depth` | `stream=4134 c=Number(0,3) = 2 go=(log ends) c-step=probe step 12:` | `draw-parity MATCH entries=4137 stream=4137` |

All three keep `result: no normalized divergence`.

The per-step draw boundaries made the class unambiguous. `prompt-playing-depth`
(probe `<ENTER> / afk / <ENTER> / afk`):

```
cbound setup 4056   probe 1 (RETURN) 4057  probe 2 (afk) 4058  probe 3 (RETURN) 4059  probe 4 (afk) 4060
gobound setup 4056  probe 1 (RETURN) 4056  probe 2 (afk) 4057  probe 3 (RETURN) 4057  probe 4 (afk) 4058
```

C advances by one at *every* step; Go only at the non-empty ones. The reported
first mismatch lands on the last empty line rather than the first because both
sides' entry *i* carries stream value *i*, so the one-draw offset stays invisible
until Go's log ends.

## Coverage registration

No new depth row. The player-side leading draw was never registered as one — the
existing sibling is the mobile path, `docs/fidelity/depth/lua.tsv:23`
(`command_interpreter (npc)` → `lua.npc-command-hide-draw`, "Every line a mobile
runs draws number(0,3) for AFF_HIDE first, known command or not (R3)",
`src/interpreter.c:889-890`), whose port is `pkg/game/npc_command.go`. This change
is the same rule on the player path, and its proof lives in the three oracle rows
above plus `pkg/session/command_empty_line_draw_test.go`.

## Gates

See the PR body. `make fmt`, `go build ./...`, `go vet ./...`, `go test ./...`,
`go test ./pkg/game/...`, `golangci-lint run ./...`, `make deadcode-ratchet`,
`make fidelity-depth`, `make fidelity-units`, `make string-census`,
`gosec -severity=high -quiet ./...`, `go test -race ./pkg/game/...`.

Corpus note: no census was launched from this branch. A batched census
(`dp-1427-v2-combined-draw`, Claude) was running on this host for the whole
session, so `scripts/census.sh start` would have refused; the evidence above is
the same harness run serially per scenario, and the corpus-level confirmation
belongs to the next batch.
