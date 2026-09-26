-- The players table of a PostgreSQL Dark Pawns install, frozen as it stood when
-- the production conversion ran (2026-09-26).
--
-- Frozen on purpose. Production PostgreSQL is rollback authority: it is no longer
-- written, because the game runs on SQLite. A fixture that tracked the live
-- runtime's schema could not prove anything about a historical conversion, and
-- the runtime no longer speaks PostgreSQL at all.
CREATE TABLE IF NOT EXISTS players (
    id SERIAL PRIMARY KEY,
    name VARCHAR(32) UNIQUE NOT NULL,
    password_hash VARCHAR(255),
    room_vnum INTEGER DEFAULT 8004,
    level INTEGER DEFAULT 1,
    exp INTEGER DEFAULT 1,
    health INTEGER DEFAULT 10,
    max_health INTEGER DEFAULT 10,
    mana INTEGER DEFAULT 100,
    max_mana INTEGER DEFAULT 100,
    move INTEGER DEFAULT 100,
    max_move INTEGER DEFAULT 100,
    strength INTEGER DEFAULT 10,
    class INTEGER DEFAULT 3,
    race INTEGER DEFAULT 0,
    stat_str INTEGER DEFAULT 10,
    stat_int INTEGER DEFAULT 10,
    stat_wis INTEGER DEFAULT 10,
    stat_dex INTEGER DEFAULT 10,
    stat_con INTEGER DEFAULT 10,
    stat_cha INTEGER DEFAULT 10,
    hunger INTEGER DEFAULT 24,
    thirst INTEGER DEFAULT 24,
    drunk INTEGER DEFAULT 0,
    hometown INTEGER DEFAULT 0,
    olc_zone INTEGER DEFAULT 0,
    inventory JSON DEFAULT '[]',
    equipment JSON DEFAULT '{}',
    description TEXT DEFAULT '',
    title VARCHAR(80) DEFAULT '',
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);
-- Columns added to existing installs, applied idempotently at boot.
ALTER TABLE players ADD COLUMN IF NOT EXISTS strength INTEGER DEFAULT 10;
ALTER TABLE players ADD COLUMN IF NOT EXISTS class INTEGER DEFAULT 3;
ALTER TABLE players ADD COLUMN IF NOT EXISTS race INTEGER DEFAULT 0;
ALTER TABLE players ADD COLUMN IF NOT EXISTS stat_str INTEGER DEFAULT 10;
ALTER TABLE players ADD COLUMN IF NOT EXISTS stat_str_add INTEGER DEFAULT 0;
ALTER TABLE players ADD COLUMN IF NOT EXISTS stat_int INTEGER DEFAULT 10;
ALTER TABLE players ADD COLUMN IF NOT EXISTS stat_wis INTEGER DEFAULT 10;
ALTER TABLE players ADD COLUMN IF NOT EXISTS stat_dex INTEGER DEFAULT 10;
ALTER TABLE players ADD COLUMN IF NOT EXISTS stat_con INTEGER DEFAULT 10;
ALTER TABLE players ADD COLUMN IF NOT EXISTS stat_cha INTEGER DEFAULT 10;
ALTER TABLE players ADD COLUMN IF NOT EXISTS inventory JSON DEFAULT '[]';
ALTER TABLE players ADD COLUMN IF NOT EXISTS equipment JSON DEFAULT '{}';
ALTER TABLE players ADD COLUMN IF NOT EXISTS move INTEGER DEFAULT 100;
ALTER TABLE players ADD COLUMN IF NOT EXISTS max_move INTEGER DEFAULT 100;
ALTER TABLE players ADD COLUMN IF NOT EXISTS hunger INTEGER DEFAULT 24;
ALTER TABLE players ADD COLUMN IF NOT EXISTS thirst INTEGER DEFAULT 24;
ALTER TABLE players ADD COLUMN IF NOT EXISTS drunk INTEGER DEFAULT 0;
ALTER TABLE players ADD COLUMN IF NOT EXISTS is_admin BOOLEAN DEFAULT false;
ALTER TABLE players ADD COLUMN IF NOT EXISTS hometown INTEGER DEFAULT 0;
ALTER TABLE players ADD COLUMN IF NOT EXISTS olc_zone INTEGER DEFAULT 0;
ALTER TABLE players ADD COLUMN IF NOT EXISTS failed_login_attempts INTEGER DEFAULT 0;
ALTER TABLE players ADD COLUMN IF NOT EXISTS locked_until TIMESTAMPTZ;
ALTER TABLE players ADD COLUMN IF NOT EXISTS description TEXT DEFAULT '';
ALTER TABLE players ADD COLUMN IF NOT EXISTS title VARCHAR(80) DEFAULT '';
ALTER TABLE players ADD COLUMN IF NOT EXISTS character_data JSON DEFAULT '{}';
-- Indexes come after the migration columns that they reference.
CREATE UNIQUE INDEX IF NOT EXISTS players_name_folded_key ON players (lower(name));
CREATE INDEX IF NOT EXISTS idx_players_name ON players(name);
CREATE INDEX IF NOT EXISTS idx_players_locked_until ON players(locked_until);
