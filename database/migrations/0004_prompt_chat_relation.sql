-- +goose Up

-- use pgvector to store embeddings of chat messages
drop table if exists twitch_chat_prompts;

alter table twitch_chat add column if not exists embedding vector(512);
alter table twitch_chat add column if not exists uuid uuid unique default gen_random_uuid();

alter table bot_response add column if not exists chat_id uuid; -- make sure this is a foreign key to twitch_chat.uuid

-- +goose Down
create table twitch_chat_prompts (
	id serial primary key,
	chats text[],
	created_at timestamptz default now()
);

alter table bot_response drop column chat_id;

alter table twitch_chat drop column embedding;
alter table twitch_chat drop column uuid;

