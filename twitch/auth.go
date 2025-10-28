package twitchirc

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/twitch"
)

// TODO: Future enhancement - Add reauth endpoint for automated token refresh
// This will allow the keepalive service to trigger token refresh when expiry is detected.
// Implementation should be done once k8s deployment is setup to handle secure token management.
// Proposed endpoint: POST /auth/refresh that triggers the OAuth flow and updates the token.

func (irc *IRC) parseAuthCode(w http.ResponseWriter, req *http.Request) {
	err := req.ParseForm()
	if err != nil {
		fmt.Printf("could not parse query: %v", err)
		http.Error(w, "could not parse query", http.StatusBadRequest)
	}
	irc.authCode = req.FormValue("code")
}

// AuthTwitch use oauth2 protocol to retrieve oauth2 token for twitch IRC.
// First checks for TWITCH_TOKEN env var. If not found, runs OAuth flow.
func (irc *IRC) AuthTwitch(ctx context.Context) error {
	// Check if token is already available via environment variable
	if tokenStr := os.Getenv("TWITCH_TOKEN"); tokenStr != "" {
		fmt.Println("Using TWITCH_TOKEN from environment")
		irc.tok = &oauth2.Token{
			AccessToken: tokenStr,
		}
		irc.tokenRefreshTime = time.Now()
		return nil
	}

	// Fall back to OAuth flow
	fmt.Println("TWITCH_TOKEN not found, initiating OAuth flow...")

	// Determine redirect host (defaults to localhost for local dev)
	redirectHost := os.Getenv("OAUTH_REDIRECT_HOST")
	if redirectHost == "" {
		redirectHost = "localhost:3000"
	}

	// Always use HTTPS for OAuth redirect
	protocol := "https"

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
	url := conf.AuthCodeURL("state", oauth2.AccessTypeOffline)
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
	irc.tokenRefreshTime = time.Now()
	fmt.Printf("Token received: %s\n", irc.tok.AccessToken)
	fmt.Println("IMPORTANT: Save this token to 1Password as TWITCH_TOKEN to avoid OAuth flow on restart")
	return nil
}
