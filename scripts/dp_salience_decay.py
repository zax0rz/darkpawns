#!/usr/bin/env python3
"""
dp_salience_decay.py -- nightly salience decay for agent narrative memory.
Mirrors DecayStaleMemories() from pkg/db/narrative_memory.go.
Run nightly: 0 2 * * * python3 /path/to/dp_salience_decay.py
"""

import os
import sys

import psycopg2

sys.path.insert(0, os.path.dirname(__file__))
from revenue.utils import send_telegram  # type: ignore

DB_URL = os.environ.get(
    "DP_DB_URL",
    "postgres://postgres:postgres@localhost/darkpawns?sslmode=disable",
)


def main():
    try:
        conn = psycopg2.connect(DB_URL)
    except Exception as e:
        print(f"DB connection error: {e}", file=sys.stderr)
        sys.exit(1)

    try:
        with conn:
            with conn.cursor() as cur:
                cur.execute("""
                    UPDATE agent_narrative_memory
                    SET salience = CASE
                            WHEN ABS(valence) >= 2 THEN salience * 0.75
                            ELSE salience * 0.5
                        END,
                        updated_at = NOW()
                    WHERE created_at < NOW() - INTERVAL '30 days'
                      AND salience > 0.05
                """)
                decayed = cur.rowcount

                cur.execute("""
                    DELETE FROM agent_narrative_memory WHERE salience <= 0.05
                """)
                pruned = cur.rowcount
    except Exception as e:
        print(f"Decay query error: {e}", file=sys.stderr)
        conn.close()
        sys.exit(1)
    finally:
        conn.close()

    print(f"Salience decay: {decayed} rows decayed, {pruned} rows pruned")
    send_telegram(f"[Dark Pawns] Salience decay: {decayed} decayed, {pruned} pruned")


if __name__ == "__main__":
    main()
