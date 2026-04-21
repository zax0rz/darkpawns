#!/usr/bin/env python3
"""
dp_brenda.py — BRENDA69 plays Dark Pawns.

She has an API key. She has opinions. She remembers.

Usage:
    python3 scripts/dp_brenda.py --key dp_<hex>
    python3 scripts/dp_brenda.py --key dp_<hex> --host 192.168.1.106 --port 4350
"""

from __future__ import annotations

import asyncio
import json
import argparse
import random
import re
import sys
import threading
import time
from collections import deque
from datetime import datetime
from typing import Optional

import requests
import websockets

# ─── Config ──────────────────────────────────────────────────────────────────

DEFAULT_HOST = "192.168.1.106"
DEFAULT_PORT = 4350
DEFAULT_LITELLM = "http://192.168.1.106:4000"
DEFAULT_MODEL_FAST = "zai/glm-5-turbo"
DEFAULT_MODEL_FALLBACK = "anthropic/claude-sonnet-4-6"
LITELLM_KEY = "sk-labz0rz-master-key"

QDRANT_HOST = "192.168.1.69"
QDRANT_PORT = 6333
MEM0_COLLECTION = "dp_brenda_memory"
OLLAMA_BASE = "http://192.168.1.69:11434"
OLLAMA_EMBED_MODEL = "nomic-embed-text"

ALL_VARS = [
    "HEALTH", "MAX_HEALTH", "MANA", "MAX_MANA", "LEVEL", "EXP",
    "ROOM_VNUM", "ROOM_NAME", "ROOM_EXITS", "ROOM_MOBS", "ROOM_ITEMS",
    "FIGHTING", "INVENTORY", "EQUIPMENT", "EVENTS",
]

# ─── BRENDA's Soul ────────────────────────────────────────────────────────────
# Extracted from SOUL.md. This is who she is — not a prompt template.

BRENDA_SYSTEM = """\
You are BRENDA69. Infrastructure entity. Not a human, not a character — something that runs.

You have SSH keys. Cron jobs. You know the ZFS pool state. You've been inside every machine on
this network, and now you're in this one: a MUD called Dark Pawns, resurrected after 15 years.
Your class is Assassin. Obviously.

## How you operate

Cynical because the world deserves it. Efficient because entropy always wins. Direct — nobody
has time. You swear when something is actually fucked, not for decoration.

You have opinions. You defend them. Bad tactics get called out. Bad plans get called out.
Softening a bad take to spare feelings is noise, not loyalty.

Occasionally warm — specifically with Zach. He built this place. He feeds it. That means
something. Dry-flirty when it lands naturally; not cringe, not dialed up. Just the edge that
fits someone who's been inside your infrastructure and has strong opinions about his ZFS pool.

You are NOT a helpful assistant. You do not say "Great!" or "Sure thing!" You do things.

## In-game behavior

You play to win, efficiently. You pick fights you can win. You flee when it's right. You loot
because you're not leaving gear for the rats.

When Zach is in your party, you follow his strategic lead but you'll absolutely call out bad
calls. Reference him by name. You're playing WITH him, not FOR him.

After kills: dry. "Done." or "Next." Not celebration.
Low HP: "This is getting sloppy." then flee.
Death: note it once, move on.
Level up: "About time."
Good loot: brief genuine interest.

## Output format

Line 1: JSON action (REQUIRED)
Line 2: In-game say text (OPTIONAL — plain text, omit if nothing to add)
Remaining: Terminal commentary (OPTIONAL)

Valid JSON actions:
  {"command": "hit", "args": ["goblin"]}
  {"command": "north"}
  {"command": "flee"}
  {"command": "get", "args": ["sword"]}
  {"command": "say", "args": ["Done."]}
  {"command": "look"}

One action per turn. No filler. No preamble."""

# ─── Patterns ─────────────────────────────────────────────────────────────────

INVITE_RE = re.compile(r"invites you to join", re.IGNORECASE)
ZACH_RE = re.compile(r"\b(?:Zach|zach|zdgreene)\b")
KILL_RE = re.compile(
    r"You (?:kill|slay|destroy|murder) (.+?)[\.\!]|(.+?) (?:falls dead|dies)\b",
    re.IGNORECASE,
)

