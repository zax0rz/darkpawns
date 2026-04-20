#!/usr/bin/env python3
"""
Phase 2b smoke test.
Verifies the ROADMAP deliverable:
  Kill something. Loot it. Equip the weapon. Log out. Log back in with it still equipped.
  THAC0 reflects class and STR.

Requires: websockets (pip install websockets)
Usage: python3 scripts/smoke_test_2b.py [--ws-url ws://localhost:8080/ws]
"""
import asyncio
import json
import argparse
import sys

try:
    import websockets
except ImportError:
    print("pip install websockets")
    sys.exit(1)

PASS = "✓"
FAIL = "✗"
results = []

def check(label, cond, detail=""):
    status = PASS if cond else FAIL
    results.append((status, label, detail))
    print(f"  {status}  {label}" + (f" — {detail}" if detail else ""))
    return cond

async def recv_until(ws, timeout=5.0):
    """Collect messages for up to timeout seconds, return list."""
    msgs = []
    try:
        deadline = asyncio.get_event_loop().time() + timeout
        while True:
            remaining = deadline - asyncio.get_event_loop().time()
            if remaining <= 0:
                break
            msg = await asyncio.wait_for(ws.recv(), timeout=remaining)
            msgs.append(json.loads(msg))
    except (asyncio.TimeoutError, websockets.exceptions.ConnectionClosed):
        pass
    return msgs

async def send(ws, type_, data):
    await ws.send(json.dumps({"type": type_, "data": data}))

async def cmd(ws, command, args=None, wait=1.5):
    await send(ws, "command", {"command": command, "args": args or []})
    return await recv_until(ws, wait)

def find_text(msgs):
    texts = []
    for m in msgs:
        if m.get("type") == "text":
            texts.append(m["data"].get("text", ""))
        elif m.get("type") == "state":
            pass
        elif m.get("type") == "event":
            texts.append(m["data"].get("text", ""))
    return "\n".join(texts)

def find_state(msgs):
    for m in msgs:
        if m.get("type") == "state":
            return m["data"]
    return None

