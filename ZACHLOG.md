# Zachlog — human-gated items

Things that need Zach's hands or decision (I can't/shouldn't do them myself).

## Open

- **Verify prod actually connects to the DB.** The server reads its DSN only
  from the `-db` flag (it ignores `DATABASE_URL`), and a wrong/missing DSN
  fails *silently* into no-persistence mode. Confirm `dark-pawns.service`'s
  `ExecStart` passes `-db "…"`, and check the logs say `Database connected.`
  rather than `continuing without persistence`. Details + verify command added
  to `docs/operational/DEPLOYMENT.md` ("Database connection"). _(2026-06-17.)_

- **Review + merge the QA branch.** Branch `qa/boot-telnet-combat-fixes` holds
  5 commits (DP-589 boot, DP-590 mob-AI deadlock, telnet login/input/crash/
  render, e2e smoke suite, this log). Built on `main`; review and fast-forward
  or PR when ready. _(2026-06-17.)_

- **DEPLOYMENT.md is uncommitted.** It had pre-existing edits from before this
  session (not mine), so the new "Database connection" section sits on top of
  those, left uncommitted — review and commit the whole file together.
  _(2026-06-17.)_

## Done

_(none yet)_
