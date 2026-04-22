#!/usr/bin/env python3
"""
dp_salience_decay.py — Nightly salience decay for agent_narrative_memory.

Runs nightly. Halves salience for memories older than 30 days.
High-valence memories (|valence| >= 2) decay at 75% the normal rate.
Prunes memories below salience threshold 0.05.

Source: PHASE4-AGENT-PROTOCOL.md Part 11 — Hosted Memory Tier
"""

import os
import sys
import psycopg2
import logging
from datetime import datetime, timezone

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s [dp_salience_decay] %(levelname)s %(message)s",
)
log = logging.getLogger(__name__)

DB_URL = os.environ.get(
    "DARKPAWNS_DB",
    "postgres://postgres:postgres@localhost/darkpawns?sslmode=disable",
)

PRUNE_THRESHOLD = 0.05   # memories below this are deleted
HALF_LIFE_DAYS = 30.0    # neutral memory half-life
HIGH_VALENCE_FACTOR = 0.75  # high-valence decays 25% slower


def run_decay(conn):
    with conn.cursor() as cur:
        # How many memories exist?
        cur.execute("SELECT COUNT(*) FROM agent_narrative_memory")
        total = cur.fetchone()[0]
        log.info(f"Total memories before decay: {total}")

        # Decay neutral memories (|valence| < 2): half-life 30 days
        # salience *= 0.5 ^ (days_since_update / 30)
        # Approximation: daily cron applies one day's worth of decay
        # 0.5^(1/30) ≈ 0.977 per day for neutral
        # 0.5^(0.75/30) ≈ 0.983 per day for high-valence
        neutral_factor = 0.5 ** (1.0 / HALF_LIFE_DAYS)
        high_val_factor = 0.5 ** (HIGH_VALENCE_FACTOR / HALF_LIFE_DAYS)

        cur.execute(
            """
            UPDATE agent_narrative_memory
            SET salience = salience * %s,
                updated_at = NOW()
            WHERE ABS(valence) < 2
              AND created_at < NOW() - INTERVAL '30 days'
            """,
            (neutral_factor,),
        )
        neutral_decayed = cur.rowcount
        log.info(f"Neutral memories decayed: {neutral_decayed} (factor={neutral_factor:.4f})")

        cur.execute(
            """
            UPDATE agent_narrative_memory
            SET salience = salience * %s,
                updated_at = NOW()
            WHERE ABS(valence) >= 2
              AND created_at < NOW() - INTERVAL '30 days'
            """,
            (high_val_factor,),
        )
        high_decayed = cur.rowcount
        log.info(f"High-valence memories decayed: {high_decayed} (factor={high_val_factor:.4f})")

        # Prune below threshold
        cur.execute(
            """
            DELETE FROM agent_narrative_memory
            WHERE salience < %s
            """,
            (PRUNE_THRESHOLD,),
        )
        pruned = cur.rowcount
        log.info(f"Pruned memories below {PRUNE_THRESHOLD}: {pruned}")

        cur.execute("SELECT COUNT(*) FROM agent_narrative_memory")
        remaining = cur.fetchone()[0]
        log.info(f"Total memories after decay: {remaining}")

    conn.commit()
    return {
        "total_before": total,
        "neutral_decayed": neutral_decayed,
        "high_valence_decayed": high_decayed,
        "pruned": pruned,
        "total_after": remaining,
    }


def main():
    log.info("Starting salience decay run")
    try:
        conn = psycopg2.connect(DB_URL)
    except Exception as e:
        log.error(f"DB connection failed: {e}")
        sys.exit(1)

    try:
        result = run_decay(conn)
        log.info(f"Decay complete: {result}")
    except Exception as e:
        log.error(f"Decay failed: {e}")
        conn.rollback()
        sys.exit(1)
    finally:
        conn.close()


if __name__ == "__main__":
    main()
