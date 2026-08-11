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
  mute       (do_wizutil/SCMD_SQUELCH)  [DP-1225: collision with admin mute]

  Total: 1 implemented-unwired, 286 registered
  Delta: 19 commands wired this PR (267 -> 286 registered)
```
