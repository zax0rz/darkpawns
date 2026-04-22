#!/usr/bin/env python3
"""
dp_session_consolidate.py — End-of-session narrative consolidation for BRENDA69.

Queries yesterday's agent_narrative_memory writes, generates a compact session
summary via LLM (falls back to template when LLM fails), and stores it as a
SESSION_SUMMARY memory with its own salience.

Run nightly after sessions end. Designed for BRENDA but works for any agent.

Source: PHASE4-AGENT-PROTOCOL.md Part 11 — Hosted Memory Tier
"""

import os
import sys
import json
import psycopg2
import requests
import logging
from datetime import datetime, timezone, timedelta

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s [dp_consolidate] %(levelname)s %(message)s",
)
log = logging.getLogger(__name__)

DB_URL = os.environ.get(
    "DARKPAWNS_DB",
    "postgres://postgres:postgres@localhost/darkpawns?sslmode=disable",
)
LITELLM_URL = os.environ.get("LITELLM_URL", "http://192.168.1.106:4000")
LITELLM_KEY = os.environ.get("LITELLM_KEY", "sk-labz0rz-master-key")
CONSOLIDATION_MODEL = os.environ.get("CONSOLIDATION_MODEL", "deepseek-chat")

# Agent to consolidate (BRENDA's character ID in the DB)
AGENT_NAME = os.environ.get("CONSOLIDATION_AGENT", "brenda69")


def fetch_recent_memories(conn, agent_name: str, hours: int = 24):
    """Fetch memories written in the last N hours for this agent."""
    with conn.cursor() as cur:
        cur.execute(
            """
            SELECT m.id, m.event_type, m.description, m.valence, m.salience,
                   m.related_entity, m.room_vnum, m.created_at
            FROM agent_narrative_memory m
            JOIN players p ON p.id = m.agent_character_id
            WHERE p.name ILIKE %s
              AND m.created_at > NOW() - INTERVAL '%s hours'
            ORDER BY m.created_at ASC
            """,
            (agent_name, hours),
        )
        rows = cur.fetchall()
    return rows


def template_summary(memories, agent_name: str) -> str:
    """Fallback: structured summary without LLM."""
    kills = [m for m in memories if m[1] == "MOB_KILL"]
    deaths = [m for m in memories if m[1] == "PLAYER_DEATH"]
    other = [m for m in memories if m[1] not in ("MOB_KILL", "PLAYER_DEATH")]

    parts = []
    if kills:
        parts.append(f"{len(kills)} kill(s)")
    if deaths:
        parts.append(f"{len(deaths)} death(s)")
    if other:
        parts.append(f"{len(other)} other event(s)")

    summary = f"Session summary for {agent_name}: " + (", ".join(parts) if parts else "no recorded events") + "."

    if kills:
        kill_descs = [m[2][:60] for m in kills[:3]]
        summary += " Notable kills: " + "; ".join(kill_descs) + "."
    if deaths:
        death_descs = [m[2][:60] for m in deaths[:2]]
        summary += " Deaths: " + "; ".join(death_descs) + "."

    return summary


def llm_summary(memories, agent_name: str) -> str | None:
    """Generate a narrative session summary via LLM."""
    if not memories:
        return None

    memory_lines = []
    for m in memories:
        ts = m[7].strftime("%H:%M") if m[7] else "?"
        memory_lines.append(f"[{ts}] {m[1]}: {m[2]} (valence={m[3]}, salience={m[4]:.2f})")

    prompt = f"""You are writing a compact session summary for {agent_name}, an AI agent playing a MUD.

Recent session events (chronological):
{chr(10).join(memory_lines)}

Write a 2-3 sentence narrative summary in first person as {agent_name}. 
Be dry, specific, and honest. Reference actual events. No fluff.
This will be injected into future sessions as CHARACTER HISTORY context."""

    try:
        resp = requests.post(
            f"{LITELLM_URL}/chat/completions",
            headers={"Authorization": f"Bearer {LITELLM_KEY}"},
            json={
                "model": CONSOLIDATION_MODEL,
                "messages": [{"role": "user", "content": prompt}],
                "max_tokens": 200,
                "temperature": 0.7,
            },
            timeout=30,
        )
        resp.raise_for_status()
        data = resp.json()
        return data["choices"][0]["message"]["content"].strip()
    except Exception as e:
        log.warning(f"LLM consolidation failed, using template: {e}")
        return None


def store_summary(conn, agent_name: str, summary: str, memory_count: int):
    """Store the session summary as a SESSION_SUMMARY memory."""
    with conn.cursor() as cur:
        # Get agent character ID
        cur.execute("SELECT id FROM players WHERE name ILIKE %s LIMIT 1", (agent_name,))
        row = cur.fetchone()
        if not row:
            log.warning(f"No player found for agent '{agent_name}' — skipping summary storage")
            return

        agent_id = row[0]

        # Salience based on memory count — more events = more significant session
        salience = min(0.3 + (memory_count * 0.05), 0.8)

        cur.execute(
            """
            INSERT INTO agent_narrative_memory
                (agent_character_id, event_type, description, valence, salience, created_at, updated_at)
            VALUES (%s, 'SESSION_SUMMARY', %s, 0, %s, NOW(), NOW())
            """,
            (agent_id, summary, salience),
        )
        log.info(f"Stored SESSION_SUMMARY for {agent_name} (salience={salience:.2f})")
    conn.commit()


def main():
    log.info(f"Starting session consolidation for agent: {AGENT_NAME}")

    try:
        conn = psycopg2.connect(DB_URL)
    except Exception as e:
        log.error(f"DB connection failed: {e}")
        sys.exit(1)

    try:
        memories = fetch_recent_memories(conn, AGENT_NAME, hours=24)
        log.info(f"Found {len(memories)} memories from last 24h")

        if not memories:
            log.info("No memories to consolidate — skipping")
            return

        # Try LLM first, fall back to template
        summary = llm_summary(memories, AGENT_NAME)
        if not summary:
            summary = template_summary(memories, AGENT_NAME)
            log.info("Used template summary (LLM unavailable)")
        else:
            log.info("Used LLM summary")

        log.info(f"Summary: {summary}")
        store_summary(conn, AGENT_NAME, summary, len(memories))

    except Exception as e:
        log.error(f"Consolidation failed: {e}")
        conn.rollback()
        sys.exit(1)
    finally:
        conn.close()

    log.info("Consolidation complete")


if __name__ == "__main__":
    main()
