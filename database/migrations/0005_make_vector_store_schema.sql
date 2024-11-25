-- +goose Up
CREATE SCHEMA vector_store;

-- +goose Down
DROP SCHEMA vector_store;

