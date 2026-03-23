-- +goose Up

-- Create FAQ entries table for semantic search-based FAQ system
-- Uses pgvector for embedding storage and similarity search
CREATE TABLE IF NOT EXISTS faq_entries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- The canonical question/trigger phrase
    question TEXT NOT NULL,

    -- The cached response or link to return
    response TEXT NOT NULL,

    -- Optional: category for organization (youtube, social, schedule, etc.)
    category TEXT,

    -- Embedding vector (1536 dimensions for OpenAI text-embedding-3-small)
    embedding vector(1536),

    -- Whether this FAQ is currently active
    is_active BOOLEAN DEFAULT true,

    -- Cooldown in seconds before this FAQ can trigger again (global cooldown)
    cooldown_seconds INTEGER DEFAULT 300,

    -- Track when this was last triggered (for global cooldown)
    last_triggered_at TIMESTAMPTZ,

    -- Metadata
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Index for fast similarity search using cosine distance
-- IVFFlat index with 100 lists for efficient approximate nearest neighbor search
CREATE INDEX IF NOT EXISTS faq_entries_embedding_idx
    ON faq_entries USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100);

-- Index for active entries lookup
CREATE INDEX IF NOT EXISTS faq_entries_is_active_idx ON faq_entries (is_active);

-- Track per-user cooldowns for FAQ entries
CREATE TABLE IF NOT EXISTS faq_user_cooldowns (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    faq_id UUID NOT NULL REFERENCES faq_entries(id) ON DELETE CASCADE,
    user_id TEXT NOT NULL,  -- Twitch user ID
    triggered_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(faq_id, user_id)
);

-- Index for fast user cooldown lookups
CREATE INDEX IF NOT EXISTS faq_user_cooldowns_lookup_idx
    ON faq_user_cooldowns (faq_id, user_id, triggered_at);

-- Track FAQ response metrics
CREATE TABLE IF NOT EXISTS faq_responses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    faq_id UUID NOT NULL REFERENCES faq_entries(id) ON DELETE CASCADE,
    user_id TEXT NOT NULL,
    user_message TEXT NOT NULL,
    similarity_score FLOAT NOT NULL,
    response_sent TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Index for FAQ response analytics
CREATE INDEX IF NOT EXISTS faq_responses_faq_id_idx ON faq_responses (faq_id);
CREATE INDEX IF NOT EXISTS faq_responses_created_at_idx ON faq_responses (created_at);

-- +goose Down

DROP INDEX IF EXISTS faq_responses_created_at_idx;
DROP INDEX IF EXISTS faq_responses_faq_id_idx;
DROP TABLE IF EXISTS faq_responses;

DROP INDEX IF EXISTS faq_user_cooldowns_lookup_idx;
DROP TABLE IF EXISTS faq_user_cooldowns;

DROP INDEX IF EXISTS faq_entries_is_active_idx;
DROP INDEX IF EXISTS faq_entries_embedding_idx;
DROP TABLE IF EXISTS faq_entries;
