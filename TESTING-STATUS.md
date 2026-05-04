# Parser Test Status — 2026-05-03

## Test Files Created
- `pkg/parser/mob_test.go` — ~30 tests (17KB) — **format bugs in test data, needs fixing**
- `pkg/parser/obj_test.go` — ~25 tests (14KB) — **23 pass, 2 real parser bugs found**
- `pkg/parser/zon_test.go` — NOT YET WRITTEN
- `pkg/parser/parser_test.go` — NOT YET WRITTEN

## Known Parser Bugs (found by tests)
1. **ExtraDesc trailing newline**: `TestParseObjFile_ExtraDesc` — extra description join adds trailing `\n`
2. **Dollar end marker ignored**: `TestParseObjFile_DollarEnd` — parser continues past `$` sentinel
3. **Mob flags line format**: Parser expects `flags E` on one line, not separate lines

## Test Data Format Requirements
### Mob format (verified correct):
```
#vnum
keywords~
Short desc~
Long desc.~
Detailed desc.~
action_flags affect_flags alignment race E
level thac0 ac hp_num hp_sides hp_plus dmg_num dmg_sides dmg_plus
gold exp
position default_pos sex
E
Str: 18
```

### Obj format:
```
#vnum
keywords~
Short desc~
Long desc~
Action desc~
type extras[0] extras[1] extras[2] extras[3] wear[0] wear[1] wear[2] wear[3]
val[0] val[1] val[2] val[3]
weight cost rent_level
E
extra_desc keywords~
description~
E
$~
```

## Model Availability (2026-05-03)
- GLM-5.1 PAYG: $0 balance — DEAD
- GLM-5.1 Coding Plan: Rate limited — USELESS
- Kimi K2.6: Works, 300s timeout on complex tasks
- Sonnet: Works, expensive

## Next Steps
1. Fix mob_test.go test data format (flags E terminator, stats 9 fields, desc ~ terminator)
2. Fix obj_test.go format issues
3. Write zon_test.go (~20 tests)
4. Write parser_test.go (~15 tests)
5. Dispatch combat, limits, char creation test batches
