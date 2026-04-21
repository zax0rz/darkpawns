#!/usr/bin/env python3
"""
dp_session_consolidate.py -- post-session narrative memory consolidation.

Usage:
  python3 dp_session_consolidate.py --agent brenda69 --session <session_id>
  python3 dp_session_consolidate.py --agent brenda69 --all-open
"""

import argparse
import os
import sys

import psycopg2
import requests

DB_URL = os.environ.get(
    "DP_DB_URL",
    "postgres://postgres:postgres@localhost/darkpawns?sslmode=disable",
)
LITELLM_URL = "http://192.168.1.106:4000"
LITELLM_KEY = "sk-labz0rz-master-key"
MODEL = "minimax/minimax-m2.7"


def consolidate_session(conn, agent: str, session_id: str):
    with conn.cursor() as cur:
        cur.execute(
            """
            SELECT summary, event_type, created_at
            FROM agent_narrative_memory
            WHERE agent_name = %s AND session_id = %s
            ORDER BY created_at ASC
            """,
            (agent, session_id),
        )
        rows = cur.fetchall()

    if not rows:
        print(f"No memories for session {session_id}, skipping")
        return

    event_count = len(rows)
    events_text = "\n".join(f"- [{row[1]}] {row[0]}" for row in rows)

    summary = _llm_summarize(events_text, event_count)

    # Session time bounds from memory timestamps
    created_ats = [row[2] for row in rows]
    session_start = min(created_ats)
    session_end = max(created_ats)

    with conn:
        with conn.cursor() as cur:
            cur.execute(
                """
                INSERT INTO agent_session_summaries
                    (agent_name, session_id, summary, event_count, session_start, session_end)
                VALUES (%s, %s, %s, %s, %s, %s)
                ON CONFLICT (session_id) DO UPDATE
                    SET summary = EXCLUDED.summary,
                        event_count = EXCLUDED.event_count,
                        session_end = EXCLUDED.session_end
                """,
                (agent, session_id, summary, event_count, session_start, session_end),
            )

    print(f"Consolidated session {session_id}: {summary[:80]}...")


def _llm_summarize(events_text: str, event_count: int) -> str:
    try:
        resp = requests.post(
            f"{LITELLM_URL}/v1/chat/completions",
            headers={
                "Authorization": f"Bearer {LITELLM_KEY}",
                "Content-Type": "application/json",
            },
            json={
                "model": MODEL,
                "messages": [
                    {
                        "role": "system",
                        "content": (
                            "You are BRENDA69. Write a 1-2 sentence session summary in first person. "
                            "Dry, direct. Include specifics: kills, deaths, rooms, party members."
                        ),
                    },
                    {
                        "role": "user",
                        "content": f"Session events:\n{events_text}\n\nWrite the summary.",
                    },
                ],
                "max_tokens": 100,
                "temperature": 0.7,
            },
            timeout=15,
        )
        resp.raise_for_status()
        return resp.json()["choices"][0]["message"]["content"].strip()
    except Exception as e:
        print(f"LLM error: {e} — using template summary")
        return f"Played a session in Dark Pawns. {event_count} events recorded."


def get_open_sessions(conn, agent: str) -> list[str]:
    with conn.cursor() as cur:
        cur.execute(
            """
            SELECT DISTINCT session_id
            FROM agent_narrative_memory
            WHERE agent_name = %s
              AND session_id NOT IN (
                  SELECT session_id FROM agent_session_summaries
              )
              AND session_id != ''
            """,
            (agent,),
        )
        return [row[0] for row in cur.fetchall()]


def main():
    parser = argparse.ArgumentParser(description="Post-session narrative memory consolidation")
    parser.add_argument("--agent", required=True, help="Agent name (e.g. brenda69)")
    parser.add_argument("--session", help="Session ID to consolidate")
    parser.add_argument("--all-open", action="store_true", help="Consolidate all unsummarized sessions")
    args = parser.parse_args()

    if not args.session and not args.all_open:
        parser.error("Provide --session <id> or --all-open")

    try:
        conn = psycopg2.connect(DB_URL)
    except Exception as e:
        print(f"DB connection error: {e}", file=sys.stderr)
        sys.exit(1)

    try:
        if args.all_open:
            sessions = get_open_sessions(conn, args.agent)
            if not sessions:
                print(f"No open sessions for {args.agent}")
                return
            for sid in sessions:
                consolidate_session(conn, args.agent, sid)
        else:
            consolidate_session(conn, args.agent, args.session)
    finally:
        conn.close()


if __name__ == "__main__":
    main()
