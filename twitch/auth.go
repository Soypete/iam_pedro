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

// AuthTwitch use oauth2 protocol to retrieve oauth2 token for twitch IRC.
// First checks for TWITCH_TOKEN env var. If not found, runs OAuth flow.
// generateStateString generates a secure random string for OAuth2 state parameter.
func generateStateString() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

func (irc *IRC) AuthTwitch(ctx context.Context) error {
	// Check if token is already available via environment variable
	if tokenStr := os.Getenv("TWITCH_TOKEN"); tokenStr != "" {
		fmt.Println("Using TWITCH_TOKEN from environment")
		irc.tok = &oauth2.Token{
			AccessToken: tokenStr,
		}
		return nil
	}

	// Fall back to OAuth flow
	fmt.Println("TWITCH_TOKEN not found, initiating OAuth flow...")

	// Determine redirect host (defaults to localhost for local dev)
	redirectHost := os.Getenv("OAUTH_REDIRECT_HOST")
	if redirectHost == "" {
		redirectHost = "localhost:3000"
	}

	// Determine protocol based on redirect host
	// Use HTTPS for Tailscale domains, HTTP for localhost
	protocol := "http"
	if redirectHost != "localhost:3000" && redirectHost != "127.0.0.1:3000" {
		protocol = "https"
	}

	http.HandleFunc("/oauth/redirect", irc.parseAuthCode)
	go func() {
		_ = http.ListenAndServe(":3000", nil)
	}()

	conf := &oauth2.Config{
		ClientID:     os.Getenv("TWITCH_ID"),
		ClientSecret: os.Getenv("TWITCH_SECRET"),
		Scopes:       []string{"chat:read", "chat:edit", "channel:moderate"},
		RedirectURL:  fmt.Sprintf("%s://%s/oauth/redirect", protocol, redirectHost),
		Endpoint:     twitch.Endpoint,
	}

	// Redirect user to consent page to ask for permission
	state, err := generateStateString()
	if err != nil {
		return fmt.Errorf("failed to generate oauth state: %w", err)
	}
	irc.oauthState = state
	url := conf.AuthCodeURL(state, oauth2.AccessTypeOffline)
	fmt.Printf("Visit the URL for the auth dialog: %v\n", url)
	fmt.Printf("OAuth redirect configured for: %s://%s/oauth/redirect\n", protocol, redirectHost)

	for irc.authCode == "" {
		// wait for auth code
		time.Sleep(1 * time.Second)
	}

	fmt.Println("auth code received")
	var err error
	irc.tok, err = conf.Exchange(ctx, irc.authCode)
	if err != nil {
		return fmt.Errorf("failed to get token with auth code: %w", err)
	}
	fmt.Printf("Token received: %s\n", irc.tok.AccessToken)
	fmt.Println("IMPORTANT: Save this token to 1Password as TWITCH_TOKEN to avoid OAuth flow on restart")
	return nil
}
