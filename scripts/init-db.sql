-- Initialize Dark Pawns database
CREATE TABLE IF NOT EXISTS agent_keys (
    id SERIAL PRIMARY KEY,
    player_name VARCHAR(255) NOT NULL UNIQUE,
    api_key VARCHAR(255) NOT NULL UNIQUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Insert default agent key for BRENDA69
INSERT INTO agent_keys (player_name, api_key)
VALUES ('brenda69', 'br3nd4-69-ag3nt-k3y-d3f4ult')
ON CONFLICT (player_name) DO NOTHING;

-- Create other necessary tables (simplified - actual schema would be more complex)
CREATE TABLE IF NOT EXISTS players (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL UNIQUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);