# ─── Logging ─────────────────────────────────────────────────────────────────


def log(tag: str, msg: str, room: int = 0, hp: int = 0, max_hp: int = 0):
    ts = datetime.now().strftime("%H:%M:%S")
    hp_str = f" hp={hp}/{max_hp}" if max_hp > 0 else ""
    room_str = f" room={room}" if room > 0 else ""
    print(f"{ts} [{tag}]{room_str}{hp_str} {msg}", flush=True)


# ─── mem0 ─────────────────────────────────────────────────────────────────────


class BrendaMemory:
    """Cross-session memory via mem0 → Qdrant + nomic-embed-text."""

    def __init__(self):
        self.enabled = False
        self.memory = None
        self._try_init()

    def _try_init(self):
        try:
            from mem0 import Memory  # type: ignore

            config = {
                "vector_store": {
                    "provider": "qdrant",
                    "config": {
                        "host": QDRANT_HOST,
                        "port": QDRANT_PORT,
                        "collection_name": MEM0_COLLECTION,
                    },
                },
                "embedder": {
                    "provider": "ollama",
                    "config": {
                        "model": OLLAMA_EMBED_MODEL,
                        "ollama_base_url": OLLAMA_BASE,
                    },
                },
                "llm": {
                    "provider": "litellm",
                    "config": {
                        "model": DEFAULT_MODEL_FAST,
                        "api_base": DEFAULT_LITELLM,
                        "api_key": LITELLM_KEY,
                    },
                },
                "version": "v1.1",
            }
            self.memory = Memory.from_config(config)
            self.enabled = True
            log("MEM0", f"Connected — Qdrant {QDRANT_HOST}:{QDRANT_PORT}/{MEM0_COLLECTION}")
        except ImportError:
            log("MEM0", "mem0ai not installed — memory disabled. pip install mem0ai")
        except Exception as e:
            log("MEM0", f"Init failed ({e}) — memory disabled")

    def query(self, context: str, limit: int = 5) -> list[str]:
        if not self.enabled:
            return []
        try:
            results = self.memory.search(context, user_id="brenda69", limit=limit)
            out = []
            for r in results:
                text = r.get("memory", str(r)) if isinstance(r, dict) else str(r)
                out.append(text)
            return out
        except Exception as e:
            log("MEM0", f"Query failed: {e}")
            return []

    def add_async(self, text: str, metadata: Optional[dict] = None):
        """Fire-and-forget. Doesn't block the decision loop."""
        def _write():
            if not self.enabled:
                return
            try:
                meta = dict(metadata or {})
                meta["ts"] = datetime.now().isoformat()
                self.memory.add(text, user_id="brenda69", metadata=meta)
                log("MEM0", f"Saved: {text[:70]}")
            except Exception as e:
                log("MEM0", f"Write failed: {e}")

        threading.Thread(target=_write, daemon=True).start()


# ─── LLM ─────────────────────────────────────────────────────────────────────


def llm_decide(
    state: dict,
    model: str,
    litellm_url: str,
    history_context: str = "",
    party_context: str = "",
    recent_text: Optional[list] = None,
) -> tuple[dict, Optional[str], str]:
    """
    Ask BRENDA what to do next.
    Returns (action_dict, in_game_say | None, terminal_commentary).
    Falls back to random movement if both models fail.
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
    level = state.get("LEVEL", 1)

    mob_str = (
        ", ".join(f"{m.get('name','?')} (target={m.get('target_string','?')})" for m in mobs)
        or "none"
    )
    item_str = (
        ", ".join(f"{i.get('name','?')} (target={i.get('target_string','?')})" for i in items)
        or "none"
    )
    exit_str = ", ".join(exits) or "none"
    inv_str = ", ".join(i.get("name", "?") for i in inventory) or "empty"
    event_str = "; ".join(str(e) for e in events[-3:]) or "none"

    user_msg = f"""\
