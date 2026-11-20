# Mutation loop decisions

Open questions that do not stop the measurement loop. No code changes have
been made for these rows.

| Round | Sample row | Class | Evidence | Follow-up |
|---|---:|---|---|---|
| 1 | 28 | `go-only-in-game` | `CmdListSkills` emits the sampled header at `pkg/command/skill_commands.go:2262`; C's `do_skills` is not in the C command table and emits a different class report. | DP-1596 follow-up: decide whether the Go skill-list command is an intended addition or should be removed. |
| 1 | 34 | `go-only-in-game` | `characterCompare` emits `looks better than`; C `do_compare` emits only the qualified comparison variants in `src/new_cmds.c:2079-2091`. | DP-1596 follow-up: decide whether this output divergence is intended or should be brought back to C. |

