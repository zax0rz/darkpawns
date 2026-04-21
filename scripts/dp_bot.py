#!/usr/bin/env python3
"""
dp_bot.py — Dark Pawns Proof-of-Concept Agent
=============================================

A deterministic state-machine bot that connects to the Dark Pawns WebSocket
server, navigates rooms, fights mobs, loots corpses, and reports dryly.

No LLM calls. Pure logic. Built for Phase 4.4 of the Dark Pawns resurrection.

Getting an API Key
------------------
1. Build and run the agent key generator:
     go run ./cmd/agentkeygen -name "your_bot_name" -db "postgres://..."
2. Copy the generated key (shown once) into Vaultwarden or your secrets manager.
3. Pass it to the bot with --key.

Running the Bot
---------------
Agent mode (existing character + API key):
     python3 dp_bot.py --host 192.168.1.106 --port 4350 --key dp_abc123... --name brenda69

New character (human mode, no key needed):
     python3 dp_bot.py --host 192.168.1.106 --port 4350 --name brenda69 --new

What It Does
------------
- Connects via WebSocket and authenticates
- Subscribes to all game variables (HEALTH, ROOM_MOBS, FIGHTING, etc.)
- Wanders rooms via random exits until it finds a mob
- Attacks the first mob, lets the 2-second combat tick run
- Flees if health drops below 25%
- Loots everything off the floor after the fight
- Prints dry one-liners about its loot
- Trips a circuit breaker on 3 repeated negative events and disconnects
- Handles death (respawn, 3-second wait, recovery navigation)
- Reconnects with exponential backoff on connection drops

Dependencies: websockets (pip install websockets)
Python: 3.10+
"""

from __future__ import annotations

import argparse
import asyncio
import json
import logging
import random
import sys
import time
from collections import deque
from dataclasses import dataclass, field
from typing import Any

import websockets
from websockets.exceptions import ConnectionClosed

# ---------------------------------------------------------------------------
# Constants
# ---------------------------------------------------------------------------

WS_PATH = "/ws"
RECONNECT_BACKOFFS = [1, 2, 4, 8, 15, 30]  # seconds, capped at 30
CIRCUIT_BREAKER_THRESHOLD = 3
NEGATIVE_EVENTS = {
    "rate_limited",
    "error",
    "died",
    "flee_failed",
    "no_exits",
    "connection_lost",
}

ALL_VARIABLES = [
    "HEALTH", "MAX_HEALTH", "MANA", "MAX_MANA", "LEVEL", "EXP",
    "ROOM_VNUM", "ROOM_NAME", "ROOM_EXITS",
    "ROOM_MOBS", "ROOM_ITEMS",
    "FIGHTING", "INVENTORY", "EQUIPMENT", "EVENTS",
]

# ---------------------------------------------------------------------------
# Logging
# ---------------------------------------------------------------------------

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s [%(levelname)s] %(message)s",
    datefmt="%Y-%m-%d %H:%M:%S",
)
log = logging.getLogger("dp_bot")


# ---------------------------------------------------------------------------
# Data classes
# ---------------------------------------------------------------------------

@dataclass
class RoomMob:
    name: str
    instance_id: str
    target_string: str
    vnum: int
    fighting: bool

    @classmethod
    def from_dict(cls, d: dict) -> "RoomMob":
        return cls(
            name=d.get("name", ""),
            instance_id=d.get("instance_id", ""),
            target_string=d.get("target_string", ""),
            vnum=d.get("vnum", 0),
            fighting=d.get("fighting", False),
        )


@dataclass
class RoomItem:
    name: str
    instance_id: str
    target_string: str
    vnum: int

    @classmethod
    def from_dict(cls, d: dict) -> "RoomItem":
        return cls(
            name=d.get("name", ""),
            instance_id=d.get("instance_id", ""),
            target_string=d.get("target_string", ""),
            vnum=d.get("vnum", 0),
        )