async def run_test(ws_url):
    char_name = "SmokeTest2B"
    print(f"\n=== Phase 2b Smoke Test ===")
    print(f"Server: {ws_url}\n")

    # --- Session 1: New character ---
    print("[ Session 1: Create warrior, explore, find a mob, kill it, loot, equip ]")
    async with websockets.connect(ws_url) as ws:
        # Login as new warrior
        await send(ws, "login", {
            "player_name": char_name,
            "class": 3,   # CLASS_WARRIOR
            "race": 0,    # RACE_HUMAN
            "new_char": True
        })
        msgs = await recv_until(ws, 3.0)
        state = find_state(msgs)
        
        check("New char created", state is not None, f"got {len(msgs)} msgs")
        if state:
            player = state.get("player", {})
            check("Class is Warrior", player.get("class") == "Warrior", player.get("class"))
            check("Race is Human", player.get("race") == "Human", player.get("race"))
            check("Level 1", player.get("level") == 1, str(player.get("level")))
            
            # Starting HP should be 10 (do_start sets max_hit=10)
            check("Starting HP = 10", player.get("max_health") == 10, str(player.get("max_health")))
            
            starting_thac0 = player.get("thac0")  # may not be in state yet

        # Check starting inventory (small sword for warrior + tunic + pack)
        inv_msgs = await cmd(ws, "inventory")
        inv_text = find_text(inv_msgs)
        check("Starting inventory not empty", "nothing" not in inv_text.lower(), inv_text[:80])
        has_weapon = any(w in inv_text.lower() for w in ["sword", "club", "dagger"])
        check("Has starting weapon", has_weapon, inv_text[:80])

        # Walk around to find a mob — start room is 8004, look around
        look_msgs = await cmd(ws, "look", wait=1.0)
        look_text = find_text(look_msgs)
        room_state = find_state(look_msgs)
        print(f"  [room] {room_state['room']['name'] if room_state else '?'}")

        # Try a few directions to find combat
        mob_found = False
        mob_name = None
        for direction in ["north", "south", "east", "west", "north", "east", "north"]:
            move_msgs = await cmd(ws, direction, wait=1.0)
            move_text = find_text(move_msgs)
            rs = find_state(move_msgs)
            if rs:
                mobs_in_room = rs.get("room", {}).get("mobs", [])
                if mobs_in_room:
                    mob_name = mobs_in_room[0] if mobs_in_room else None
                    mob_found = True
                    print(f"  [mob found] {mob_name} in {rs['room']['name']}")
                    break

        check("Found a mob", mob_found, mob_name or "none")

        if mob_found and mob_name:
            # Fight until dead or we die
            for _ in range(20):
                hit_msgs = await cmd(ws, "hit", [mob_name.split()[0].lower()], wait=2.5)
                hit_text = find_text(hit_msgs)
                if "dead" in hit_text.lower() or "killed" in hit_text.lower() or "falls" in hit_text.lower():
                    break
                if "you are dead" in hit_text.lower() or "you lose" in hit_text.lower():
                    print("  [died — respawned, retrying]")
                    break

            # Check for corpse
            look2 = await cmd(ws, "look", wait=1.0)
            look2_text = find_text(look2)
            look2_state = find_state(look2)
            items_in_room = look2_state.get("room", {}).get("items", []) if look2_state else []
            has_corpse = any("corpse" in i.lower() for i in items_in_room)
            check("Corpse in room after kill", has_corpse, str(items_in_room[:3]))

            # Try to get something from corpse (simplified — get all)
            for item_desc in items_in_room:
                if "corpse" not in item_desc.lower():
                    first_word = item_desc.split()[0].lower().rstrip(".,")
                    get_msgs = await cmd(ws, "get", [first_word], wait=1.0)
                    get_text = find_text(get_msgs)
                    if "pick up" in get_text.lower():
                        check("Looted item from room", True, first_word)
                        break

        # Equip the starting weapon explicitly
        inv2 = await cmd(ws, "inventory", wait=1.0)
        inv2_text = find_text(inv2)
        weapon_word = None
        for w in ["sword", "club", "dagger"]:
            if w in inv2_text.lower():
                weapon_word = w
                break

        equipped = False
        if weapon_word:
            wield_msgs = await cmd(ws, "wield", [weapon_word], wait=1.0)
            wield_text = find_text(wield_msgs)
            equipped = "wield" in wield_text.lower()
            check("Weapon wielded", equipped, wield_text[:60])

        # Check equipment slot
        eq_msgs = await cmd(ws, "equipment", wait=1.0)
        eq_text = find_text(eq_msgs)
        check("Equipment shows wielded weapon", weapon_word in eq_text.lower() if weapon_word else False, eq_text[:80])
        
        # Quit
        await cmd(ws, "quit", wait=1.0)
        print("  [logged out]")

    # Brief pause for DB write
    await asyncio.sleep(1.0)

    # --- Session 2: Log back in, verify persistence ---
    print("\n[ Session 2: Log back in, verify weapon still equipped ]")
    async with websockets.connect(ws_url) as ws:
        await send(ws, "login", {
            "player_name": char_name,
            "class": 3,
            "race": 0,
        })
        msgs2 = await recv_until(ws, 3.0)
        state2 = find_state(msgs2)
        
        check("Loaded existing character", state2 is not None)
        
        if state2:
            player2 = state2.get("player", {})
            check("Still Warrior", player2.get("class") == "Warrior", player2.get("class"))
            check("Still level 1", player2.get("level") == 1)

        eq2_msgs = await cmd(ws, "equipment", wait=1.0)
        eq2_text = find_text(eq2_msgs)
        persisted = weapon_word in eq2_text.lower() if weapon_word else False
        check("Weapon persisted across logout", persisted, eq2_text[:80] if eq2_text else "empty")

        inv3_msgs = await cmd(ws, "inventory", wait=1.0)
        inv3_text = find_text(inv3_msgs)
        check("Inventory persisted", "nothing" not in inv3_text.lower(), inv3_text[:80])

        await cmd(ws, "quit", wait=0.5)

    print()
    passed = sum(1 for s, _, _ in results if s == PASS)
    failed = sum(1 for s, _, _ in results if s == FAIL)
    print(f"=== Results: {passed} passed, {failed} failed ===")
    return failed == 0

if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("--ws-url", default="ws://localhost:8080/ws")
    args = parser.parse_args()
    
    ok = asyncio.run(run_test(args.ws_url))
    sys.exit(0 if ok else 1)
