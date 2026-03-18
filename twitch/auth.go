package twitchirc

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/twitch"
)

// generateStateString generates a secure random string for OAuth2 state parameter.
func generateStateString() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

func (irc *IRC) parseAuthCode(w http.ResponseWriter, req *http.Request) {
	err := req.ParseForm()
	if err != nil {
		fmt.Printf("could not parse query: %v", err)
		http.Error(w, "could not parse query", http.StatusBadRequest)
		return
	}
	receivedState := req.FormValue("state")
	if receivedState != irc.oauthState {
		fmt.Printf("invalid oauth state, expected '%s' got '%s'\n", irc.oauthState, receivedState)
		http.Error(w, "Invalid OAuth state", http.StatusBadRequest)
		return
	}
	irc.authCode = req.FormValue("code")
}

// tryRefreshToken uses the stored refresh token to silently obtain a new access token.
func (irc *IRC) tryRefreshToken(ctx context.Context) error {
	if irc.tok == nil || irc.tok.RefreshToken == "" || irc.oauthConf == nil {
		return fmt.Errorf("no refresh token available")
	}
	ts := irc.oauthConf.TokenSource(ctx, irc.tok)
	newTok, err := ts.Token()
	if err != nil {
		return fmt.Errorf("refresh failed: %w", err)
	}
	irc.tok = newTok
	irc.tokenRefreshTime = time.Now()
	fmt.Printf("Token refreshed silently. New token received.\n")
	return nil
}

// AuthTwitch uses oauth2 protocol to retrieve oauth2 token for twitch IRC.
// Priority: TWITCH_REFRESH_TOKEN (silent refresh) → TWITCH_TOKEN (direct) → full OAuth flow.
// Safe to call multiple times; the HTTP handler is registered only once.
func (irc *IRC) AuthTwitch(ctx context.Context) error {
	// Build OAuth config — needed for refresh and/or full OAuth flow.
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
	if irc.oauthConf == nil {
		irc.oauthConf = conf
	}

	tokenStr := os.Getenv("TWITCH_TOKEN")
	refreshStr := os.Getenv("TWITCH_REFRESH_TOKEN")

	// If a refresh token is available, try silent refresh first.
	// This handles both initial startup and re-auth after expiry.
	if refreshStr != "" {
		fmt.Println("Attempting silent token refresh using TWITCH_REFRESH_TOKEN")
		irc.tok = &oauth2.Token{
			AccessToken:  tokenStr,
			RefreshToken: refreshStr,
		}
		if err := irc.tryRefreshToken(ctx); err == nil {
			return nil
		}
		fmt.Println("Silent refresh failed, falling back to other auth methods")
	}

	// Fall back to direct access token from environment.
	if tokenStr != "" {
		fmt.Println("Using TWITCH_TOKEN from environment")
		irc.tok = &oauth2.Token{
			AccessToken: tokenStr,
		}
		irc.tokenRefreshTime = time.Now()
		return nil
	}

	// No env-based token available — run interactive OAuth flow.
	fmt.Println("TWITCH_TOKEN not found, initiating OAuth flow...")
	fmt.Printf("OAuth redirect configured for: https://%s/oauth/redirect\n", redirectHost)

	// Register the redirect handler exactly once across all AuthTwitch calls.
	irc.handlerOnce.Do(func() {
		http.HandleFunc("/oauth/redirect", irc.parseAuthCode)
		go func() {
			_ = http.ListenAndServe(":3000", nil)
		}()
	})

	state, err := generateStateString()
	if err != nil {
		return fmt.Errorf("failed to generate oauth state: %w", err)
	}
	irc.oauthState = state

	// Reset authCode so a stale value from a previous flow doesn't skip the wait.
	irc.authCode = ""

	url := irc.oauthConf.AuthCodeURL(state, oauth2.AccessTypeOffline)
	fmt.Printf("Visit the URL for the auth dialog: %v\n", url)

	for irc.authCode == "" {
		time.Sleep(1 * time.Second)
	}

	fmt.Println("auth code received")
	irc.tok, err = irc.oauthConf.Exchange(ctx, irc.authCode)
	if err != nil {
		return fmt.Errorf("failed to get token with auth code: %w", err)
	}
	irc.tokenRefreshTime = time.Now()
	fmt.Printf("Token received: %s\n", irc.tok.AccessToken)
	fmt.Println("IMPORTANT: Save this token to 1Password as TWITCH_TOKEN to avoid OAuth flow on restart")
	if irc.tok.RefreshToken != "" {
		fmt.Printf("Refresh token received — save as TWITCH_REFRESH_TOKEN for automatic refresh\n")
	}
	return nil
}
