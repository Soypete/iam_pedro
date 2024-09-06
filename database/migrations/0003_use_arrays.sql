-- +goose Up
CREATE TABLE if not exists twitch_chat_prompts (
		id serial PRIMARY KEY,
		chats text[], -- array of twitch/youtube chat messages
		created_at timestamptz DEFAULT NOW()
		);

-- +goose Down

