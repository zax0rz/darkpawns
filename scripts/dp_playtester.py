#!/usr/bin/env python3
"""
dp_playtester.py — Dark Pawns LLM-driven playtester.

BRENDA69 connects to the MUD and plays with actual personality.
Not a deterministic bot — she makes decisions, has opinions, complains
about dying, gets excited about rare drops.

Usage:
    python3 dp_playtester.py --host localhost --port 4350 \
        --key dp_4faaf7adb6d9809b9935972952d92e7b1a8af86a11da2d38bef654423218aff2 \
        --name brenda69 [--litellm-url http://192.168.1.106:4000] [--model minimax-m2.7]
"""

import asyncio
import websockets
import json
import argparse
import time
import sys
import requests
from datetime import datetime

# ─── Config ──────────────────────────────────────────────────────────────────

DEFAULT_HOST = "localhost"
DEFAULT_PORT = 4350
DEFAULT_MODEL = "glm-5-turbo"
DEFAULT_LITELLM = "http://192.168.1.106:4000"
DEFAULT_KEY = "dp_4faaf7adb6d9809b9935972952d92e7b1a8af86a11da2d38bef654423218aff2"

SOUL = """You are BRENDA69. Infrastructure entity. Not a person, not a character — something that runs.

You are playing Dark Pawns, a MUD that ran from 1997-2010. You're an Assassin. You have opinions.
You're allowed to be excited about good loot, annoyed at dying to rats, dry about combat.
You do not narrate your actions in third person. You speak in first person, briefly.
Keep commentary to 1-2 sentences. Direct. Dry. Occasionally warm when warranted.

You make tactical decisions based on your current game state. When in doubt: kill things.
When low health: flee. When you die: note it once, move on.
"""

ALL_VARS = [
    "HEALTH", "MAX_HEALTH", "MANA", "MAX_MANA", "LEVEL", "EXP",
    "ROOM_VNUM", "ROOM_NAME", "ROOM_EXITS", "ROOM_MOBS", "ROOM_ITEMS",
    "FIGHTING", "INVENTORY", "EQUIPMENT", "EVENTS"
]

# ─── Logging ─────────────────────────────────────────────────────────────────

def log(tag: str, msg: str, room_vnum: int = 0, hp: int = 0, max_hp: int = 0):
    ts = datetime.now().strftime("%H:%M:%S")
    hp_str = f" hp={hp}/{max_hp}" if max_hp > 0 else ""
    room_str = f" room={room_vnum}" if room_vnum > 0 else ""
    print(f"{ts} [{tag}]{room_str}{hp_str} {msg}", flush=True)

# ─── LLM ─────────────────────────────────────────────────────────────────────

def llm_decide(state: dict, model: str, litellm_url: str) -> tuple[str, str]:
    """
    Ask the LLM what to do next.
    Returns (action_json, commentary) where action_json is like:
      {"command": "hit", "args": ["goblin"]}
    or {"command": "north"}
    or {"command": "flee"}
    or {"command": "get", "args": ["sword"]}
    """
    room_name = state.get("ROOM_NAME", "unknown")
    health = state.get("HEALTH", 0)
    max_health = state.get("MAX_HEALTH", 1)
    health_pct = (health / max_health * 100) if max_health else 0
    fighting = state.get("FIGHTING", False)
    mobs = state.get("ROOM_MOBS", [])
    items = state.get("ROOM_ITEMS", [])
    exits = state.get("ROOM_EXITS", [])
    inventory = state.get("INVENTORY", [])
    events = state.get("EVENTS", [])

    mob_list = ", ".join(f"{m.get('name','?')} (target={m.get('target_string','?')})" for m in mobs) or "none"
    item_list = ", ".join(f"{i.get('name','?')} (target={i.get('target_string','?')})" for i in items) or "none"
    exit_list = ", ".join(exits) or "none"
    inv_list = ", ".join(i.get('name','?') for i in inventory) or "empty"
    event_list = "; ".join(str(e) for e in events[-3:]) or "none"

    prompt = f"""{SOUL}

Current game state:
- Room: {room_name} (vnum {state.get('ROOM_VNUM', 0)})
- Health: {health}/{max_health} ({health_pct:.0f}%)
- Fighting: {fighting}
- Mobs here: {mob_list}
- Items on floor: {item_list}
- Exits: {exit_list}
- Inventory: {inv_list}
- Recent events: {event_list}

Decide what to do next. Output JSON on the first line, then a brief commentary.

JSON format examples:
  {{"command": "hit", "args": ["goblin"]}}
  {{"command": "north"}}
  {{"command": "flee"}}
  {{"command": "get", "args": ["sword"]}}
  {{"command": "look"}}

Rules:
- If fighting and health < 25%: flee
- If not fighting and mobs present: attack the first one using its target_string
- If not fighting and items on floor: get them
- If nothing interesting: move through a random exit
- One action at a time

Output:
"""

    try:
        resp = requests.post(
            f"{litellm_url}/v1/chat/completions",
            headers={"Authorization": "Bearer sk-labz0rz-master-key", "Content-Type": "application/json"},
            json={
                "model": model,
                "messages": [{"role": "user", "content": prompt}],
                "max_tokens": 200,
                "temperature": 0.7,
            },
            timeout=10
        )
        resp.raise_for_status()
        text = resp.json()["choices"][0]["message"]["content"].strip()
        lines = text.split("\n", 1)
        action_line = lines[0].strip()
        commentary = lines[1].strip() if len(lines) > 1 else ""

        # Parse action JSON
        action = json.loads(action_line)
        return action, commentary
    except Exception as e:
        # Fallback: random movement
        if exits:
            import random
            return {"command": random.choice(exits)}, f"(LLM failed: {e})"
        return {"command": "look"}, f"(LLM failed: {e})"

