-- Migration: Add emotional tagging columns to agent_narrative_memory
-- Date: 2026-04-22
-- Author: BRENDA69

BEGIN;

-- Add emotional tagging columns
ALTER TABLE agent_narrative_memory 
ADD COLUMN IF NOT EXISTS emotion_category VARCHAR(10),
ADD COLUMN IF NOT EXISTS emotion_intensity INTEGER,
ADD COLUMN IF NOT EXISTS primary_emotions JSONB DEFAULT '[]',
ADD COLUMN IF NOT EXISTS emotion_confidence FLOAT;

-- Add constraints
ALTER TABLE agent_narrative_memory 
ADD CONSTRAINT emotion_category_check 
CHECK (emotion_category IN ('positive', 'negative', 'neutral') OR emotion_category IS NULL);

ALTER TABLE agent_narrative_memory 
ADD CONSTRAINT emotion_intensity_check 
CHECK (emotion_intensity BETWEEN 1 AND 5 OR emotion_intensity IS NULL);

ALTER TABLE agent_narrative_memory 
ADD CONSTRAINT emotion_confidence_check 
CHECK (emotion_confidence BETWEEN 0.0 AND 1.0 OR emotion_confidence IS NULL);

-- Create indexes for efficient querying
CREATE INDEX IF NOT EXISTS idx_anm_emotion_category ON agent_narrative_memory(emotion_category);
CREATE INDEX IF NOT EXISTS idx_anm_emotion_intensity ON agent_narrative_memory(emotion_intensity);
CREATE INDEX IF NOT EXISTS idx_anm_primary_emotions ON agent_narrative_memory USING GIN(primary_emotions);

-- Create composite index for common query patterns
CREATE INDEX IF NOT EXISTS idx_anm_emotion_query 
ON agent_narrative_memory(agent_name, emotion_category, emotion_intensity DESC, salience DESC);

-- Update comment on table
COMMENT ON TABLE agent_narrative_memory IS 'Agent narrative memories with emotional valence tagging (-3 to +3 scale) and emotional category tagging (positive/negative/neutral with intensity 1-5)';

-- Update comment on columns
COMMENT ON COLUMN agent_narrative_memory.emotion_category IS 'Emotional category: positive, negative, or neutral';
COMMENT ON COLUMN agent_narrative_memory.emotion_intensity IS 'Emotional intensity: 1 (subtle) to 5 (intense)';
COMMENT ON COLUMN agent_narrative_memory.primary_emotions IS 'Primary emotions: joy, anger, fear, sadness, surprise, disgust, trust, anticipation';
COMMENT ON COLUMN agent_narrative_memory.emotion_confidence IS 'Confidence in emotional tagging (0.0-1.0)';

-- Create view for emotional statistics
CREATE OR REPLACE VIEW agent_emotion_stats AS
SELECT 
    agent_name,
    COUNT(*) as total_memories,
    COUNT(emotion_category) as tagged_memories,
    ROUND(COUNT(emotion_category) * 100.0 / NULLIF(COUNT(*), 0), 2) as tagging_coverage_pct,
    
    -- Category distribution
    SUM(CASE WHEN emotion_category = 'positive' THEN 1 ELSE 0 END) as positive_count,
    SUM(CASE WHEN emotion_category = 'negative' THEN 1 ELSE 0 END) as negative_count,
    SUM(CASE WHEN emotion_category = 'neutral' THEN 1 ELSE 0 END) as neutral_count,
    
    -- Average intensity by category
    ROUND(AVG(CASE WHEN emotion_category = 'positive' THEN emotion_intensity END), 2) as avg_positive_intensity,
    ROUND(AVG(CASE WHEN emotion_category = 'negative' THEN emotion_intensity END), 2) as avg_negative_intensity,
    ROUND(AVG(CASE WHEN emotion_category = 'neutral' THEN emotion_intensity END), 2) as avg_neutral_intensity,
    
    -- Overall averages
    ROUND(AVG(emotion_intensity), 2) as avg_intensity,
    ROUND(AVG(emotion_confidence), 3) as avg_confidence,
    ROUND(AVG(valence), 2) as avg_valence,
    
    -- Most recent tagged memory
    MAX(CASE WHEN emotion_category IS NOT NULL THEN created_at END) as last_tagged_at
    
FROM agent_narrative_memory
GROUP BY agent_name;

COMMENT ON VIEW agent_emotion_stats IS 'Emotional statistics for each agent';

