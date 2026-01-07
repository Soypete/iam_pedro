package database

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Soypete/twitch-llm-bot/types"
	"github.com/google/uuid"
)

// ModActionWriter is the interface for writing moderation actions to the database
type ModActionWriter interface {
	InsertModAction(ctx context.Context, action types.ModAction) (uuid.UUID, error)
}

// InsertModAction inserts a moderation action into the database
func (p *Postgres) InsertModAction(ctx context.Context, action types.ModAction) (uuid.UUID, error) {
	p.logger.Debug("inserting mod action", "tool", action.ToolCallName, "target", action.TargetUsername)

	if action.ID == uuid.Nil {
		action.ID = uuid.New()
	}

	// Ensure ToolCallParams is not nil
	if action.ToolCallParams == nil {
		action.ToolCallParams = json.RawMessage("{}")
	}

	query := `
		INSERT INTO mod_actions (
			id,
			trigger_message_id,
			trigger_username,
			trigger_message_content,
			llm_model,
			llm_reasoning,
			tool_call_name,
			tool_call_params,
			target_username,
			target_user_id,
			twitch_api_response,
			success,
			error_message,
			channel_id,
			channel_name
		) VALUES (
			:id,
			:trigger_message_id,
			:trigger_username,
			:trigger_message_content,
			:llm_model,
			:llm_reasoning,
			:tool_call_name,
			:tool_call_params,
			:target_username,
			:target_user_id,
			:twitch_api_response,
			:success,
			:error_message,
			:channel_id,
			:channel_name
		)
	`

	_, err := p.connections.NamedExecContext(ctx, query, action)
	if err != nil {
		p.logger.Error("failed to insert mod action", "error", err.Error(), "tool", action.ToolCallName)
		return uuid.Nil, fmt.Errorf("failed to insert mod action: %w", err)
	}

	p.logger.Debug("mod action inserted successfully", "id", action.ID, "tool", action.ToolCallName)
	return action.ID, nil
}

// GetRecentModActions retrieves recent moderation actions for a user
func (p *Postgres) GetRecentModActions(ctx context.Context, username string, limit int) ([]types.ModAction, error) {
	p.logger.Debug("getting recent mod actions", "username", username, "limit", limit)

	query := `
		SELECT
			id, created_at, trigger_message_id, trigger_username, trigger_message_content,
			llm_model, llm_reasoning, tool_call_name, tool_call_params,
			target_username, target_user_id, twitch_api_response, success, error_message,
			channel_id, channel_name
		FROM mod_actions
		WHERE target_username = $1
		ORDER BY created_at DESC
		LIMIT $2
	`

	var actions []types.ModAction
	err := p.connections.SelectContext(ctx, &actions, query, username, limit)
	if err != nil {
		p.logger.Error("failed to get recent mod actions", "error", err.Error(), "username", username)
		return nil, fmt.Errorf("failed to get recent mod actions: %w", err)
	}

	return actions, nil
}

// GetUserModActionCount returns the count of moderation actions against a user in a time period
func (p *Postgres) GetUserModActionCount(ctx context.Context, username string, hoursBack int) (int, error) {
	p.logger.Debug("getting user mod action count", "username", username, "hoursBack", hoursBack)

	query := `
		SELECT COUNT(*)
		FROM mod_actions
		WHERE target_username = $1
		AND created_at > NOW() - INTERVAL '1 hour' * $2
		AND success = true
	`

	var count int
	err := p.connections.GetContext(ctx, &count, query, username, hoursBack)
	if err != nil {
		p.logger.Error("failed to get user mod action count", "error", err.Error(), "username", username)
		return 0, fmt.Errorf("failed to get user mod action count: %w", err)
	}

	return count, nil
}
