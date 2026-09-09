-- Local-dev bootstrap only. The authoritative schema is managed by the
-- application's own migrations at startup (agent keys are stored HASHED in the
-- real schema, not in plaintext). Do NOT seed real or default credentials here:
-- the server rejects the old default key and example/test keys outright
-- (pkg/db/player.go -> ValidateAgentKey). Generate real keys via cmd/agentkeygen.
CREATE TABLE IF NOT EXISTS agent_keys (
    id SERIAL PRIMARY KEY,
    player_name VARCHAR(255) NOT NULL UNIQUE,
    api_key VARCHAR(255) NOT NULL UNIQUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create other necessary tables (simplified - actual schema would be more complex)
CREATE TABLE IF NOT EXISTS players (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL UNIQUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);