@dataclass
class BotState:
    health: int = 0
    max_health: int = 0
    mana: int = 0
    max_mana: int = 0
    level: int = 0
    exp: int = 0
    room_vnum: int | None = None
    room_name: str = ""
    room_exits: list[str] = field(default_factory=list)
    room_mobs: list[RoomMob] = field(default_factory=list)
    room_items: list[RoomItem] = field(default_factory=list)
    fighting: bool = False
    inventory: list[dict] = field(default_factory=list)
    equipment: dict[str, Any] = field(default_factory=dict)
    events: list[dict] = field(default_factory=list)

    def reset_combat(self) -> None:
        """Clear tactical state after death or on demand."""
        self.fighting = False
        self.events.clear()


# ---------------------------------------------------------------------------
# Circuit Breaker
# ---------------------------------------------------------------------------

class CircuitBreaker:
    """Trips after N repeated negative events."""

    def __init__(self, threshold: int = CIRCUIT_BREAKER_THRESHOLD) -> None:
        self.threshold = threshold
        self._history: deque[str] = deque(maxlen=threshold)
        self._tripped = False

    def record(self, event: str) -> bool:
        """Record an event. Returns True if breaker is now tripped."""
        if self._tripped:
            return True
        if event in NEGATIVE_EVENTS:
            self._history.append(event)
            if len(self._history) == self.threshold and len(set(self._history)) == 1:
                self._tripped = True
                return True
        else:
            self._history.clear()
        return False

    @property
    def tripped(self) -> bool:
        return self._tripped


# ---------------------------------------------------------------------------
# DPBot
# ---------------------------------------------------------------------------

