#!/usr/bin/env python3
"""
Phase 2b smoke test.
Verifies the ROADMAP deliverable over WebSocket JSON:
  Create a warrior, equip the starting weapon, log out, log back in, and verify
  the weapon is still equipped and inventory persisted.

Note: on a fresh MUD (empty player store) the FIRST character created is crowned
Implementor — C init_char's first-player-God, faithfully ported to Go. CI wipes
the DB each run, so this test seeds a throwaway bootstrap admin (Session 0)
first, making the warrior under test character #2 — an ordinary mortal — exactly
as on a real server that already has an admin.

Requires: websockets (pip install websockets)
Usage: python3 scripts/smoke_test_2b.py [--ws-url ws://localhost:4350/ws]
"""
import asyncio
import json
import argparse
import sys
import time

try:
    import websockets
except ImportError:
    print("pip install websockets")
    sys.exit(1)

# The C nanny collects new-character passwords after name confirmation and
# limits them to 10 characters. The same password logs the character back in.
PASSWORD = "smoke2bpw"

PASS = "✓"
FAIL = "✗"
results = []


def check(label, cond, detail=""):
    status = PASS if cond else FAIL
    results.append((status, label, detail))
    print(f"  {status}  {label}" + (f" — {detail}" if detail else ""))
    return cond


async def recv_one(ws, timeout=5.0):
    """Read and parse a single JSON message, or None on timeout."""
    try:
        raw = await asyncio.wait_for(ws.recv(), timeout=timeout)
        return json.loads(raw)
    except asyncio.TimeoutError:
        return None


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


def find_welcome_state(msgs):
    """Return the first state message that includes player class/race info."""
    for m in msgs:
        if m.get("type") == "state":
            player = m.get("data", {}).get("player", {})
            if player.get("class") or player.get("race"):
                return m["data"]
    return None


async def create_character(ws):
    """Walk the interactive character-creation prompts and return the welcome state."""
    choices = {
        "confirm_name": "Y",
        "create_password": PASSWORD,
        "confirm_password": PASSWORD,
        "color": "N",
        "sex": "M",
        "race": "H",
        "class": "W",
        "hometown": "K",
        "stats_roll": "Y",
        "motd": "",
        "menu": "1",
    }
    deadline = asyncio.get_event_loop().time() + 30.0
    while True:
        remaining = deadline - asyncio.get_event_loop().time()
        if remaining <= 0:
            raise RuntimeError("Timed out waiting for character creation to complete")

        msg = await recv_one(ws, timeout=remaining)
        if msg is None:
            raise RuntimeError("No message from server during character creation")

        msg_type = msg.get("type")
        if msg_type == "state":
            return msg.get("data")
        if msg_type == "error":
            raise RuntimeError(f"Server error during character creation: {msg.get('data')}")
        if msg_type == "char_create":
            stage = msg.get("data", {}).get("stage")
            if stage not in choices:
                raise RuntimeError(f"Unexpected char_create stage: {stage}")
            await send(ws, "char_input", {"choice": choices[stage]})
            continue
        # Ignore text/event/MOTD traffic while in creation.


async def login_existing(ws, player_name, password):
    """Log in an existing character and return the full welcome state."""
    await send(ws, "login", {"player_name": player_name, "password": password})
    deadline = asyncio.get_event_loop().time() + 10.0
    choices = {"motd": "", "menu": "1"}
    while True:
        remaining = deadline - asyncio.get_event_loop().time()
        if remaining <= 0:
            return None

        msg = await recv_one(ws, timeout=remaining)
        if msg is None:
            return None
        if msg.get("type") == "state":
            return msg.get("data")
        if msg.get("type") == "error":
            raise RuntimeError(f"Server error during login: {msg.get('data')}")
        if msg.get("type") == "char_create":
            stage = msg.get("data", {}).get("stage")
            if stage not in choices:
                raise RuntimeError(f"Unexpected returning-login stage: {stage}")
            await send(ws, "char_input", {"choice": choices[stage]})


