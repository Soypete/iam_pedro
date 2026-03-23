-- +goose Up
ALTER TABLE discord_twenty_questions_games RENAME COLUMN thread_timesout TO thread_timeout;

-- +goose Down
ALTER TABLE discord_twenty_questions_games RENAME COLUMN thread_timeout TO thread_timesout;
