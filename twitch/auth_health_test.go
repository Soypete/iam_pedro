package twitchirc

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Soypete/twitch-llm-bot/logging"
	"golang.org/x/oauth2"
)

func TestGetAuthHealth(t *testing.T) {
	tests := []struct {
		name             string
		token            *oauth2.Token
		tokenRefreshTime time.Time
		wantHasToken     bool
		wantIsExpired    bool
	}{
		{
			name: "valid token not expired",
			token: &oauth2.Token{
				AccessToken: "test-token",
			},
			tokenRefreshTime: time.Now().Add(-6 * time.Hour), // 6 hours ago
			wantHasToken:     true,
			wantIsExpired:    false,
		},
		{
			name: "token expired",
			token: &oauth2.Token{
				AccessToken: "test-token",
			},
			tokenRefreshTime: time.Now().Add(-13 * time.Hour), // 13 hours ago
			wantHasToken:     true,
			wantIsExpired:    true,
		},
		{
			name:             "no token",
			token:            nil,
			tokenRefreshTime: time.Time{},
			wantHasToken:     false,
			wantIsExpired:    true,
		},
		{
			name: "empty token",
			token: &oauth2.Token{
				AccessToken: "",
			},
			tokenRefreshTime: time.Now(),
			wantHasToken:     false,
			wantIsExpired:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := logging.NewLogger(logging.LogLevel("error"), nil)
			irc := &IRC{
				tok:              tt.token,
				tokenRefreshTime: tt.tokenRefreshTime,
				logger:           logger,
			}

			health := irc.GetAuthHealth()

			if health.HasToken != tt.wantHasToken {
				t.Errorf("HasToken = %v, want %v", health.HasToken, tt.wantHasToken)
			}

			if health.IsExpired != tt.wantIsExpired {
				t.Errorf("IsExpired = %v, want %v", health.IsExpired, tt.wantIsExpired)
			}

			// Verify expiration time is correctly calculated
			expectedExpiry := tt.tokenRefreshTime.Add(tokenExpiryDuration)
			if !health.ExpirationTime.Equal(expectedExpiry) {
				t.Errorf("ExpirationTime = %v, want %v", health.ExpirationTime, expectedExpiry)
			}
		})
	}
}

func TestAuthHealthHandler(t *testing.T) {
	tests := []struct {
		name               string
		method             string
		token              *oauth2.Token
		tokenRefreshTime   time.Time
		wantStatus         int
		wantHasToken       bool
		wantIsExpired      bool
		checkHoursUntilExp bool
	}{
		{
			name:   "GET request with valid token",
			method: http.MethodGet,
			token: &oauth2.Token{
				AccessToken: "test-token",
			},
			tokenRefreshTime:   time.Now().Add(-2 * time.Hour),
			wantStatus:         http.StatusOK,
			wantHasToken:       true,
			wantIsExpired:      false,
			checkHoursUntilExp: true,
		},
		{
			name:   "GET request with expired token",
			method: http.MethodGet,
			token: &oauth2.Token{
				AccessToken: "test-token",
			},
			tokenRefreshTime:   time.Now().Add(-13 * time.Hour),
			wantStatus:         http.StatusOK,
			wantHasToken:       true,
			wantIsExpired:      true,
			checkHoursUntilExp: false,
		},
		{
			name:             "POST request should fail",
			method:           http.MethodPost,
			token:            &oauth2.Token{AccessToken: "test-token"},
			tokenRefreshTime: time.Now(),
			wantStatus:       http.StatusMethodNotAllowed,
		},
		{
			name:             "PUT request should fail",
			method:           http.MethodPut,
			token:            &oauth2.Token{AccessToken: "test-token"},
			tokenRefreshTime: time.Now(),
			wantStatus:       http.StatusMethodNotAllowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := logging.NewLogger(logging.LogLevel("error"), nil)
			irc := &IRC{
				tok:              tt.token,
				tokenRefreshTime: tt.tokenRefreshTime,
				logger:           logger,
			}

			req := httptest.NewRequest(tt.method, "/healthz/auth", nil)
			w := httptest.NewRecorder()

			handler := irc.AuthHealthHandler()
			handler(w, req)

			resp := w.Result()
			defer resp.Body.Close()

			if resp.StatusCode != tt.wantStatus {
				t.Errorf("Status = %d, want %d", resp.StatusCode, tt.wantStatus)
			}

			// Only check JSON response for successful GET requests
			if tt.method == http.MethodGet && tt.wantStatus == http.StatusOK {
				var health AuthHealthResponse
				if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}

				if health.HasToken != tt.wantHasToken {
					t.Errorf("HasToken = %v, want %v", health.HasToken, tt.wantHasToken)
				}

				if health.IsExpired != tt.wantIsExpired {
					t.Errorf("IsExpired = %v, want %v", health.IsExpired, tt.wantIsExpired)
				}

				if tt.checkHoursUntilExp && health.HoursUntilExpiry <= 0 {
					t.Errorf("HoursUntilExpiry should be positive, got %f", health.HoursUntilExpiry)
				}

				// Verify content type
				contentType := resp.Header.Get("Content-Type")
				if contentType != "application/json" {
					t.Errorf("Content-Type = %s, want application/json", contentType)
				}
			}
		})
	}
}

func TestAuthHealthResponse200Status(t *testing.T) {
	logger := logging.NewLogger(logging.LogLevel("error"), nil)
	irc := &IRC{
		tok: &oauth2.Token{
			AccessToken: "valid-token",
		},
		tokenRefreshTime: time.Now(),
		logger:           logger,
	}

	req := httptest.NewRequest(http.MethodGet, "/healthz/auth", nil)
	w := httptest.NewRecorder()

	handler := irc.AuthHealthHandler()
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response AuthHealthResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Verify the payload structure
	if !response.HasToken {
		t.Error("Expected HasToken to be true")
	}

	if response.LastRefreshTime.IsZero() {
		t.Error("LastRefreshTime should not be zero")
	}

	if response.ExpirationTime.IsZero() {
		t.Error("ExpirationTime should not be zero")
	}
}

func TestAuthHealthResponseFailure(t *testing.T) {
	logger := logging.NewLogger(logging.LogLevel("error"), nil)
	irc := &IRC{
		tok:              nil, // No token
		tokenRefreshTime: time.Time{},
		logger:           logger,
	}

	req := httptest.NewRequest(http.MethodGet, "/healthz/auth", nil)
	w := httptest.NewRecorder()

	handler := irc.AuthHealthHandler()
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 even with no token, got %d", w.Code)
	}

	var response AuthHealthResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.HasToken {
		t.Error("Expected HasToken to be false when no token present")
	}

	if response.IsExpired != true {
		t.Error("Expected IsExpired to be true when no token present")
	}
}