async def run_test(ws_url):
    # Use a unique suffix so repeated local runs don't collide with a character
    # left behind by a previous test run.
    char_name = f"SmokeTest2B{int(time.time()) % 100000}"
    print(f"\n=== Phase 2b Smoke Test ===")
    print(f"Server: {ws_url}\n")

    # --- Session 0: consume the first-player-God crown ---
    # On a fresh MUD (empty player store) the FIRST character created is crowned
    # Implementor — C init_char's first-player-God, now faithfully ported to Go.
    # CI wipes the DB each run, so without this the warrior below would be that
    # God (level 40, all skills 100). Create + quit a throwaway bootstrap char
    # first so the warrior under test is character #2 — an ordinary mortal —
    # exactly as on a real server that already has an admin.
    bootstrap_name = f"SmokeBoot{int(time.time()) % 100000}"
    print("[ Session 0: consume first-player-God crown (throwaway bootstrap char) ]")
    async with websockets.connect(ws_url) as ws:
        await send(ws, "login", {
            "player_name": bootstrap_name,
            "password": PASSWORD,
            "new_char": True,
        })
        await create_character(ws)
        await cmd(ws, "quit", wait=1.0)
        print("  [bootstrap god created + logged out]")
    await asyncio.sleep(1.0)  # let the DB write land before Session 1

    # --- Session 1: New character ---
    print("[ Session 1: Create warrior, look around, wield weapon, quit ]")
    async with websockets.connect(ws_url) as ws:
        # new_char enters the shared nanny; its login password field is ignored.
        await send(ws, "login", {
            "player_name": char_name,
            "password": PASSWORD,
            "new_char": True,
        })
        state = await create_character(ws)

        check("New char created", state is not None)
        if state:
            player = state.get("player", {})
            check("Class is Warrior", player.get("class") == "Warrior", player.get("class"))
            check("Race is Human", player.get("race") == "Human", player.get("race"))
            check("Level 1", player.get("level") == 1, str(player.get("level")))

            # Base HP is 10, but AdvanceLevel adds a class/CON bonus at level 1.
            check("Starting HP >= 10", player.get("max_health", 0) >= 10, str(player.get("max_health")))

        # The newbie birth transition is pulse-driven (start_room via
        # room_activity); a command issued from the Burning Hut fires it at
        # command time instead. Wait for the relocation to the hometown,
        # then look.
        birth_msgs = await cmd(ws, "look", wait=6.0)
        for _ in range(8):
            if "Your life begins now" in find_text(birth_msgs):
                break
            birth_msgs += await recv_until(ws, 1.5)
        look_msgs = await cmd(ws, "look", wait=1.0)
        look_text = find_text(look_msgs)
        room_state = find_state(look_msgs)
        check("Look returned room state", room_state is not None)
        if room_state:
            print(f"  [room] {room_state['room']['name']}")

        # Check starting inventory (small sword for warrior + tunic + pack)
        inv_msgs = await cmd(ws, "inventory")
        inv_text = find_text(inv_msgs)
        check("Starting inventory not empty", "nothing" not in inv_text.lower(), inv_text[:80])
        has_weapon = any(w in inv_text.lower() for w in ["sword", "club", "dagger"])
        check("Has starting weapon", has_weapon, inv_text[:80])

        # Equip the starting weapon explicitly.
        weapon_word = None
        for w in ["sword", "club", "dagger"]:
            if w in inv_text.lower():
                weapon_word = w
                break

        equipped = False
        if weapon_word:
            wield_msgs = await cmd(ws, "wield", [weapon_word], wait=1.0)
            wield_text = find_text(wield_msgs)
            equipped = "wield" in wield_text.lower() and "can't" not in wield_text.lower()
            check("Weapon wielded", equipped, wield_text[:60])

        # Check equipment slot.
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
        state2 = await login_existing(ws, char_name, PASSWORD)

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
    parser.add_argument("--ws-url", default="ws://localhost:4350/ws")
    args = parser.parse_args()

    ok = asyncio.run(run_test(args.ws_url))
    sys.exit(0 if ok else 1)
