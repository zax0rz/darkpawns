-- The four moderation tables of a PostgreSQL Dark Pawns install, frozen as they
-- stood when the production conversion ran (2026-09-26). See the players fixture
-- for why this is frozen rather than generated.
CREATE TABLE IF NOT EXISTS abuse_reports (
    id SERIAL PRIMARY KEY,
    reporter VARCHAR(32) NOT NULL,
    target VARCHAR(32) NOT NULL,
    report_type VARCHAR(32) NOT NULL,
    description TEXT NOT NULL,
    room_vnum INTEGER DEFAULT 0,
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    status VARCHAR(32) DEFAULT 'pending',
    reviewed_by VARCHAR(32),
    reviewed_at TIMESTAMP,
    resolution TEXT
);
CREATE TABLE IF NOT EXISTS admin_log (
    id SERIAL PRIMARY KEY,
    admin VARCHAR(32) NOT NULL,
    action VARCHAR(32) NOT NULL,
    target VARCHAR(32) NOT NULL,
    reason TEXT NOT NULL,
    duration INTERVAL,
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    ip_address VARCHAR(45)
);
CREATE TABLE IF NOT EXISTS player_penalties (
    player_name VARCHAR(32) NOT NULL,
    penalty_type VARCHAR(32) NOT NULL,
    issued_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP,
    expired_at TIMESTAMP,
    status VARCHAR(32) DEFAULT 'active',
    reason TEXT NOT NULL,
    issued_by VARCHAR(32) NOT NULL,
    PRIMARY KEY (player_name, penalty_type, issued_at)
);
-- Columns added to existing installs, applied idempotently at boot.
ALTER TABLE player_penalties ADD COLUMN IF NOT EXISTS expired_at TIMESTAMP;
ALTER TABLE player_penalties ADD COLUMN IF NOT EXISTS status VARCHAR(32) DEFAULT 'active';
CREATE TABLE IF NOT EXISTS word_filters (
    id SERIAL PRIMARY KEY,
    pattern VARCHAR(255) NOT NULL,
    is_regex BOOLEAN DEFAULT false,
    action VARCHAR(32) NOT NULL,
    created_by VARCHAR(32) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