class DPBot:
    """Deterministic Dark Pawns agent."""

    def __init__(
        self,
        host: str,
        port: int,
        name: str,
        api_key: str | None = None,
        new_char: bool = False,
    ) -> None:
        self.host = host
        self.port = port
        self.name = name
        self.api_key = api_key
        self.new_char = new_char

        self.ws: websockets.WebSocketClientProtocol | None = None
        self.state = BotState()
        self.circuit = CircuitBreaker()
        self._connected = False
        self._authenticated = False
        self._vars_ready = False
        self._dead = False
        self._recovery_mode = False
        self._reconnect_attempt = 0
        self._shutdown = False

    # ------------------------------------------------------------------
    # Connection lifecycle
    # ------------------------------------------------------------------

    async def connect(self) -> bool:
        uri = f"ws://{self.host}:{self.port}{WS_PATH}"
        log.info("Connecting to %s", uri)
        try:
            self.ws = await websockets.connect(uri)
            self._connected = True
            self._reconnect_attempt = 0
            log.info("WebSocket connected")
            return True
        except Exception as exc:
            log.error("Connection failed: %s", exc)
            return False

    async def disconnect(self) -> None:
        self._connected = False
        self._authenticated = False
        self._vars_ready = False
        if self.ws:
            try:
                await self.ws.close()
            except Exception:
                pass
            self.ws = None
            log.info("Disconnected")

    async def reconnect(self) -> bool:
        await self.disconnect()
        backoff = min(
            RECONNECT_BACKOFFS[min(self._reconnect_attempt, len(RECONNECT_BACKOFFS) - 1)],
            30,
        )
        self._reconnect_attempt += 1
        log.info("Reconnecting in %ds (attempt %d)", backoff, self._reconnect_attempt)
        await asyncio.sleep(backoff)
        return await self.connect()

    # ------------------------------------------------------------------
    # Messaging helpers
    # ------------------------------------------------------------------

    async def send(self, msg: dict) -> None:
        if not self.ws:
            return
        raw = json.dumps(msg)
        await self.ws.send(raw)
        log.debug("→ %s", raw[:200])

    async def send_login(self) -> None:
        if self.api_key:
            data = {"player_name": self.name, "api_key": self.api_key, "mode": "agent"}
        else:
            data = {
                "player_name": self.name,
                "class": 3,      # Warrior
                "race": 0,       # Human
                "new_char": self.new_char,
            }
        await self.send({"type": "login", "data": data})
        log.info("Sent login for %s (mode=%s)", self.name, "agent" if self.api_key else "human")

    async def send_subscribe(self) -> None:
        await self.send({"type": "subscribe", "data": {"variables": ALL_VARIABLES}})
        log.info("Subscribed to all variables")

    async def send_command(self, command: str, args: list[str] | None = None) -> None:
        if self.circuit.tripped:
            log.warning("Circuit breaker tripped — dropping command: %s", command)
            return
        payload: dict[str, Any] = {"command": command}
        if args:
            payload["args"] = args
        await self.send({"type": "command", "data": payload})
        log.info("Command: %s %s", command, " ".join(args) if args else "")

    # ------------------------------------------------------------------
    # Message dispatch
    # ------------------------------------------------------------------

    async def handle_message(self, raw: str) -> None:
        try:
            msg = json.loads(raw)
        except json.JSONDecodeError:
            log.warning("Invalid JSON: %s", raw[:200])
            return

        mtype = msg.get("type", "")
        data = msg.get("data", {})

        match mtype:
            case "state":
                await self.on_state(data)
            case "vars":
                await self.on_vars(data)
            case "event":
                await self.on_event(data)
            case "text":
                await self.on_text(data)
            case "error":
                await self.on_error(data)
            case _:
                log.debug("Unhandled message type: %s", mtype)

    async def on_state(self, data: dict) -> None:
        """Initial full state on login (human mode)."""
        player = data.get("player", {})
        room = data.get("room", {})
        self.state.health = player.get("health", 0)
        self.state.max_health = player.get("max_health", 0)
        self.state.level = player.get("level", 0)
        self.state.room_vnum = room.get("vnum")
        self.state.room_name = room.get("name", "")
        self.state.room_exits = room.get("exits", [])
        self._authenticated = True
        log.info("State received | room=%s hp=%d/%d",
                 self.state.room_vnum, self.state.health, self.state.max_health)

    async def on_vars(self, data: dict) -> None:
        """Variable update — delta or full dump."""
        if "HEALTH" in data:
            self.state.health = data["HEALTH"]
        if "MAX_HEALTH" in data:
            self.state.max_health = data["MAX_HEALTH"]
        if "MANA" in data:
            self.state.mana = data["MANA"]
        if "MAX_MANA" in data:
            self.state.max_mana = data["MAX_MANA"]
        if "LEVEL" in data:
            self.state.level = data["LEVEL"]
        if "EXP" in data:
            self.state.exp = data["EXP"]
        if "ROOM_VNUM" in data:
            self.state.room_vnum = data["ROOM_VNUM"]
        if "ROOM_NAME" in data:
            self.state.room_name = data["ROOM_NAME"]
        if "ROOM_EXITS" in data:
            self.state.room_exits = data["ROOM_EXITS"]
        if "ROOM_MOBS" in data:
            self.state.room_mobs = [RoomMob.from_dict(m) for m in data["ROOM_MOBS"]]
        if "ROOM_ITEMS" in data:
            self.state.room_items = [RoomItem.from_dict(i) for i in data["ROOM_ITEMS"]]
        if "FIGHTING" in data:
            was_fighting = self.state.fighting
            self.state.fighting = bool(data["FIGHTING"])
            if was_fighting and not self.state.fighting:
                log.info("Combat ended")
        if "INVENTORY" in data:
            self.state.inventory = data["INVENTORY"]
        if "EQUIPMENT" in data:
            self.state.equipment = data["EQUIPMENT"]
        if "EVENTS" in data:
            self.state.events = data["EVENTS"]
            for ev in self.state.events:
                await self.handle_event_entry(ev)

        # Detect full var dump (has many keys — initial dump after auth)
        if len(data) >= 5 and not self._vars_ready:
            self._vars_ready = True
            log.info("Full var dump received | room=%s mobs=%d items=%d",
                     self.state.room_vnum,
                     len(self.state.room_mobs),
                     len(self.state.room_items))

        # Death detection
        if self.state.health == 0 and not self._dead:
            self._dead = True
            self.state.reset_combat()
            log.info("Died. Respawned. Switching to recovery mode.")
            if self.circuit.record("died"):
                log.error("Circuit breaker tripped: died. Stopping.")
                self._shutdown = True

        # Low health flee
        if self.state.fighting and self.state.max_health > 0:
            if self.state.health < self.state.max_health * 0.25:
                log.info("Health critical (%d/%d) — fleeing", self.state.health, self.state.max_health)
                await self.send_command("flee")

    async def on_event(self, data: dict) -> None:
        """Game events (combat, enter, leave, say)."""
        etype = data.get("type", "")
        text = data.get("text", "")
        log.info("Event [%s]: %s", etype, text)

        if etype == "combat":
            if "died" in text.lower() or "dead" in text.lower():
                pass  # handled in on_vars via HEALTH==0

    async def on_text(self, data: dict) -> None:
        text = data.get("text", "")
        log.info("Text: %s", text)

    async def on_error(self, data: dict) -> None:
        msg = data.get("message", "")
        log.error("Server error: %s", msg)
        if self.circuit.record("error"):
            log.error("Circuit breaker tripped: error. Stopping.")
            self._shutdown = True

    async def handle_event_entry(self, ev: dict) -> None:
        """Process individual events from EVENTS array."""
        etype = ev.get("type", "")
        if etype == "rate_limited":
            log.warning("Rate limited on command: %s", ev.get("command", ""))
            if self.circuit.record("rate_limited"):
                log.error("Circuit breaker tripped: rate_limited. Stopping.")
                self._shutdown = True

    # ------------------------------------------------------------------
    # Bot actions
    # ------------------------------------------------------------------

    async def navigate(self) -> None:
        """Pick a random exit and move."""
        exits = self.state.room_exits
        if not exits:
            log.warning("No exits from room %s", self.state.room_vnum)
            if self.circuit.record("no_exits"):
                log.error("Circuit breaker tripped: no_exits. Stopping.")
                self._shutdown = True
            return

        direction = random.choice(exits)
        log.info("Moving %s from room %s", direction, self.state.room_vnum)
        await self.send_command(direction)

    async def attack(self) -> None:
        """Attack the first non-fighting mob in the room."""
        targets = [m for m in self.state.room_mobs if not m.fighting]
        if not targets:
            log.info("No attackable mobs in room %s", self.state.room_vnum)
            return

        target = targets[0]
        log.info("Attacking %s (target=%s)", target.name, target.target_string)
        await self.send_command("hit", [target.target_string])

    async def loot(self) -> None:
        """Pick up everything in the room."""
        items = list(self.state.room_items)
        if not items:
            return

        for item in items:
            log.info("Looting %s (target=%s)", item.name, item.target_string)
            await self.send_command("get", [item.target_string])

        # Inventory snapshot before loot isn't available server-side yet,
        # so we report what we tried to pick up.
        names = [i.name for i in items]
        if len(names) == 1:
            log.info("Picked up %s. Riveting.", names[0])
        else:
            log.info("Picked up %s. Thrilling.", ", ".join(names))

    # ------------------------------------------------------------------
    # State machine
    # ------------------------------------------------------------------

    async def run_cycle(self) -> None:
        """One decision cycle after vars are ready."""
        if self._dead:
            await self.handle_death()
            return

        if self._recovery_mode:
            # Move a few rooms away from respawn before fighting again
            log.info("Recovery mode — navigating away from respawn")
            await self.navigate()
            self._recovery_mode = False
            return

        if self.state.fighting:
            # Combat is autonomous — just wait for events
            return

        # Not fighting — check for loot first (items may be on floor from last kill)
        if self.state.room_items:
            await self.loot()
            # Give server a tick to process gets before moving on
            await asyncio.sleep(0.5)
            return

        # Check for mobs to fight
        attackable = [m for m in self.state.room_mobs if not m.fighting]
        if attackable:
            await self.attack()
            return

        # Nothing here — wander
        await self.navigate()

    async def handle_death(self) -> None:
        """Handle respawn and recovery after death."""
        self.state.reset_combat()
        await asyncio.sleep(3)  # respawn delay
        self._dead = False
        self._recovery_mode = True
        log.info("Recovery mode engaged")

    # ------------------------------------------------------------------
    # Main loop
    # ------------------------------------------------------------------

    async def run(self) -> None:
        while not self._shutdown:
            if not await self.connect():
                if not await self.reconnect():
                    continue
                if self._reconnect_attempt >= 5:
                    log.error("Max reconnect attempts reached. Giving up.")
                    break
                continue

            # Authenticate
            await self.send_login()

            # Wait for state or vars dump
            timeout = 10.0
            deadline = time.time() + timeout
            self._authenticated = False
            self._vars_ready = False

            while time.time() < deadline and not self._shutdown:
                try:
                    raw = await asyncio.wait_for(self.ws.recv(), timeout=1.0)
                except asyncio.TimeoutError:
                    continue
                except ConnectionClosed:
                    log.warning("Connection closed during auth")
                    if self.circuit.record("connection_lost"):
                        log.error("Circuit breaker tripped: connection_lost. Stopping.")
                        self._shutdown = True
                    break

                await self.handle_message(raw)

                if self._authenticated and not self.api_key:
                    # Human mode gets state message — subscribe now
                    await self.send_subscribe()
                    self._vars_ready = True
                    break
                elif self.api_key and self._vars_ready:
                    # Agent mode got full var dump
                    break

            if self._shutdown:
                break

            if not self._authenticated or not self._vars_ready:
                log.warning("Auth/vars timeout — reconnecting")
                await self.reconnect()
                continue

            # Ensure subscribed (agent mode may not have sent yet if dump arrived fast)
            if self.api_key:
                await self.send_subscribe()

            log.info("Bot ready | room=%s hp=%d/%d mobs=%d",
                     self.state.room_vnum,
                     self.state.health, self.state.max_health,
                     len(self.state.room_mobs))

            # Main game loop
            while not self._shutdown:
                try:
                    raw = await asyncio.wait_for(self.ws.recv(), timeout=0.5)
                    await self.handle_message(raw)
                except asyncio.TimeoutError:
                    # No message — good time to make a decision
                    if self._vars_ready:
                        await self.run_cycle()
                except ConnectionClosed:
                    log.warning("Connection lost mid-game")
                    if self.circuit.record("connection_lost"):
                        log.error("Circuit breaker tripped: connection_lost. Stopping.")
                        self._shutdown = True
                    break

            if not self._shutdown:
                await self.reconnect()

        log.info("Bot shutting down")
        await self.disconnect()


# ---------------------------------------------------------------------------
# CLI
# ---------------------------------------------------------------------------

def main() -> int:
    parser = argparse.ArgumentParser(description="Dark Pawns proof-of-concept agent")
    parser.add_argument("--host", default="192.168.1.106", help="WebSocket host")
    parser.add_argument("--port", type=int, default=4350, help="WebSocket port")
    parser.add_argument("--key", default=None, help="Agent API key (agent mode)")
    parser.add_argument("--name", default="dp_bot", help="Character name")
    parser.add_argument("--new", action="store_true", help="Create new character (human mode)")
    args = parser.parse_args()

    bot = DPBot(
        host=args.host,
        port=args.port,
        name=args.name,
        api_key=args.key,
        new_char=args.new,
    )

    try:
        asyncio.run(bot.run())
    except KeyboardInterrupt:
        log.info("Interrupted by user")
    return 0


if __name__ == "__main__":
    sys.exit(main())
