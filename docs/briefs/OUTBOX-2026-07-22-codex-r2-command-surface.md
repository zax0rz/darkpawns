# fix: restore the C command surface for punctuation and `grats`

Closes DP-1185, DP-1186, and DP-1187.

## Summary

- Mirror `command_interpreter()` tokenization: after leading whitespace, a non-letter first byte is a one-character command and the remaining input is its argument. This makes attached forms such as `'hello`, `:grins`, `.hi`, and `;test` work without a separating space. The same tokenizer is used after player alias expansion.
- Register the C command-table names `'`, `.`, `:`, `;`, `?`, and `grats` with their existing Go handlers and exact C level/position gates.
- Remove the player-typed Go-only `gratz` spelling and its gate-generator rationale. The internal channel identifier remains `gratz`, matching the existing channel implementation.
- Add focused registry, tokenization, and dispatch tests, including both `'hello` and `' hello`.

The behavior follows `src/interpreter.c:430,473,488,648,671,831` and the `command_interpreter()` tokenization at `src/interpreter.c:883-907`, under fidelity rules R2a, R2b, and R4.

## Handler and gate mapping

| Player input | Go path | Minimum position | Minimum level |
|---|---|---:|---:|
| `grats` | `cmdGratz` / channel `gratz` | sleeping | 0 |
| `.` | `cmdReply` | sleeping | 0 |
| `:` | `cmdEmote` (C `do_echo`, `SCMD_EMOTE`) | resting | 1 |
| `;` | `cmdWiznet` | dead | 0 |
| `?` | `cmdHelp` | dead | 0 |
| `'` | `cmdSay` | resting | 0 |

`gratz` is intentionally absent from the registry. In C, only `grats` and the separate toggle `nograts` exist.

## Differential oracle scenarios

Run every scenario against pre-fix `main` first and require RED, then against this change and require GREEN. Use a plain room with no mobs or special procedures unless the scenario says otherwise; normalize only the usual prompt/session noise and compare the command response byte-for-byte.

1. Say shorthand without a space
   - Put a mortal player in a plain room at `POS_RESTING` or higher, with a second player present to capture observer text.
   - Send `'hello`.
   - Compare both the actor's say echo and the observer's room message with C.
   - Pre-fix Go should fail registry lookup because it treats `'hello` as the whole command; post-fix Go should match C `do_say`.

2. Say shorthand with a space
   - Use the same setup and send `' hello`.
   - Compare actor and observer output with C and with the no-space case.
   - Both forms must dispatch to the same say behavior after the fix.

3. Attached emote shorthand
   - Use a level-1 mortal at `POS_RESTING` or higher, with an observer in the same plain room.
   - Send `:grins broadly`.
   - Compare the actor and observer emote messages with C's `do_echo` + `SCMD_EMOTE` behavior.

4. Attached reply shorthand with no pending tell
   - Use a mortal at `POS_SLEEPING` or higher with no reply target/pending tell state.
   - Send `.hi`.
   - Compare the exact no-reply response with C. Do not invent an expected string in the harness; record the C response and require Go to match it.

5. Help shorthand
   - Use a mortal in any position, including `POS_DEAD` if the harness supports it.
   - Send `?`.
   - Compare the complete help response with C.

6. Mortal wiznet punctuation
   - Use a non-immortal player in any position.
   - Send `;test`.
   - Capture whatever C returns after the level-0 command-table lookup and its internal handler checks, then require Go to match. Do not assume the response in advance.

7. Canonical congratulations spelling
   - Use a mortal at `POS_SLEEPING` or higher with channel delivery enabled and a second eligible receiver if needed.
   - Send `grats everyone`.
   - Compare sender and receiver channel output with C.

8. Removed Go-only spelling
   - In the same player state, send `gratz everyone`.
   - Require the exact C unknown-command response (`Huh?!?`) from Go after the fix.

Claude owns authoring/normalizing and running these `dp-oracle-diff` cases; this change does not require `DP_ORACLE_BIN` in the Codex environment.

## Reachability

Generated with:

```text
python3 scripts/gen_reachability.py --out /tmp/reachability-after.tsv
```

Compared with `docs/reports/reachability-2026-07-22.tsv`, exactly six rows changed and every change was to `registered`:

| Command | Before | After |
|---|---|---|
| `'` | `specproc` | `registered` |
| `.` | `implemented-unwired` | `registered` |
| `:` | `implemented-unwired` | `registered` |
| `;` | `implemented-unwired` | `registered` |
| `?` | `implemented-unwired` | `registered` |
| `grats` | `missing` | `registered` |

The `'` speech spec-procedure intercept remains in place; registry lookup now supplies the plain-room fallthrough.

| Status | Before | After | Delta |
|---|---:|---:|---:|
| Total C entries | 508 | 508 | 0 |
| `registered` | 252 | 258 | +6 |
| `implemented-unwired` | 25 | 21 | -4 |
| `missing` | 36 | 35 | -1 |
| `specproc` | 10 | 9 | -1 |
| `social` | 183 | 183 | 0 |
| `abbrev-stub` | 2 | 2 | 0 |

No command regressed and no status outside these six rows changed.

## Verification

- `make fmt` — passed.
- `go build ./...` — passed (with a sandbox-only module stat-cache write warning).
- `go vet ./...` — passed.
- `golangci-lint run ./...` — passed with 0 issues using a writable temporary lint cache.
- Focused command-surface, tokenization, dispatch, and gate-golden tests — passed.
- `go test ./pkg/session/... ./pkg/command/... -skip 'TestWebSocket_NewCharThenLook|TestWritePumpExitTriggersCleanup'` — passed; this runs the requested package suites except the two tests that require opening a local listener.
- Unfiltered `go test ./pkg/session/... ./pkg/command/...` — command package passed, but the sandbox forbids the existing `httptest` listener in `TestWritePumpExitTriggersCleanup` (`listen tcp6 [::1]:0: operation not permitted`).
- `python3 scripts/gen_reachability.py --out /tmp/reachability-after.tsv` — passed; 508 rows and all sanity checks passed.

## Oracle gate

- [ ] Claude runs the differential oracle scenarios above and confirms RED on pre-fix `main`.
- [ ] Claude confirms GREEN after this change.
