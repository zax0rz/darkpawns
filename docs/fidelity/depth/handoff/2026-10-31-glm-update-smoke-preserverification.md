# Follow-up after the release: pre-serveControl launcher deaths write no exit record (2026-10-31, GLM)

From #2262's reconstruction of the windows update-smoke flake (run
41151721701): a candidate update launcher that fails BEFORE reaching
`serveControl` — `run()` in internal/portable/run.go — writes no
`.last-exit.json` receipt, because the receipt writer is `closeControl`,
deferred after `serveControl` is established. `startUpdateCandidate`'s
poll therefore never sees a new run id and burns its full 40-second
readiness deadline before failing with "new release did not become
ready", with no cause attached. That is 40 silent seconds on every
pre-start candidate failure — verification, lock, or load — on every
platform, and it cost the CI diagnosis real evidence.

#2262's pre-spawn retry removes the one observed trigger (a transient
open of a just-extracted executable under a real-time scanner), and its
first-attempt log line names that class when it recurs. The structural
gap remains: write an exit record (or an equivalent early failure
record) as soon as the launcher owns the run lock, so a pre-serveControl
death names its cause and the updater's deadline error carries it. Not
in #2262 — it changes the receipt contract and belongs to its own
reviewed change after the release.
