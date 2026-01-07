-- +goose Up
CREATE TABLE IF NOT EXISTS mod_actions (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at timestamptz NOT NULL DEFAULT NOW(),

    -- Trigger context
    trigger_message_id text,
    trigger_username text NOT NULL,
    trigger_message_content text,

    -- LLM decision
    llm_model text NOT NULL,
    llm_reasoning text,
    tool_call_name text NOT NULL,
    tool_call_params jsonb NOT NULL,

    -- Target
    target_username text,
    target_user_id text,

    -- Result
    twitch_api_response jsonb,
    success boolean NOT NULL,
    error_message text,

    -- Metadata
    channel_id text NOT NULL,
    channel_name text NOT NULL
);

CREATE INDEX idx_mod_actions_created_at ON mod_actions(created_at);
CREATE INDEX idx_mod_actions_target ON mod_actions(target_username);
CREATE INDEX idx_mod_actions_tool ON mod_actions(tool_call_name);
CREATE INDEX idx_mod_actions_channel ON mod_actions(channel_id);

-- +goose Down
DROP INDEX IF EXISTS idx_mod_actions_channel;
DROP INDEX IF EXISTS idx_mod_actions_tool;
DROP INDEX IF EXISTS idx_mod_actions_target;
DROP INDEX IF EXISTS idx_mod_actions_created_at;
DROP TABLE IF EXISTS mod_actions;
