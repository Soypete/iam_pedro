-- +goose Up
CREATE TABLE IF NOT EXISTS discord_ask_pedro (
		id serial PRIMARY KEY,
		username text,
		message text,
		thread_id text unique, 
		thread_timeout int, -- time in seconds before the thread times out
		is_form_pedro BOOLEAN,
		created_at timestamptz DEFAULT NOW()
		);

CREATE TABLE IF NOT EXISTS discord_twenty_questions (
		id serial PRIMARY KEY,
		game_id uuid references discord_twenty_questions_games(game_id),
		question text,
		response text,
		created_at timestamptz DEFAULT NOW()
		);

CREATE TYPE IF NOT EXISTS game_status AS ENUM ('started', 'in_progress', 'ended', 'abandoned');

CREATE TABLE IF NOT EXISTS discord_twenty_questions_games (
		game_id uuid PRIMARY KEY,
		answer text, -- the thing that the user is trying to guess
		question_count int,
		status game_status, 
		thread_id text,
		thread_timesout int, -- time in seconds before the thread times out
		username text,
		created_at timestamptz DEFAULT NOW()
		);

-- +goose Down
DROP TABLE IF EXISTS discord_ask_pedro;
DROP TABLE IF EXISTS discord_twenty_questions;
DROP TABLE IF EXISTS discord_twenty_questions_games;
DROP TYPE IF EXISTS game_status;


