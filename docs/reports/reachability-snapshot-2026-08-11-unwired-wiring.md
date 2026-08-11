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

## After: 0 implemented-unwired, 287 registered

```
  (none — all 20 implemented-unwired commands wired)

  Total: 0 implemented-unwired, 287 registered
  Delta: 20 commands wired this PR (267 -> 287 registered)
```

## Wired this PR (20 commands)

| Command | C handler | Go handler | Notes |
|---|---|---|---|
| dig | do_dig | CmdDig (foraging skill) | semantic divergence — see DP-1225 |
| finger | do_whois | cmdWhois | pure alias |
| freeze | do_wizutil SCMD_FREEZE | cmdFreeze (new wrapper) | wizutilDispatch |
| gold | do_coins | cmdCoins | pure alias |
| kabuki | do_hide SCMD_KABUKI | CmdKabuki (new) + DoKabuki | new SkillKabuki const |
| murder | do_hit (SCMD_MURDER) | cmdHit | pure alias — C do_hit ignores subcmd |
| mute | do_wizutil SCMD_SQUELCH | cmdMute (new wrapper) | wizutilDispatch; admin_commands.go mute is dead code (never wired) — see DP-1225 |
| notitle | do_wizutil SCMD_NOTITLE | cmdNotitle (new wrapper) | wizutilDispatch |
| pardon | do_wizutil SCMD_PARDON | cmdPardon (new wrapper) | wizutilDispatch |
| poofin | do_poofset SCMD_POOFIN | cmdPoofin (new wrapper) | wraps cmdPoofset |
| poofout | do_poofset SCMD_POOFOUT | cmdPoofout (new wrapper) | wraps cmdPoofset |
| qecho | do_qcomm SCMD_QECHO | cmdQecho (new) | immortal raw echo, LVL_IMMORT |
| qsay | do_qcomm SCMD_QSAY | cmdQsay (new) | quest-say wording |
| rsay | do_race_say | cmdRaceSay | pure alias (delegates to game.ExecRaceSay) |
| shadow | do_follow (subcmd=TRUE) | cmdShadow (new wrapper) | quiet follow; SKILL_SHADOW affect TODO |
| socials | do_commands SCMD_SOCIALS | cmdSocials (new) | lists game.Socials |
| thaw | do_wizutil SCMD_THAW | cmdThaw (new wrapper) | wizutilDispatch |
| uptime | do_date SCMD_UPTIME | cmdUptime (new) | C day(s)/H:MM format, boot time |
| wizhelp | do_commands SCMD_WIZHELP | cmdWizhelp (new) | lists LVL_IMMORT+ commands |
| wnewbie | do_newbie | cmdNewbie | already existed as 'newbiegive'; now also C name |

## Open fidelity items (DP-1225)

- **dig** — Go CmdDig is a foraging skill; the C OLC exit-creation do_dig (new_cmds2.c) is unported.
- **mute** — resolved: the C SCMD_SQUELCH behavior is now registered under `mute` (the game's command). The duration-based admin `mute` in pkg/command/admin_commands.go is dead code (AdminCommands is never instantiated/wired at startup), so there is no runtime collision.
