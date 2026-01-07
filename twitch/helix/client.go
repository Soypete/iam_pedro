// Package helix provides a client for the Twitch Helix API moderation endpoints
package helix

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/Soypete/twitch-llm-bot/logging"
)

const (
	baseURL = "https://api.twitch.tv/helix"
)

// Client is a Twitch Helix API client for moderation actions
type Client struct {
	httpClient    *http.Client
	clientID      string
	accessToken   string
	broadcasterID string
	moderatorID   string
	logger        *logging.Logger
}

// NewClient creates a new Twitch Helix API client
func NewClient(clientID, accessToken, broadcasterID, moderatorID string, logger *logging.Logger) *Client {
	if logger == nil {
		logger = logging.Default()
	}
	return &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		clientID:      clientID,
		accessToken:   accessToken,
		broadcasterID: broadcasterID,
		moderatorID:   moderatorID,
		logger:        logger,
	}
}

// UpdateToken updates the access token (for token refresh scenarios)
func (c *Client) UpdateToken(accessToken string) {
	c.accessToken = accessToken
}

// doRequest performs an HTTP request to the Twitch API
func (c *Client) doRequest(ctx context.Context, method, endpoint string, query url.Values, body interface{}) ([]byte, error) {
	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	fullURL := baseURL + endpoint
	if len(query) > 0 {
		fullURL += "?" + query.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, method, fullURL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.accessToken)
	req.Header.Set("Client-Id", c.clientID)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	c.logger.Debug("making Twitch API request", "method", method, "endpoint", endpoint)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		c.logger.Error("Twitch API error", "status", resp.StatusCode, "body", string(respBody))
		return respBody, fmt.Errorf("API error: status %d, body: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// BanUser bans or times out a user
// If duration is 0, it's a permanent ban; otherwise it's a timeout
func (c *Client) BanUser(ctx context.Context, userID string, duration int, reason string) ([]byte, error) {
	query := url.Values{}
	query.Set("broadcaster_id", c.broadcasterID)
	query.Set("moderator_id", c.moderatorID)

	body := map[string]interface{}{
		"data": map[string]interface{}{
			"user_id": userID,
			"reason":  reason,
		},
	}

	if duration > 0 {
		body["data"].(map[string]interface{})["duration"] = duration
	}

	return c.doRequest(ctx, http.MethodPost, "/moderation/bans", query, body)
}

// UnbanUser removes a ban or timeout from a user
func (c *Client) UnbanUser(ctx context.Context, userID string) ([]byte, error) {
	query := url.Values{}
	query.Set("broadcaster_id", c.broadcasterID)
	query.Set("moderator_id", c.moderatorID)
	query.Set("user_id", userID)

	return c.doRequest(ctx, http.MethodDelete, "/moderation/bans", query, nil)
}

// DeleteMessage deletes a specific chat message
func (c *Client) DeleteMessage(ctx context.Context, messageID string) ([]byte, error) {
	query := url.Values{}
	query.Set("broadcaster_id", c.broadcasterID)
	query.Set("moderator_id", c.moderatorID)
	query.Set("message_id", messageID)

	return c.doRequest(ctx, http.MethodDelete, "/moderation/chat", query, nil)
}

// ClearChat clears all messages in chat
func (c *Client) ClearChat(ctx context.Context) ([]byte, error) {
	query := url.Values{}
	query.Set("broadcaster_id", c.broadcasterID)
	query.Set("moderator_id", c.moderatorID)

	return c.doRequest(ctx, http.MethodDelete, "/moderation/chat", query, nil)
}

// ChatSettings represents the chat settings for a channel
type ChatSettings struct {
	EmoteMode              *bool `json:"emote_mode,omitempty"`
	FollowerMode           *bool `json:"follower_mode,omitempty"`
	FollowerModeDuration   *int  `json:"follower_mode_duration,omitempty"`
	SlowMode               *bool `json:"slow_mode,omitempty"`
	SlowModeWaitTime       *int  `json:"slow_mode_wait_time,omitempty"`
	SubscriberMode         *bool `json:"subscriber_mode,omitempty"`
	UniqueChatMode         *bool `json:"unique_chat_mode,omitempty"`
	NonModeratorChatDelay  *bool `json:"non_moderator_chat_delay,omitempty"`
	NonModChatDelaySeconds *int  `json:"non_moderator_chat_delay_duration,omitempty"`
}

// UpdateChatSettings updates the chat settings for the channel
func (c *Client) UpdateChatSettings(ctx context.Context, settings ChatSettings) ([]byte, error) {
	query := url.Values{}
	query.Set("broadcaster_id", c.broadcasterID)
	query.Set("moderator_id", c.moderatorID)

	return c.doRequest(ctx, http.MethodPatch, "/chat/settings", query, settings)
}

// AddModerator adds a user as a moderator
func (c *Client) AddModerator(ctx context.Context, userID string) ([]byte, error) {
	query := url.Values{}
	query.Set("broadcaster_id", c.broadcasterID)
	query.Set("user_id", userID)

	return c.doRequest(ctx, http.MethodPost, "/moderation/moderators", query, nil)
}

// RemoveModerator removes a user as a moderator
func (c *Client) RemoveModerator(ctx context.Context, userID string) ([]byte, error) {
	query := url.Values{}
	query.Set("broadcaster_id", c.broadcasterID)
	query.Set("user_id", userID)

	return c.doRequest(ctx, http.MethodDelete, "/moderation/moderators", query, nil)
}

// AddVIP adds VIP status to a user
func (c *Client) AddVIP(ctx context.Context, userID string) ([]byte, error) {
	query := url.Values{}
	query.Set("broadcaster_id", c.broadcasterID)
	query.Set("user_id", userID)

	return c.doRequest(ctx, http.MethodPost, "/channels/vips", query, nil)
}

// RemoveVIP removes VIP status from a user
func (c *Client) RemoveVIP(ctx context.Context, userID string) ([]byte, error) {
	query := url.Values{}
	query.Set("broadcaster_id", c.broadcasterID)
	query.Set("user_id", userID)

	return c.doRequest(ctx, http.MethodDelete, "/channels/vips", query, nil)
}

// PollChoice represents a choice in a poll
type PollChoice struct {
	Title string `json:"title"`
}

// CreatePollRequest represents the request body for creating a poll
type CreatePollRequest struct {
	BroadcasterID              string       `json:"broadcaster_id"`
	Title                      string       `json:"title"`
	Choices                    []PollChoice `json:"choices"`
	Duration                   int          `json:"duration"`
	ChannelPointsVotingEnabled bool         `json:"channel_points_voting_enabled,omitempty"`
	ChannelPointsPerVote       int          `json:"channel_points_per_vote,omitempty"`
}

// CreatePoll creates a new poll (requires broadcaster token)
func (c *Client) CreatePoll(ctx context.Context, title string, choices []string, duration int) ([]byte, error) {
	pollChoices := make([]PollChoice, len(choices))
	for i, choice := range choices {
		pollChoices[i] = PollChoice{Title: choice}
	}

	body := CreatePollRequest{
		BroadcasterID: c.broadcasterID,
		Title:         title,
		Choices:       pollChoices,
		Duration:      duration,
	}

	return c.doRequest(ctx, http.MethodPost, "/polls", nil, body)
}

// EndPoll ends an active poll
func (c *Client) EndPoll(ctx context.Context, pollID, status string) ([]byte, error) {
	query := url.Values{}
	query.Set("broadcaster_id", c.broadcasterID)
	query.Set("id", pollID)
	query.Set("status", status) // TERMINATED or ARCHIVED

	return c.doRequest(ctx, http.MethodPatch, "/polls", query, nil)
}

// PredictionOutcome represents an outcome in a prediction
type PredictionOutcome struct {
	Title string `json:"title"`
}

// CreatePredictionRequest represents the request body for creating a prediction
type CreatePredictionRequest struct {
	BroadcasterID    string              `json:"broadcaster_id"`
	Title            string              `json:"title"`
	Outcomes         []PredictionOutcome `json:"outcomes"`
	PredictionWindow int                 `json:"prediction_window"`
}

// CreatePrediction creates a new prediction (requires broadcaster token)
func (c *Client) CreatePrediction(ctx context.Context, title string, outcomes []string, duration int) ([]byte, error) {
	predictionOutcomes := make([]PredictionOutcome, len(outcomes))
	for i, outcome := range outcomes {
		predictionOutcomes[i] = PredictionOutcome{Title: outcome}
	}

	body := CreatePredictionRequest{
		BroadcasterID:    c.broadcasterID,
		Title:            title,
		Outcomes:         predictionOutcomes,
		PredictionWindow: duration,
	}

	return c.doRequest(ctx, http.MethodPost, "/predictions", nil, body)
}

// ResolvePrediction resolves a prediction with a winning outcome
func (c *Client) ResolvePrediction(ctx context.Context, predictionID, winningOutcomeID string) ([]byte, error) {
	query := url.Values{}
	query.Set("broadcaster_id", c.broadcasterID)
	query.Set("id", predictionID)
	query.Set("status", "RESOLVED")
	query.Set("winning_outcome_id", winningOutcomeID)

	return c.doRequest(ctx, http.MethodPatch, "/predictions", query, nil)
}

// CancelPrediction cancels a prediction and refunds points
func (c *Client) CancelPrediction(ctx context.Context, predictionID string) ([]byte, error) {
	query := url.Values{}
	query.Set("broadcaster_id", c.broadcasterID)
	query.Set("id", predictionID)
	query.Set("status", "CANCELED")

	return c.doRequest(ctx, http.MethodPatch, "/predictions", query, nil)
}

// SendAnnouncementRequest represents the request body for sending an announcement
type SendAnnouncementRequest struct {
	Message string `json:"message"`
	Color   string `json:"color,omitempty"` // blue, green, orange, purple, primary
}

// SendAnnouncement sends a chat announcement
func (c *Client) SendAnnouncement(ctx context.Context, message, color string) ([]byte, error) {
	query := url.Values{}
	query.Set("broadcaster_id", c.broadcasterID)
	query.Set("moderator_id", c.moderatorID)

	body := SendAnnouncementRequest{
		Message: message,
		Color:   color,
	}

	return c.doRequest(ctx, http.MethodPost, "/chat/announcements", query, body)
}

// SendShoutout sends a shoutout to another broadcaster
func (c *Client) SendShoutout(ctx context.Context, toBroadcasterID string) ([]byte, error) {
	query := url.Values{}
	query.Set("from_broadcaster_id", c.broadcasterID)
	query.Set("to_broadcaster_id", toBroadcasterID)
	query.Set("moderator_id", c.moderatorID)

	return c.doRequest(ctx, http.MethodPost, "/chat/shoutouts", query, nil)
}

// SetShieldMode enables or disables shield mode
func (c *Client) SetShieldMode(ctx context.Context, isActive bool) ([]byte, error) {
	query := url.Values{}
	query.Set("broadcaster_id", c.broadcasterID)
	query.Set("moderator_id", c.moderatorID)

	body := map[string]bool{
		"is_active": isActive,
	}

	return c.doRequest(ctx, http.MethodPut, "/moderation/shield_mode", query, body)
}

// GetUsers retrieves user information by login name
func (c *Client) GetUsers(ctx context.Context, logins []string) ([]byte, error) {
	query := url.Values{}
	for _, login := range logins {
		query.Add("login", login)
	}

	return c.doRequest(ctx, http.MethodGet, "/users", query, nil)
}

// UserResponse represents the response from the Get Users endpoint
type UserResponse struct {
	Data []UserData `json:"data"`
}

// UserData represents a user from the Twitch API
type UserData struct {
	ID              string `json:"id"`
	Login           string `json:"login"`
	DisplayName     string `json:"display_name"`
	Type            string `json:"type"`
	BroadcasterType string `json:"broadcaster_type"`
	Description     string `json:"description"`
	ProfileImageURL string `json:"profile_image_url"`
	OfflineImageURL string `json:"offline_image_url"`
	ViewCount       int    `json:"view_count"`
	Email           string `json:"email,omitempty"`
	CreatedAt       string `json:"created_at"`
}

// GetUserIDByLogin retrieves a user's ID by their login name
func (c *Client) GetUserIDByLogin(ctx context.Context, login string) (string, error) {
	respBody, err := c.GetUsers(ctx, []string{login})
	if err != nil {
		return "", err
	}

	var resp UserResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return "", fmt.Errorf("failed to parse user response: %w", err)
	}

	if len(resp.Data) == 0 {
		return "", fmt.Errorf("user not found: %s", login)
	}

	return resp.Data[0].ID, nil
}