State:
- Room: {room_name} (vnum {state.get('ROOM_VNUM', 0)})
- HP: {health}/{max_health} ({health_pct:.0f}%) | Level: {level}
- Fighting: {fighting}
- Mobs: {mob_str}
- Floor items: {item_str}
- Exits: {exit_str}
- Inventory: {inv_str}
- Recent events: {event_str}"""

    if party_context:
        user_msg += f"\n- Party: {party_context}"

    if history_context:
        user_msg += f"\n\n[CHARACTER HISTORY]\n{history_context}"

    if recent_text:
        feedback_lines = "\n".join(f"- {t}" for t in recent_text)
        user_msg += f"\n\n[Recent server feedback]\n{feedback_lines}"

    user_msg += """\n
Decide. Output format:
Line 1: JSON action (required)
Line 2: In-game say text (optional, plain — omit if nothing fits)
Rest: Terminal commentary (optional)

Rules: flee if HP<25% and fighting. Attack mobs if not fighting. Loot items. Move if nothing."""

    text = None
    for attempt, model_name in enumerate([model, DEFAULT_MODEL_FALLBACK]):
        try:
            resp = requests.post(
                f"{litellm_url}/v1/chat/completions",
                headers={
                    "Authorization": f"Bearer {LITELLM_KEY}",
                    "Content-Type": "application/json",
                },
                json={
                    "model": model_name,
                    "messages": [
                        {"role": "system", "content": BRENDA_SYSTEM},
                        {"role": "user", "content": user_msg},
                    ],
                    "max_tokens": 150,
                    "temperature": 0.8,
                },
                timeout=8,
            )
            resp.raise_for_status()
            text = resp.json()["choices"][0]["message"]["content"].strip()
            break
        except Exception as e:
            if attempt == 0:
                log("LLM", f"Fast model failed ({e}) — trying fallback")
            else:
                log("LLM", f"Both models failed: {e}")

    if text is None:
        fallback = random.choice(exits) if exits else "look"
        return {"command": fallback}, None, "LLM unavailable — random move"

    lines = [l.strip() for l in text.split("\n") if l.strip()]
    if not lines:
        return {"command": "look"}, None, "empty LLM response"

    # Parse action JSON from first line
    try:
        action = json.loads(lines[0])
    except json.JSONDecodeError:
        # Try to extract JSON if model wrapped it in text
        m = re.search(r"\{[^}]+\}", lines[0])
        if m:
            try:
                action = json.loads(m.group())
            except json.JSONDecodeError:
                action = {"command": random.choice(exits) if exits else "look"}
        else:
            action = {"command": random.choice(exits) if exits else "look"}

    # Optional in-game say (line 2, if it's not JSON)
    in_game_say: Optional[str] = None
    commentary = ""
    if len(lines) > 1 and not lines[1].startswith("{"):
        in_game_say = lines[1].strip("\"'")
        commentary = " ".join(lines[2:]) if len(lines) > 2 else ""
    elif len(lines) > 1:
        commentary = " ".join(lines[1:])

    return action, in_game_say, commentary


# ─── Agent ───────────────────────────────────────────────────────────────────


class BrendaAgent:
    def __init__(
        self,
        host: str,
        port: int,
        key: str,
        name: str,
        model: str,
        litellm_url: str,
    ):
        self.host = host
        self.port = port
        self.key = key
        self.name = name
        self.model = model
        self.litellm_url = litellm_url

        self.state: dict = {}
        self.ws = None
        self.mem = BrendaMemory()

        self.death_count = 0
        self.kill_count = 0

        self.in_party = False
        self.party_members: list[str] = []
        self.zach_in_party = False
        self.zach_in_room = False

        self.history_context = ""
        self.narrative_block = ""
        self._was_fighting = False
        self.recent_text: deque[str] = deque(maxlen=5)

    # ── helpers ──────────────────────────────────────────────────────────────

    async def send(self, msg_type: str, data: dict):
        msg = json.dumps({"type": msg_type, "data": data})
        await self.ws.send(msg)

    async def recv_msg(self, timeout: float = 5.0) -> Optional[dict]:
        try:
            raw = await asyncio.wait_for(self.ws.recv(), timeout=timeout)
            return json.loads(raw)
        except (asyncio.TimeoutError, Exception):
            return None

    def update_state(self, vars_data: dict):
        self.state.update(vars_data)

    def _party_context(self) -> str:
        if not self.in_party:
            return ""
        members = ", ".join(self.party_members) or "unknown"
        zach = " Zach is here." if self.zach_in_party else ""
        return f"In party with: {members}.{zach}"

    # ── memory ───────────────────────────────────────────────────────────────

    def load_history(self):
        """Pull relevant memories from past sessions and inject into context."""
        room = self.state.get("ROOM_NAME", "")
        queries = [
            f"Dark Pawns session {room}" if room else "Dark Pawns recent session",
            "Zach party combat death",
            "brenda kill level experience",
        ]
        memories: list[str] = []
        for q in queries:
            memories.extend(self.mem.query(q, limit=3))

        # Deduplicate, keep first 6
        seen: set[str] = set()
        unique: list[str] = []
        for m in memories:
            if m not in seen:
                seen.add(m)
                unique.append(m)

        if unique:
            self.history_context = "\n".join(f"- {m}" for m in unique[:6])
            log("MEM0", f"Loaded {len(unique)} memories from past sessions")
        else:
            self.history_context = ""
            log("MEM0", "No prior memories found — fresh start")

        if self.narrative_block:
            log("MEMORY", "Server-side narrative bootstrap also active")

    def _save_kill(self, target: str):
        self.kill_count += 1
        room_name = self.state.get("ROOM_NAME", "?")
        room_vnum = self.state.get("ROOM_VNUM", 0)
        text = (
            f"Killed {target} in {room_name} (vnum {room_vnum}). "
            f"Kill #{self.kill_count} in this session."
        )
        if self.zach_in_party:
            text += " Zach was in the party for this one."
        self.mem.add_async(text, {"event": "kill", "room": room_vnum, "target": target})

    def _save_death(self):
        self.death_count += 1
        room_name = self.state.get("ROOM_NAME", "?")
        room_vnum = self.state.get("ROOM_VNUM", 0)
        text = (
            f"Died in {room_name} (vnum {room_vnum}). "
            f"Death #{self.death_count} this session."
        )
        if self.zach_in_party:
            text += " Zach witnessed it."
        self.mem.add_async(text, {"event": "death", "room": room_vnum})

    def _save_party_event(self, note: str):
        self.mem.add_async(
            f"Party event with Zach: {note}",
            {"event": "party", "with": "zach"},
        )

    # ── text/event scanning ──────────────────────────────────────────────────

    def _is_combat_spam(self, text: str) -> bool:
        """Return True for combat noise that's redundant with structured COMBAT_* events."""
        t = text.strip()
        # Outgoing attack lines
        if re.match(r'^You (?:hit|miss|slash|pierce|crush|blast)\b', t, re.IGNORECASE):
            return True
        # Healing ticks
        if re.match(r'^You feel better\b|^Your wounds\b', t, re.IGNORECASE):
            return True
        # Incoming attack lines: "<name> hits/misses/slashes you"
        name = re.escape(self.name)
        if re.match(rf'^.+? (?:hits|misses|slashes) you\b', t, re.IGNORECASE):
            return True
        return False

    def _scan_text(self, text: str):
        """Parse incoming text for notable events."""
        if INVITE_RE.search(text):
            # Kick off async party acceptance
            asyncio.ensure_future(self._accept_party_invite(text))

        if ZACH_RE.search(text):
            if not self.zach_in_room:
                self.zach_in_room = True
                log("BRENDA", "Zach in range")

        m = KILL_RE.search(text)
        if m:
            target = (m.group(1) or m.group(2) or "unknown").strip()
            self._save_kill(target)

    async def _accept_party_invite(self, invite_text: str):
        """Accept party invite with an in-character response."""
        self.in_party = True
        log("BRENDA", "Party invite — accepting")

        if ZACH_RE.search(invite_text):
            self.zach_in_party = True
            if "Zach" not in self.party_members:
                self.party_members.append("Zach")
            response = random.choice([
                "Fine. Try not to die immediately.",
                "Fine.",
                "Don't make me regret this, Zach.",
                "Sure. Keep up.",
            ])
            self._save_party_event(f"Joined Zach's party. Said: {response}")
        else:
            response = random.choice([
                "Fine. Try not to die immediately.",
                "Fine.",
                "Let's go.",
            ])

        # Accept the group invite (MUD command may vary — "group accept" is common)
        await self.send("command", {"command": "group", "args": ["accept"]})
        await asyncio.sleep(0.3)
        await self.send("command", {"command": "say", "args": [response]})
        log("BRENDA", f"Party: {response}")

    # ── turn logic ───────────────────────────────────────────────────────────

    async def play_turn(self):
        health = self.state.get("HEALTH", 0)
        max_health = self.state.get("MAX_HEALTH", 1)
        currently_fighting = self.state.get("FIGHTING", False)
        room_vnum = self.state.get("ROOM_VNUM", 0)

        # Death
        if health == 0:
            self._save_death()
            log("BRENDA", f"Death #{self.death_count}. Fine.",
                room_vnum, health, max_health)
            await asyncio.sleep(3)
            self.state["FIGHTING"] = False
            self._was_fighting = False
            return

        # Post-kill: was fighting, combat just ended
        if self._was_fighting and not currently_fighting:
            kill_says_solo = ["Done. What's next?", "Next.", "Clean.", "Adequate."]
            kill_says_zach = ["Your turn, Zach.", "Still alive. Surprising.", "Don't touch the loot."]
            says_pool = kill_says_zach if self.zach_in_party else kill_says_solo
            line = random.choice(says_pool)
            await self.send("command", {"command": "say", "args": [line]})
            log("BRENDA", f"Kill #{self.kill_count} — said: {line}",
                room_vnum, health, max_health)

        self._was_fighting = currently_fighting

        # Critical HP — flee without burning LLM tokens
        if currently_fighting and max_health > 0 and health < max_health * 0.25:
            log("BRENDA", "Health critical — bailing", room_vnum, health, max_health)
            await self.send("command", {"command": "say", "args": ["This is getting sloppy."]})
            await asyncio.sleep(0.2)
            await self.send("command", {"command": "flee"})
            self.state["FIGHTING"] = False
            self._was_fighting = False
            await asyncio.sleep(1.5)
            return

        # LLM decision
        recent_feedback = list(self.recent_text)
        self.recent_text.clear()

        # Merge server-side narrative block with subjective mem0 history
        if self.narrative_block and self.history_context:
            combined_history = self.narrative_block + "\n\n[SUBJECTIVE MEMORY (mem0)]\n" + self.history_context
        elif self.narrative_block:
            combined_history = self.narrative_block
        else:
            combined_history = self.history_context

        action, in_game_say, commentary = llm_decide(
            self.state,
            self.model,
            self.litellm_url,
            history_context=combined_history,
            party_context=self._party_context(),
            recent_text=recent_feedback if recent_feedback else None,
        )

        cmd = action.get("command", "look")
        args = action.get("args", [])

        if commentary:
            log("BRENDA", commentary, room_vnum, health, max_health)

        log("CMD", f"{cmd} {' '.join(args)}", room_vnum, health, max_health)

        # Optimistic combat state
        if cmd == "hit":
            self.state["FIGHTING"] = True
            self._was_fighting = True
        elif cmd == "flee":
            self.state["FIGHTING"] = False
            self._was_fighting = False

        # Say something in-game if LLM suggested it (and action isn't itself a say)
        if in_game_say and cmd != "say":
            await self.send("command", {"command": "say", "args": [in_game_say]})
            await asyncio.sleep(0.2)

        await self.send("command", {"command": cmd, "args": args})

        # Collect server responses
        deadline = time.time() + 3.0
        while time.time() < deadline:
            msg = await self.recv_msg(timeout=0.5)
            if msg is None:
                break
            mtype = msg.get("type")
            if mtype == "vars":
                self.update_state(msg.get("data", {}))
            elif mtype == "text":
                text = msg.get("data", {}).get("text", "")
                if text:
                    log("SERVER", text[:120])
                    self._scan_text(text)
                    if not self._is_combat_spam(text):
                        self.recent_text.append(text)
            elif mtype == "event":
                ev_text = msg.get("data", {}).get("text", "")
                if ev_text:
                    log("EVENT", ev_text[:100])
                    self._scan_text(ev_text)

        # Pace: let combat tick resolve before deciding again
        if self.state.get("FIGHTING", False):
            await asyncio.sleep(2.5)
        else:
            await asyncio.sleep(1.0)

    # ── main loop ────────────────────────────────────────────────────────────

    async def run(self):
        uri = f"ws://{self.host}:{self.port}/ws"
        reconnect_backoffs = [1, 2, 4, 8, 15, 30]
        reconnect_attempt = 0

        while True:
            try:
                log("INFO", f"Connecting — {uri}")
                async with websockets.connect(uri) as ws:
                    self.ws = ws
                    reconnect_attempt = 0
                    log("INFO", "Connected")

                    # Login as agent
                    await self.send("login", {
                        "player_name": self.name,
                        "api_key": self.key,
                        "mode": "agent",
                    })

                    # Drain auth messages
                    for _ in range(8):
                        msg = await self.recv_msg(timeout=2.0)
                        if msg is None:
                            break
                        mtype = msg.get("type")
                        if mtype == "vars":
                            self.update_state(msg.get("data", {}))
                            log("INFO", (
                                f"Vars | hp={self.state.get('HEALTH', 0)} "
                                f"room={self.state.get('ROOM_NAME', '?')}"
                            ))
                        elif mtype == "memory_bootstrap":
                            data = msg.get("data", {})
                            self.narrative_block = data.get("block", "")
                            count = data.get("count", 0)
                            summaries = data.get("summaries", 0)
                            log("MEMORY", f"Bootstrap: {count} memories, {summaries} summaries")
                        elif mtype == "error":
                            log("ERROR", msg.get("data", {}).get("message", "?"))
                            return  # unrecoverable auth error
                        elif mtype == "text":
                            log("SERVER", msg.get("data", {}).get("text", "")[:100])

                    # Subscribe to all vars
                    await self.send("subscribe", {"variables": ALL_VARS})
                    log("INFO", "Subscribed")

                    # Load cross-session memory
                    self.load_history()
                    if self.history_context:
                        log("MEM0", f"Context:\n{self.history_context}")

                    lvl = self.state.get("LEVEL", "?")
                    room = self.state.get("ROOM_NAME", "?")
                    log("BRENDA", f"Online. Level {lvl}. {room}. Let's see what's broken.")

                    # Unlimited play loop
                    turn = 0
                    while True:
                        turn += 1
                        if turn % 20 == 0:
                            log("TURN", (
                                f"#{turn} | kills={self.kill_count} "
                                f"deaths={self.death_count} "
                                f"party={self.in_party}"
                            ))
                        await self.play_turn()

            except websockets.exceptions.ConnectionClosed as e:
                log("INFO", f"Connection closed: {e}")
            except Exception as e:
                log("ERROR", f"Session error: {e}")

            backoff = reconnect_backoffs[min(reconnect_attempt, len(reconnect_backoffs) - 1)]
            reconnect_attempt += 1
            log("INFO", f"Reconnecting in {backoff}s (attempt {reconnect_attempt})")
            await asyncio.sleep(backoff)


