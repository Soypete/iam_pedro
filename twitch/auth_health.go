package twitchirc

import (
	"encoding/json"
	"net/http"
	"time"
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
