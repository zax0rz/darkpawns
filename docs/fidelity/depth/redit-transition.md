# REDIT transition checklist

This is the C-backed transition inventory for the bounded Phase 7 `redit`
surface. It is written before the Go state machine so every reachable
`redit_parse` mode has an explicit input, output, mutation, and next-state
contract. C references are the authority; proof status belongs in
`redit.tsv`.

| State/menu | Accepted input | Invalid input/output | Mutation | Next state |
| --- | --- | --- | --- | --- |
| `do_olc` entry | empty → current room; decimal VNUM; `save <zone>` | missing save zone: `Save which zone?`; nonnumeric: `Yikes!  Stop that, someone will get hurt!`; unknown range: `Sorry, there is no zone for that number!`; wrong OLC zone: `You do not have permission to edit this zone.`; duplicate VNUM: `That room is currently being edited by NAME.` | allocate descriptor OLC; existing room is a deep working copy; missing room gets the unfinished-room defaults; save writes the zone and clears its save-list entry | `REDIT_MAIN_MENU`, or playing after entry rejection/save |
| main menu | `1`, `2`, `3`, `4`, `5`–`9`, `A`, `B`, `C`, `S`, `Q` | other input: `Invalid choice!` plus main menu | room name/description/flags/sector/exit/extra/script/copy selection; `Q` only prompts when `OLC_VAL` is dirty | selected mode; `Q` either cleanup or confirm-save |
| confirm-save | `Y`/`y`, `N`/`n` | other input: `Invalid choice!` plus same prompt | `Y` commits working room to memory and save list; `N` frees the working copy | playing; `N` emits no success text, `Y` emits `Room saved to memory.` |
| name | any line | none; C truncates only when length exceeds 75 (`MAX_ROOM_NAME-1`) | replace working name | main menu, dirty |
| description string editor | ordinary line, `@`, `/s`, `/a`, improved `/c /d /e /f /fi /h /i /l /n /r /ra` | improved-editor diagnostics are the shared `improved-edit.c` bytes | ordinary/editor actions change only the working description; `@`/`/s` save the field; `/a` restores the `backstr` | main menu after save/abort |
| room flags | `0`–`28` toggles bit; `0` exits | outside range: `That's not a valid choice!` plus flags menu | toggle working C room-flag bit | flags menu, or main on `0` |
| sector | `0`–`15` | outside range: `Invalid choice!` plus sector menu | set working sector | main menu |
| exit menu | `0` back; `1` target; `2` description; `3` keywords; `4` key; `5` door flags; `6` purge | other input: `Try again : ` and remain | entering a missing direction allocates a zeroed exit; field actions mutate that working exit; purge removes it | main, target/description/keyword/key/door-flags |
| exit target | `-1` or an existing room VNUM | missing target: `That room does not exist, try again : ` | set working exit target (`-1` is no target) | exit menu |
| exit description string editor | ordinary line, `@`, `/s`, `/a`, improved editor commands | shared improved-editor diagnostics | change only working exit description; save/abort returns through `redit_string_cleanup` | exit menu |
| exit keyword | any line | none | replace working keyword | exit menu |
| exit key | any `atoi` line | none | set working key number | exit menu |
| exit door flags | `0` no door, `1` closeable, `2` pickproof | outside range: `That's not a valid choice!` plus door menu | replace working `exit_info` capability bits | exit menu |
| extra-description menu | `0` quit, `1` keyword, `2` description, `3` next | `3` while incomplete: `You can't edit the next extra desc without completing this one.` plus menu | `0` deletes the current incomplete node; `3` selects or appends the next node | main, keyword/description/next |
| extra keyword | any line | none | set current working keyword | extra menu |
| extra description string editor | ordinary line, `@`, `/s`, `/a`, improved editor commands | shared improved-editor diagnostics | change only current working extra description; save/abort returns through cleanup | extra menu |
| copy | `-1` no-op or existing room VNUM | missing source: `That room does not exist, try again : ` | copy source name and description into working room | main menu, dirty after normal completion |
| script menu | `0`, `1` name, `2` flags | any other input falls back to main menu | existing-room script fields mutate the live room script pointer immediately; new rooms reject script menu with `Cannot assign a script until the room is saved at least once.` | main, script name/flags |
| script name | any line, including empty | none | replace live script name (`empty` clears it); no working-room dirty bit | script menu |
| script flags | `0` exits or `1`–`6` toggles | other numeric values redisplay with no diagnostic | toggle live script flag; no working-room dirty bit | script menu/flags menu |
| quit/disconnect cleanup | `Q` in main; transport disconnect in any mode | dirty `Q` requires confirm; disconnect has no save prompt | `Q` frees or commits according to confirmation; disconnect drops OLC and working strings without committing; live script changes already made remain | playing/retained linkdead state |
| internal save/disk save | `Y` confirm commits memory; `redit save <zone>` writes all rooms in the zone | disk open failure is log-only after the entry banner | existing room replaces live room while preserving occupants; new room is inserted by VNUM and all C RNUM references/zone commands/load rooms/exits are adjusted; save-list gets room entry; disk emits C `.wld` order and removes entry | playing for confirm; playing after explicit zone save |

Shared improved-editor behavior is the existing bounded `modify.c`/`improved-edit.c`
port used by `tedit`; this checklist treats its line/action branches as part of
each reachable room/exit/extra string boundary, including blank lines and raw
line routing.
