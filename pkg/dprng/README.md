# Dark Pawns random stream

`dprng` is the only production pseudo-random stream. It ports `src/random.c`
and the `number()` / `dice()` conversions in `src/utils.c`, including their
`float32` rounding and draw counts.

The process-wide stream is mutex-protected, so every operation is atomic. C is
single-threaded and Go is not, however, so a mutex cannot make scheduling order
across concurrent goroutines deterministic. Oracle scenarios that exercise RNG
must use one actor and serialize all draws.

Most consumers correspond directly to C `number()` or `dice()` sites. The
following Go-only consumers also use this stream so they cannot introduce a
second generator:

- `pkg/agentcli/reconnect.go`: reconnect jitter
- `pkg/engine/affect.go`: generated affect IDs
- `pkg/engine/skill.go`: the temporary Go skill-progression model
- `pkg/session/session_temp.go`: the session `RandomInt` helper

The import guard in `internal/lintguard` rejects production imports of
`math/rand` and `math/rand/v2` outside this package.
