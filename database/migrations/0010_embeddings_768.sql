-- +goose Up
-- Resize embedding columns from the OpenAI dimensions (1536 / 512) to 768, the
-- dimension of nomic-embed-text-v1.5 served by the dedicated embeddings sidecar.
-- The chat server now runs MTP and no longer serves /v1/embeddings, so embeddings
-- moved to a separate server + model; the stored vector dimension follows it.
--
-- These columns are currently dormant/empty (FAQ semantic matching and the chat
-- message-embedding store are not on the live path), so dropping and re-adding the
-- columns is low-risk. Any vectors that do exist are invalidated by the model change
-- and must be re-embedded (faq/sync.go RegenerateAllEmbeddings, or a message re-embed
-- pass) — they were produced by a different model and are not comparable to 768-dim
-- nomic vectors.

-- faq_entries.embedding: 1536 -> 768
DROP INDEX IF EXISTS faq_entries_embedding_idx;
ALTER TABLE faq_entries DROP COLUMN IF EXISTS embedding;
ALTER TABLE faq_entries ADD COLUMN embedding vector(768);
CREATE INDEX IF NOT EXISTS faq_entries_embedding_idx
    ON faq_entries USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100);

-- twitch_chat.embedding: 512 -> 768
ALTER TABLE twitch_chat DROP COLUMN IF EXISTS embedding;
ALTER TABLE twitch_chat ADD COLUMN embedding vector(768);

-- +goose Down
-- Restore the previous dimensions (1536 for FAQ, 512 for chat messages).
DROP INDEX IF EXISTS faq_entries_embedding_idx;
ALTER TABLE faq_entries DROP COLUMN IF EXISTS embedding;
ALTER TABLE faq_entries ADD COLUMN embedding vector(1536);
CREATE INDEX IF NOT EXISTS faq_entries_embedding_idx
    ON faq_entries USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100);

ALTER TABLE twitch_chat DROP COLUMN IF EXISTS embedding;
ALTER TABLE twitch_chat ADD COLUMN embedding vector(512);