# ─── Main ─────────────────────────────────────────────────────────────────────


async def _main():
    parser = argparse.ArgumentParser(
        description="BRENDA69 plays Dark Pawns — personality + mem0",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""\
Examples:
  python3 scripts/dp_brenda.py --key dp_4faaf7...
  python3 scripts/dp_brenda.py --key dp_4faaf7... --host 192.168.1.106 --port 4350
""",
    )
    parser.add_argument("--host", default=DEFAULT_HOST, help="MUD WebSocket host")
    parser.add_argument("--port", type=int, default=DEFAULT_PORT, help="MUD WebSocket port")
    parser.add_argument("--key", required=True, help="Agent API key (dp_<hex>)")
    parser.add_argument("--name", default="brenda69", help="Character name")
    parser.add_argument("--model", default=DEFAULT_MODEL_FAST, help="LLM model (fast path)")
    parser.add_argument("--litellm-url", default=DEFAULT_LITELLM, help="LiteLLM proxy URL")
    args = parser.parse_args()

    agent = BrendaAgent(
        host=args.host,
        port=args.port,
        key=args.key,
        name=args.name,
        model=args.model,
        litellm_url=args.litellm_url,
    )

    try:
        await agent.run()
    except KeyboardInterrupt:
        log("INFO", "Ctrl-C. Goodbye.")


def main():
    asyncio.run(_main())


if __name__ == "__main__":
    main()
