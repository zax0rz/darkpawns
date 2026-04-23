-- Initialize moderation database for Dark Pawns

-- Admin users table
CREATE TABLE IF NOT EXISTS admin_users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(32) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    email VARCHAR(255),
    role VARCHAR(32) DEFAULT 'moderator', -- moderator, admin, superadmin
    permissions JSONB DEFAULT '[]',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_login TIMESTAMP,
    is_active BOOLEAN DEFAULT true
);

-- Abuse reports (extends the one in moderation package)
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
    resolution TEXT,
    severity INTEGER DEFAULT 1, -- 1=low, 2=medium, 3=high, 4=critical
    evidence JSONB DEFAULT '[]' -- URLs, chat logs, etc.
);

-- Admin actions log
CREATE TABLE IF NOT EXISTS admin_log (
    id SERIAL PRIMARY KEY,
    admin VARCHAR(32) NOT NULL,
    action VARCHAR(32) NOT NULL,
    target VARCHAR(32) NOT NULL,
    reason TEXT NOT NULL,
    duration INTERVAL,
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    ip_address VARCHAR(45),
    details JSONB DEFAULT '{}'
);

-- Player penalties
CREATE TABLE IF NOT EXISTS player_penalties (
    player_name VARCHAR(32) NOT NULL,
    penalty_type VARCHAR(32) NOT NULL,
    issued_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP,
    reason TEXT NOT NULL,
    issued_by VARCHAR(32) NOT NULL,
    PRIMARY KEY (player_name, penalty_type, issued_at)
);

-- Word filters
CREATE TABLE IF NOT EXISTS word_filters (
    id SERIAL PRIMARY KEY,
    pattern VARCHAR(255) NOT NULL,
    is_regex BOOLEAN DEFAULT false,
    action VARCHAR(32) NOT NULL,
    created_by VARCHAR(32) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    is_active BOOLEAN DEFAULT true
);

-- Chat logs for investigation
CREATE TABLE IF NOT EXISTS chat_logs (
    id SERIAL PRIMARY KEY,
    player_name VARCHAR(32) NOT NULL,
    message TEXT NOT NULL,
    room_vnum INTEGER,
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    filtered BOOLEAN DEFAULT false,
    filter_action VARCHAR(32)
);

-- Player notes for moderators
CREATE TABLE IF NOT EXISTS player_notes (
    id SERIAL PRIMARY KEY,
    player_name VARCHAR(32) NOT NULL,
    note TEXT NOT NULL,
    created_by VARCHAR(32) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    is_private BOOLEAN DEFAULT false
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_abuse_reports_status ON abuse_reports(status);
CREATE INDEX IF NOT EXISTS idx_abuse_reports_target ON abuse_reports(target);
CREATE INDEX IF NOT EXISTS idx_admin_log_target ON admin_log(target);
CREATE INDEX IF NOT EXISTS idx_admin_log_timestamp ON admin_log(timestamp);
CREATE INDEX IF NOT EXISTS idx_player_penalties_expires ON player_penalties(expires_at);
CREATE INDEX IF NOT EXISTS idx_player_penalties_player ON player_penalties(player_name);
CREATE INDEX IF NOT EXISTS idx_chat_logs_player_time ON chat_logs(player_name, timestamp);
CREATE INDEX IF NOT EXISTS idx_chat_logs_timestamp ON chat_logs(timestamp);

-- Insert default admin user (password: admin123)
INSERT INTO admin_users (username, password_hash, email, role) 
VALUES ('admin', '$2a$10$N9qo8uLOickgx2ZMRZoMye3.ZL6bK7v.7.6.6.6.6.6.6.6.6.6', 'admin@darkpawns.local', 'superadmin')
ON CONFLICT (username) DO NOTHING;

-- Insert some example word filters
INSERT INTO word_filters (pattern, is_regex, action, created_by) VALUES
('badword', false, 'censor', 'system'),
('(?i)hate.*speech', true, 'block', 'system'),
('(?i)cheat.*engine', true, 'block', 'system'),
('spamword', false, 'warn', 'system')
ON CONFLICT DO NOTHING;