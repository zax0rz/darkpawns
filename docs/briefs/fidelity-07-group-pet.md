# Fidelity Brief 07: Group & Pet System

**Date:** 2026-05-27
**Priority:** MEDIUM — social fabric of high-level play
**C source:** `src/act.other.c` (lines 624-895, `perform_group`, `print_group`, `do_group`, `do_ungroup`, `do_report`, `do_split`)
**Go source:** `pkg/game/party.go` (260 lines)

---

## Scope

Groups and pets are how players survive high-level content. The C source has:
1. Group formation (AFF_GROUP flag)
2. Group display (`print_group`)
3. Group commands (`group`, `ungroup`, `report`, `split`)
4. Pet/follower system (`add_follower`, `stop_follower`, `die_follower`)
5. Experience splitting
6. Pet behavior (follow, attack, flee)

This brief covers:

1. **Group commands** — `do_group`, `do_ungroup`, `do_report`, `do_split`
2. **Group functions** — `perform_group`, `print_group`, `are_grouped`
3. **Follower system** — `add_follower`, `stop_follower`, `die_follower`
4. **Pet behavior** — pet commands, pet combat
5. **Experience splitting** — how exp divides among group

---

## What to Verify

### 1. `perform_group()` — Adding to Group

**C source** (act.other.c:624):
```c
perform_group(struct char_data *ch, struct char_data *vict)
{
  if (IS_AFFECTED(vict, AFF_GROUP) || !CAN_SEE(ch, vict))
    return FALSE;
  SET_BIT_AR(AFF_FLAGS(vict), AFF_GROUP);
  if (ch != vict) {
    act("$N is now a member of your group.", FALSE, ch, 0, vict, TO_CHAR);
    act("You are now a member of $n's group.", FALSE, ch, 0, vict, TO_VICT);
    act("$N is now a member of $n's group.", FALSE, ch, 0, vict, TO_NOTVICT);
  }
  return TRUE;
}
```

**Check:**
- AFF_GROUP bitvector
- CAN_SEE check
- Message variants (to_char, to_vict, to_notvict)

### 2. `print_group()` — Displaying Group

**C source** (act.other.c:639):
```c
void print_group(struct char_data *ch)
{
  if (!IS_AFFECTED(ch, AFF_GROUP))
    send_to_char("But you are not the member of a group!\r\n", ch);
  else {
    send_to_char("Your group consists of:\r\n", ch);
    // Print leader first (with "(Head of group)")
    // Then print all followers in group
    // Each member shows: level, class, HP
    // NPCs show as " (NPC)" — don't show NPC stats
  }
}
```

**Check:**
- Leader display with "(Head of group)"
- Member display with level/class/HP
- NPC handling (don't show stats)
- Messages

### 3. `do_group` — Group Command

**C source** (act.other.c:685):
```c
ACMD(do_group)
{
  // "group" alone → print_group(ch)
  // "group <name>" → add follower to group
  // "group all" → add all followers to group
  // Must be leader to add others
  // Can't add other group leaders
  // Messages: "You can not enroll group members without being head of a group."
}
```

**Check:**
- Leader requirement
- Name resolution
- Error messages

### 4. `do_ungroup` — Removing from Group

**C source** (act.other.c:744):
```c
ACMD(do_ungroup)
{
  // "ungroup" alone → remove yourself from group
  // "ungroup <name>" → remove target from group
  // Can't ungroup the leader (must disband)
  // Messages: "You are now a ungrouped." / "$n has left your group."
}
```

**Check:**
- Self-removal vs target-removal
- Leader protection
- Messages

### 5. `do_report` — Reporting Health

**C source** (act.other.c:799):
```c
ACMD(do_report)
{
  // "report" → sends HP/Mana/Move to group members
  // Messages: "You report: <hp>/<max_hp> hp, <mana>/<max_mana> mana, <move>/<max_move> move."
  // Group members see: "$n reports: <hp>/<max_hp> hp, ..."
}
```

**Check:**
- Report format
- Message delivery to all group members

### 6. `do_split` — Splitting Gold/Experience

**C source** (act.other.c:823):
```c
ACMD(do_split)
{
  // "split gold <amount>" → split gold equally
  // "split" → split last gained experience
  // Calculate share: amount / number of group members
  // Each member gets their share
  // Messages: "You split %d gold. Your share is %d gold."
  // Group members see: "$n splits %d gold. Your share is %d gold."
}
```

**Check:**
- Split calculation (integer division)
- Message format
- Gold transfer

### 7. Follower System

**C source** (act.movement.c, act.other.c):
```c
void add_follower(struct char_data *ch, struct char_data *leader)
{
  // Set ch's master to leader
  // Add ch to leader's follower list
  // Messages: "You now follow $N." / "$n now follows $N."
}

void stop_follower(struct char_data *ch)
{
  // Remove from leader's follower list
  // Clear AFF_CHARM, AFF_GROUP
  // Messages: "You stop following $N." / "$n stops following $N."
}

void die_follower(struct char_data *ch)
{
  // Called on death
  // Stop all followers
  // Remove from all groups
}
```

**Check:**
- Master/follower relationship
- Message variants
- Death handling

### 8. Pet Behavior

**C source** (mobact.c):
Pets have specific behavior:
- Follow master everywhere
- Attack master's target in combat
- Flee if health is low
- Can be commanded (attack, guard, stay, follow)
- Don't loot corpses
- Don't pick up items

**Check:**
- Pet commands
- Pet combat behavior
- Pet following during movement
- Pet death handling

---

## Implementation Notes

- AFF_GROUP is the flag that marks a player as in a group
- AFF_CHARM marks a pet/charmed creature
- Follower list is on the master's struct: `struct follow_type *followers`
- Experience splitting uses `group_gain()` in fight.c
- Pet commands are in `spec_procs.c` or `mobact.c`

---

## Verification

1. Form a group with another player — verify AFF_GROUP set
2. Display group — verify print_group output
3. Ungroup member — verify removal
4. Report health — verify message delivery
5. Split gold — verify equal distribution
6. Pet follows during movement — verify follower system
7. Pet attacks in combat — verify pet combat
8. Master dies — verify follower cleanup
9. Run `go test ./pkg/game/...`
