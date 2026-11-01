# Reassessment of all 57 original A records

15 retain A; 42 move to uncovered. Prefix-proof adds one A from former C; see register.jsonl for its claiming check. All source citations remain in the JSONL audit and register.

| Record | V2 disposition | Claiming check or discovery | Assessment |
| --- | --- | --- | --- |
| pf-003 | uncovered | Fable/live-call-path review of MakeHit, July 3 | TestMakeHit_Basic and related tests claim isolated MakeHit behavior, not live caller/wiring; do not convert that narrower valid claim into an end-to-end claim. |
| pf-005 | uncovered | Later flee depth pilot (PR #601), Aug 23 | No named check claiming the missing behavior identified in the reviewed sources |
| saved-player | uncovered | Human Mudlet playtest and lifecycle-store investigation, Sept 23 | No named check claiming the missing behavior identified in the reviewed sources |
| entry-room | uncovered | Human Mudlet playtest / lifecycle re-login probes, Sept 23 | No named check claiming the missing behavior identified in the reviewed sources |
| idle-clock | uncovered | Human Mudlet playtest and idle-clock source audit, Sept 23 | No named check claiming the missing behavior identified in the reviewed sources |
| logout-menu | core | quit.safe-logout / quit-safe-depth | named claiming check established |
| flag-overlap | uncovered | Persistence/flag audit while fixing DP-1314 (PR #1586) | No named check claiming the missing behavior identified in the reviewed sources |
| flag-two-stores | uncovered | Persistence/lycanthropy write-read audit (PR #1586) | No named check claiming the missing behavior identified in the reviewed sources |
| combat-death | core | combat-death | named claiming check established |
| browser-renderer | uncovered | Owner browser play and browser-client census (PR #1606) | No named check claiming the missing behavior identified in the reviewed sources |
| agent-group | uncovered | Agent-surface removal/source audit (PR #1611) | No named check claiming the missing behavior identified in the reviewed sources |
| damage-gates | core | groinrip.shared-damage-gates (delegated to combat-entry) | named claiming check established |
| level-heal | uncovered | Later groinrip sleeping-victim depth probe above level 10 (DP-1329) | No named check claiming the missing behavior identified in the reviewed sources |
| victim-death-output | uncovered | Later victim-side death probe after DP-1329 fix | No named check claiming the missing behavior identified in the reviewed sources |
| reconnect-identity | uncovered | perform_dupe_check call-path audit / new reconnect journey (DP-1325) | No named check claiming the missing behavior identified in the reviewed sources |
| reconnect-echo | uncovered | New reconnect journey and raw telnet echo inspection (DP-1325) | No named check claiming the missing behavior identified in the reviewed sources |
| lua-attachments | uncovered | DP-1333 world/script attachment inventory | No named check claiming the missing behavior identified in the reviewed sources |
| mob-flags | uncovered | Lua binding flag probes (PR #1641) | No named check claiming the missing behavior identified in the reviewed sources |
| object-flags | uncovered | Lua binding flag probes (PR #1641) | No named check claiming the missing behavior identified in the reviewed sources |
| look-space | uncovered | Lua object-binding probe and do_description class audit | No named check claiming the missing behavior identified in the reviewed sources |
| seated-spell-proof | core | lua-bind-actions / lua.bind-spell | named claiming check established |
| skill-storage | uncovered | Skill catalog/storage-key audit and new God skill-key vehicle (DP-1342) | No named check claiming the missing behavior identified in the reviewed sources |
| ws-name-hold | uncovered | Other-readers review and live pre-login WebSocket disconnect check (#1748) | No named check claiming the missing behavior identified in the reviewed sources |
| mob-wear-slot | uncovered | Reader audit of zone-armed mobile equipment (#1764/#1765) | No named check claiming the missing behavior identified in the reviewed sources |
| first-byte | core | wizard-zreset-depth@2 / zreset.world | named claiming check established |
| foreign-level | uncovered | Oct 4 human playtest, then foreign-level constant sweep (#1775) | No named check claiming the missing behavior identified in the reviewed sources |
| followers-death | uncovered | Oct 4 human playtest (DP-1383) | No named check claiming the missing behavior identified in the reviewed sources |
| idle-timeout | uncovered | Oct 4 human playtest (DP-1385) | No named check claiming the missing behavior identified in the reviewed sources |
| restore-ordinal | uncovered | Oct 5 live duplicate-body playtest and C resolver inspection | No named check claiming the missing behavior identified in the reviewed sources |
| steal-peaceful | core | steal-depth; steal.inventory-success / steal.coins-zero-gold | named claiming check established |
| ban-bytes | uncovered | Exhaustive mudlog producer inventory / admission audit (#1870) | No named check claiming the missing behavior identified in the reviewed sources |
| lua-return | uncovered | Lua bad-argument diagnostics producer inventory | No named check claiming the missing behavior identified in the reviewed sources |
| host-padding | uncovered | Sept 27 raw connect trace, revisited by Oct 9 admission/mudlog audit | No named check claiming the missing behavior identified in the reviewed sources |
| ambient-draw | uncovered | Oct 13 paired RNG draw trace for teleport-victim, then DP-1426 | Named teleport audience/transfer checks exist, but a claim of complete setup RNG-stream identity was not identified; do not infer it from byte agreement. |
| mob-walk-order | core | room-script-pattern-onpulse; scripts.pattern-room-onpulse-mortal/immortal | named claiming check established |
| combat-entry | uncovered | Populated-world / draw-parity investigation (DP-1438) | No named check claiming the missing behavior identified in the reviewed sources |
| mob-speech | uncovered | Human noticed empty review, then mobile speech sink audit (DP-1440) | No named check claiming the missing behavior identified in the reviewed sources |
| butler-door-args | uncovered | NPC direct-command inventory follow-up and inventory-shadow control (#2153) | No named check claiming the missing behavior identified in the reviewed sources |
| hunter-doors | uncovered | Returning-player production reports, Oct 24–25 | No named check claiming the missing behavior identified in the reviewed sources |
| lantern | uncovered | Returning-player production report; independently rediscovered by play lab | No named check claiming the missing behavior identified in the reviewed sources |
| death-affects | uncovered | Returning-player production report, Oct 24–25 (#2158) | No named check claiming the missing behavior identified in the reviewed sources |
| mana-regen | uncovered | Returning-player production report, Oct 24–25 | No named check claiming the missing behavior identified in the reviewed sources |
| practice-keys | uncovered | Returning-player production report, Oct 24–25 | No named check claiming the missing behavior identified in the reviewed sources |
| save-totals | uncovered | Persistence reload audit during PR #1586 | No named check claiming the missing behavior identified in the reviewed sources |
| ws-linkdead | uncovered | Browser-client census / tab-close investigation (DP-1323) | No named check claiming the missing behavior identified in the reviewed sources |
| god-draws | uncovered | July 24 paired C/Go draw-trace investigation (DP-1212) | No pre-fix named God-creation zero-draw check identified; do not retrofit later draw-count tests as earlier coverage. |
| aiko-entry | uncovered | Aiko browser entry/persistence incident, Sept 8 | No named check claiming the missing behavior identified in the reviewed sources |
| report-proof | core | stat.file-success / wizard-valid-reports-depth | named claiming check established |
| creation-reskin | uncovered | Creation transcript made observable by DiffSetup, July 16 | No named check claiming the missing behavior identified in the reviewed sources |
| tokenizer | uncovered | First real telnet differential gate for punctuation (PR #420) | No named check claiming the missing behavior identified in the reviewed sources |
| crash-stop-proof | core | TestProcessKillIsImmediateSIGKILL | named claiming check established |
| mutation-M010 | core | kick.message-source / TestDoKick_MissMessageFromSkillMessages | named claiming check established |
| mutation-M021 | core | mob.dragon-breath-combat-roll / TestSpecDragonBreath_CombatRollAndSharedReturn | named claiming check established |
| mutation-M025 | core | room.pray-command-gate / TestSpecPrayForItems_FallsThroughOrdinarySocial | named claiming check established |
| mutation-M035 | core | charge.mounted / TestDoCharge_MountedBonusDamage | named claiming check established |
| mutation-M042 | core | spike.success-player-state / TestDoSpike_KillsWerewolf | named claiming check established |
| carrion-prototype-assertion | core | room.carrion-spawn-state; TestSpecCarrion_SuccessfulSpawnStateAndAudience | named claiming check established |