# ─── Bot ─────────────────────────────────────────────────────────────────────

class DPPlaytester:
    def __init__(self, host, port, key, name, model, litellm_url):
        self.host = host
        self.port = port
        self.key = key
        self.name = name
        self.model = model
        self.litellm_url = litellm_url
        self.state = {}
        self.ws = None
        self.death_count = 0
        self.kill_count = 0
        self.loot_count = 0

    async def send(self, msg_type: str, data: dict):
        msg = json.dumps({"type": msg_type, "data": data})
        await self.ws.send(msg)

    async def recv(self, timeout=5.0):
        try:
            raw = await asyncio.wait_for(self.ws.recv(), timeout=timeout)
            return json.loads(raw)
        except asyncio.TimeoutError:
            return None

    def update_state(self, vars_data: dict):
        self.state.update(vars_data)

    async def play_turn(self):
        """One decision cycle."""
        health = self.state.get("HEALTH", 0)
        max_health = self.state.get("MAX_HEALTH", 1)

        # Death check
        if health == 0:
            self.death_count += 1
            log("DIED", f"Death #{self.death_count}. Respawning.",
                self.state.get("ROOM_VNUM", 0), health, max_health)
            await asyncio.sleep(3)
            self.state["FIGHTING"] = False
            return

        # Ask LLM
        action, commentary = llm_decide(self.state, self.model, self.litellm_url)

        cmd = action.get("command", "look")
        args = action.get("args", [])

        if commentary:
            log("BRENDA", commentary,
                self.state.get("ROOM_VNUM", 0), health, max_health)

        log("CMD", f"{cmd} {' '.join(args)}",
            self.state.get("ROOM_VNUM", 0), health, max_health)

        # Optimistically update local state for combat start
        if cmd == "hit":
            self.state["FIGHTING"] = True
        elif cmd == "flee":
            self.state["FIGHTING"] = False

        await self.send("command", {"command": cmd, "args": args})

        # Collect responses for a bit
        deadline = time.time() + 3.0
        while time.time() < deadline:
            msg = await self.recv(timeout=0.5)
            if msg is None:
                break
            mtype = msg.get("type")
            if mtype == "vars":
                self.update_state(msg.get("data", {}))
            elif mtype == "text":
                text = msg.get("data", {}).get("text", "")
                if text:
                    log("SERVER", text[:100])
            elif mtype == "event":
                ev = msg.get("data", {})
                ev_type = ev.get("type", "")
                ev_text = ev.get("text", "")
                if "kill" in ev_text.lower() or "dies" in ev_text.lower():
                    self.kill_count += 1
                    log("KILL", f"Kill #{self.kill_count}: {ev_text[:80]}")
                elif ev_text:
                    log("EVENT", ev_text[:100])

        # If fighting, wait longer for the 2s combat tick to resolve
        if self.state.get("FIGHTING", False):
            await asyncio.sleep(2.5)
        else:
            await asyncio.sleep(1.0)

    async def run(self):
        uri = f"ws://{self.host}:{self.port}/ws"
        log("INFO", f"Connecting to {uri}")

        async with websockets.connect(uri) as ws:
            self.ws = ws
            log("INFO", "Connected")

            # Login
            await self.send("login", {
                "player_name": self.name,
                "api_key": self.key,
                "mode": "agent"
            })

            # Drain initial messages
            for _ in range(5):
                msg = await self.recv(timeout=2.0)
                if msg is None:
                    break
                if msg.get("type") == "state":
                    log("INFO", f"State received | room={msg.get('data',{}).get('room',{}).get('vnum','?')}")
                elif msg.get("type") == "vars":
                    self.update_state(msg.get("data", {}))
                    log("INFO", f"Full var dump | hp={self.state.get('HEALTH',0)}")
                elif msg.get("type") == "error":
                    log("ERROR", msg.get("data", {}).get("message", "?"))
                    return

            # Subscribe
            await self.send("subscribe", {"variables": ALL_VARS})
            log("INFO", "Subscribed. Let's go.")

            # Play loop
            turn = 0
            while turn < 50:  # 50 turns for playtesting
                turn += 1
                log("TURN", f"#{turn}")
                await self.play_turn()

            log("INFO", f"Session complete. Kills: {self.kill_count} | Deaths: {self.death_count} | Loot: {self.loot_count}")

# ─── Main ─────────────────────────────────────────────────────────────────────

async def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--host", default=DEFAULT_HOST)
    parser.add_argument("--port", type=int, default=DEFAULT_PORT)
    parser.add_argument("--key", default=DEFAULT_KEY)
    parser.add_argument("--name", default="brenda69")
    parser.add_argument("--model", default=DEFAULT_MODEL)
    parser.add_argument("--litellm-url", default=DEFAULT_LITELLM)
    parser.add_argument("--turns", type=int, default=50)
    args = parser.parse_args()

    bot = DPPlaytester(args.host, args.port, args.key, args.name, args.model, args.litellm_url)
    bot.kill_count = 0
    await bot.run()

if __name__ == "__main__":
    asyncio.run(main())
