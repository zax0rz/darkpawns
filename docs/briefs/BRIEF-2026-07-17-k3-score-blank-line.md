# BRIEF (kimi k3) — `score` extra blank line (character-view red)

**Owner:** kimi k3. **Gate:** Claude runs `character-view` red→green (workers have no `DP_ORACLE_BIN`). **Branch off `main`, one PR.**

> **SCOPE GUARDRAILS.** This is a ONE-LINE fix. Change exactly the one string literal below in `pkg/session/cmd_info.go` and nothing else. Do NOT touch other score fields, other commands, or reformat the file. Do NOT "improve" adjacent lines.

## The bug
The `score` command emits an **extra blank line** after the "You are a <Race> <Class>." line, before "Your pack is …". C's `do_score` does not.

- **Go** (`pkg/session/cmd_info.go:220`):
  ```go
  fmt.Fprintf(&buf, "You are %s %s %s.\r\n\r\n", articleFor(raceName), raceName, className)
  ```
  The trailing `\r\n\r\n` is a **double** line break — that's the stray blank line.
- **C** (`src/act.informative.c:1294-1298`): the race/class line ends with a single break (`sprintf(buf, "%s.\n\r", buf);`), immediately followed by `strcat(buf, "Your pack is light.\r\n");` — **no blank line between them**.

## The fix
Change the format string on `cmd_info.go:220` from `"You are %s %s %s.\r\n\r\n"` to `"You are %s %s %s.\r\n"` (drop one `\r\n`). That's the whole change.

## Acceptance (Claude-gated)
- `--scenario character-view` → `no normalized divergence`.
- Full sweep (correct `result:`-line check) shows no regression; `hunger-thirst` stays red only on its *separate* HP issue (a different bug, not in scope here).
- `score` output otherwise byte-identical to before.
