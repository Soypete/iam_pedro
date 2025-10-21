-- +goose Up
ALTER TABLE discord_ask_pedro ADD COLUMN IF NOT EXISTS message_id text;
ALTER TABLE discord_ask_pedro ADD COLUMN IF NOT EXISTS parent_message_id text;
ALTER TABLE discord_ask_pedro ADD COLUMN IF NOT EXISTS timestamp timestamptz;

-- +goose Down
ALTER TABLE discord_ask_pedro DROP COLUMN IF EXISTS message_id;
ALTER TABLE discord_ask_pedro DROP COLUMN IF EXISTS parent_message_id;
ALTER TABLE discord_ask_pedro DROP COLUMN IF EXISTS timestamp;
