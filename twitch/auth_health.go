package twitchirc

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/twitch"
)

// AuthHealthResponse represents the JSON response for the auth health check endpoint
type AuthHealthResponse struct {
	HasToken         bool      `json:"has_token"`
	LastRefreshTime  time.Time `json:"last_refresh_time"`
	ExpirationTime   time.Time `json:"expiration_time"`
	IsExpired        bool      `json:"is_expired"`
	HoursUntilExpiry float64   `json:"hours_until_expiry"`
}

const tokenExpiryDuration = 12 * time.Hour

// GetAuthHealth returns the current auth token health status
func (irc *IRC) GetAuthHealth() AuthHealthResponse {
	hasToken := irc.tok != nil && irc.tok.AccessToken != ""
	expirationTime := irc.tokenRefreshTime.Add(tokenExpiryDuration)
	hoursUntilExpiry := time.Until(expirationTime).Hours()
	isExpired := time.Now().After(expirationTime)

	return AuthHealthResponse{
		HasToken:         hasToken,
		LastRefreshTime:  irc.tokenRefreshTime,
		ExpirationTime:   expirationTime,
		IsExpired:        isExpired,
		HoursUntilExpiry: hoursUntilExpiry,
	}
}

// AuthHealthHandler returns an HTTP handler for the auth health check endpoint
func (irc *IRC) AuthHealthHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		health := irc.GetAuthHealth()

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		if err := json.NewEncoder(w).Encode(health); err != nil {
			irc.logger.Error("failed to encode auth health response", "error", err.Error())
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
	}
}

// GetOAuthURL generates the Twitch OAuth authorize URL for manual token refresh
func (irc *IRC) GetOAuthURL() (string, error) {
	redirectHost := os.Getenv("OAUTH_REDIRECT_HOST")
	if redirectHost == "" {
		redirectHost = "localhost:3000"
	}

	conf := &oauth2.Config{
		ClientID:     os.Getenv("TWITCH_ID"),
		ClientSecret: os.Getenv("TWITCH_SECRET"),
		Scopes:       []string{"chat:read", "chat:edit", "channel:moderate"},
		RedirectURL:  fmt.Sprintf("https://%s/oauth/redirect", redirectHost),
		Endpoint:     twitch.Endpoint,
	}

	state, err := generateStateString()
	if err != nil {
		return "", fmt.Errorf("failed to generate oauth state: %w", err)
	}

	return conf.AuthCodeURL(state, oauth2.AccessTypeOffline), nil
}

// TriggerRefreshHandler returns an HTTP handler for triggering token refresh
// Returns the OAuth URL that the user must visit to complete the refresh
func (irc *IRC) TriggerRefreshHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost && r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		oauthURL, err := irc.GetOAuthURL()
		if err != nil {
			irc.logger.Error("failed to generate OAuth URL", "error", err.Error())
			http.Error(w, "Failed to generate OAuth URL", http.StatusInternalServerError)
			return
		}

		irc.logger.Info("OAuth URL generated for token refresh", "oauth_url", oauthURL)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		resp := map[string]string{
			"oauth_url": oauthURL,
			"message":   "Visit the URL below to refresh your Twitch token. The bot will automatically pick up the new token after authorization.",
		}

		if err := json.NewEncoder(w).Encode(resp); err != nil {
			irc.logger.Error("failed to encode trigger refresh response", "error", err.Error())
		}
	}
}
