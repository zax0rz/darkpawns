# Reachability snapshot — implemented-unwired wiring (2026-08-11)

## Before: 20 implemented-unwired, 267 registered

```
  dig        (do_dig/0)
  finger     (do_whois/0)
  freeze     (do_wizutil/SCMD_FREEZE)
  gold       (do_coins/0)
  kabuki     (do_hide/SCMD_KABUKI)
  murder     (do_hit/SCMD_MURDER)
  mute       (do_wizutil/SCMD_SQUELCH)
  notitle    (do_wizutil/SCMD_NOTITLE)
  pardon     (do_wizutil/SCMD_PARDON)
  poofin     (do_poofset/SCMD_POOFIN)
  poofout    (do_poofset/SCMD_POOFOUT)
  qecho      (do_qcomm/SCMD_QECHO)
  qsay       (do_qcomm/SCMD_QSAY)
  rsay       (do_race_say/0)
  shadow     (do_follow/TRUE)
  socials    (do_commands/SCMD_SOCIALS)
  thaw       (do_wizutil/SCMD_THAW)
  uptime     (do_date/SCMD_UPTIME)
  wizhelp    (do_commands/SCMD_WIZHELP)
  wnewbie    (do_newbie/0)

  Total: 20 implemented-unwired, 267 registered
```

## After: 1 implemented-unwired, 286 registered

```
  dig        (do_dig/0)  [fidelity gap: Go handler ≠ C behavior — DP-1225]

  Total: 1 implemented-unwired, 286 registered
  Delta: 19 commands wired this PR (267 -> 286 registered)
```

## Wired this PR (19 commands)

Each wired command was verified against its C source before registration. See the PR description for the full per-command breakdown (handler, approach, fidelity notes).

## Remaining implemented-unwired (1) — fidelity gap, not a wiring gap

- **dig** — C `do_dig` (src/new_cmds2.c:818) is a `LVL_BUILDER` OLC exit-creator (`dig <dir> <room#>`). The Go `CmdDig` is an unrelated mortal foraging skill. The two are different features; wiring the foraging handler under the C name (plus the C builder gate) would serve neither audience — builders would get foraging, and mortals couldn't reach the foraging skill at all (locked at level 31). The C OLC dig is unported; tracked in DP-1225. Per the PR objective's own step ("verify it matches C's behavior"), `dig` fails verification, so it is deliberately left unwired rather than shipped broken.
