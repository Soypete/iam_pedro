package twitchirc

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/bwmarrin/discordgo"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/twitch"
)

const discordAuthChannel = "pedrogpt"

// sendDiscordAuthURL sends the Twitch OAuth URL to the pedrogpt Discord channel.
// Falls back to logging if Discord is unavailable.
func (irc *IRC) sendDiscordAuthURL(authURL string) {
	token := os.Getenv("DISCORD_SECRET")
	if token == "" {
		irc.logger.Warn("DISCORD_SECRET not set, OAuth URL only in logs", "url", authURL)
		return
	}

	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		irc.logger.Error("failed to create discord session for auth notification", "error", err.Error())
		return
	}
	if err := dg.Open(); err != nil {
		irc.logger.Error("failed to open discord session for auth notification", "error", err.Error())
		return
	}
	defer func() { _ = dg.Close() }()

	var channelID string
	for _, guild := range dg.State.Guilds {
		channels, err := dg.GuildChannels(guild.ID)
		if err != nil {
			continue
		}
		for _, ch := range channels {
			if ch.Name == discordAuthChannel {
				channelID = ch.ID
				break
			}
		}
		if channelID != "" {
			break
		}
	}

	if channelID == "" {
		irc.logger.Error("could not find discord channel for auth notification", "channel", discordAuthChannel)
		return
	}

	msg := fmt.Sprintf("🔒 **Pedro needs Twitch auth!** Click to authorize (log in as the BOT account):\n%s", authURL)
	if _, err := dg.ChannelMessageSend(channelID, msg); err != nil {
		irc.logger.Error("failed to send auth URL to discord", "error", err.Error())
	}
}

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
		irc.logger.Error("could not parse oauth query", "error", err.Error())
		http.Error(w, "could not parse query", http.StatusBadRequest)
		return
	}
	receivedState := req.FormValue("state")
	if receivedState != irc.oauthState {
		irc.logger.Error("invalid oauth state", "expected", irc.oauthState, "got", receivedState)
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
	irc.logger.Info("token refreshed silently")
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

	// If we already have a token, this is a re-auth call (e.g. after "login authentication failed").
	// Skip env var paths — they'd just load the same expired token again.
	// Try silent refresh first; if that fails, fall through to interactive OAuth.
	if irc.tok != nil {
		irc.logger.Info("re-auth: existing token invalid, attempting silent refresh")
		if err := irc.tryRefreshToken(ctx); err == nil {
			return nil
		}
		irc.logger.Warn("silent refresh failed, falling through to interactive OAuth")
	} else {
		// First-time auth: try env vars.
		// If a refresh token is available, try silent refresh first.
		if refreshStr != "" {
			irc.logger.Info("attempting silent token refresh using TWITCH_REFRESH_TOKEN")
			irc.tok = &oauth2.Token{
				AccessToken:  tokenStr,
				RefreshToken: refreshStr,
			}
			if err := irc.tryRefreshToken(ctx); err == nil {
				return nil
			}
			irc.logger.Warn("silent refresh failed, falling back to other auth methods")
		}

		// Fall back to direct access token from environment.
		if tokenStr != "" {
			irc.logger.Info("using TWITCH_TOKEN from environment")
			irc.tok = &oauth2.Token{
				AccessToken: tokenStr,
			}
			irc.tokenRefreshTime = time.Now()
			return nil
		}
	}

	// No env-based token available — run interactive OAuth flow.
	irc.logger.Info("initiating interactive OAuth flow", "redirect_host", redirectHost)

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

	authURL := irc.oauthConf.AuthCodeURL(state, oauth2.AccessTypeOffline)
	irc.sendDiscordAuthURL(authURL)

	for irc.authCode == "" {
		time.Sleep(1 * time.Second)
	}

	irc.logger.Info("auth code received, exchanging for token")
	irc.tok, err = irc.oauthConf.Exchange(ctx, irc.authCode)
	if err != nil {
		return fmt.Errorf("failed to get token with auth code: %w", err)
	}
	irc.tokenRefreshTime = time.Now()
	irc.logger.Info("token received successfully")
	return nil
}