-- Create view for emotion timeline
CREATE OR REPLACE VIEW agent_emotion_timeline AS
SELECT 
    agent_name,
    DATE(created_at) as date,
    emotion_category,
    COUNT(*) as memory_count,
    ROUND(AVG(emotion_intensity), 2) as avg_intensity,
    ROUND(AVG(emotion_confidence), 3) as avg_confidence,
    ROUND(AVG(valence), 2) as avg_valence,
    STRING_AGG(DISTINCT event_type, ', ' ORDER BY event_type) as event_types,
    STRING_AGG(DISTINCT related_entity, ', ' FILTER (WHERE related_entity IS NOT NULL)) as related_entities
FROM agent_narrative_memory
WHERE emotion_category IS NOT NULL
GROUP BY agent_name, DATE(created_at), emotion_category
ORDER BY agent_name, date DESC, emotion_category;

COMMENT ON VIEW agent_emotion_timeline IS 'Daily emotional timeline for each agent';

-- Create function to get emotional summary
CREATE OR REPLACE FUNCTION get_agent_emotional_summary(
    p_agent_name VARCHAR(64),
    p_limit INTEGER DEFAULT 10
)
RETURNS TABLE(
    emotion_category VARCHAR(10),
    memory_count INTEGER,
    avg_intensity NUMERIC,
    avg_confidence NUMERIC,
    most_common_emotions TEXT[],
    example_summary TEXT
) AS $$
BEGIN
    RETURN QUERY
    WITH emotion_stats AS (
        SELECT 
            emotion_category,
            COUNT(*) as count,
            ROUND(AVG(emotion_intensity), 2) as avg_intensity,
            ROUND(AVG(emotion_confidence), 3) as avg_confidence,
            ARRAY(
                SELECT DISTINCT unnest(primary_emotions)
                FROM agent_narrative_memory m2
                WHERE m2.agent_name = p_agent_name 
                  AND m2.emotion_category = m.emotion_category
                  AND m2.primary_emotions IS NOT NULL
                LIMIT 5
            ) as common_emotions,
            (
                SELECT summary
                FROM agent_narrative_memory m2
                WHERE m2.agent_name = p_agent_name 
                  AND m2.emotion_category = m.emotion_category
                ORDER BY emotion_intensity DESC, salience DESC
                LIMIT 1
            ) as example
        FROM agent_narrative_memory m
        WHERE m.agent_name = p_agent_name
          AND m.emotion_category IS NOT NULL
        GROUP BY emotion_category
        ORDER BY count DESC
        LIMIT p_limit
    )
    SELECT 
        emotion_category,
        count::INTEGER,
        avg_intensity,
        avg_confidence,
        common_emotions,
        example
    FROM emotion_stats;
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION get_agent_emotional_summary IS 'Get emotional summary for an agent';

-- Create function to find memories by emotion
CREATE OR REPLACE FUNCTION find_memories_by_emotion(
    p_agent_name VARCHAR(64),
    p_category VARCHAR(10) DEFAULT NULL,
    p_min_intensity INTEGER DEFAULT 1,
    p_max_intensity INTEGER DEFAULT 5,
    p_emotions TEXT[] DEFAULT NULL,
    p_limit INTEGER DEFAULT 20
)
RETURNS TABLE(
    id BIGINT,
    summary TEXT,
    event_type VARCHAR(32),
    emotion_category VARCHAR(10),
    emotion_intensity INTEGER,
    primary_emotions JSONB,
    emotion_confidence FLOAT,
    valence INTEGER,
    salience FLOAT,
    created_at TIMESTAMPTZ
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        m.id,
        m.summary,
        m.event_type,
        m.emotion_category,
        m.emotion_intensity,
        m.primary_emotions,
        m.emotion_confidence,
        m.valence,
        m.salience,
        m.created_at
    FROM agent_narrative_memory m
    WHERE m.agent_name = p_agent_name
      AND m.emotion_category IS NOT NULL
      AND (p_category IS NULL OR m.emotion_category = p_category)
      AND m.emotion_intensity BETWEEN p_min_intensity AND p_max_intensity
      AND (p_emotions IS NULL OR m.primary_emotions ?| p_emotions)
    ORDER BY 
        m.emotion_intensity DESC,
        m.salience DESC,
        m.created_at DESC
    LIMIT p_limit;
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION find_memories_by_emotion IS 'Find memories by emotional criteria';

COMMIT;

-- Print migration summary
DO $$
BEGIN
    RAISE NOTICE 'Migration 001_add_emotional_tags completed successfully';
    RAISE NOTICE 'Added emotional tagging columns to agent_narrative_memory';
    RAISE NOTICE 'Created indexes for efficient querying';
    RAISE NOTICE 'Created views: agent_emotion_stats, agent_emotion_timeline';
    RAISE NOTICE 'Created functions: get_agent_emotional_summary, find_memories_by_emotion';
END $$;