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

func (irc *IRC) parseAuthCode(w http.ResponseWriter, req *http.Request) {
	err := req.ParseForm()
	if err != nil {
		fmt.Printf("could not parse query: %v", err)
		http.Error(w, "could not parse query", http.StatusBadRequest)
	}
	irc.authCode = req.FormValue("code")
}

// AuthTwitch use oauth2 protocol to retrieve oauth2 token for twitch IRC.
// _NOTE_: this has not been tested on long standing projects.
func (irc *IRC) AuthTwitch(ctx context.Context) error {
	http.HandleFunc("/oauth/redirect", irc.parseAuthCode)
	go http.ListenAndServe("localhost:3000", nil)

	conf := &oauth2.Config{
		// TODO: use const for the following.
		ClientID:     os.Getenv("TWITCH_ID"),
		ClientSecret: os.Getenv("TWITCH_SECRET"),
		Scopes:       []string{"chat:read", "chat:edit", "channel:moderate"},
		RedirectURL:  "http://localhost:3000/oauth/redirect",
		Endpoint:     twitch.Endpoint,
	}
	irc.wg.Add(1)

	// Redirect user to consent page to ask for permission
	// for the scopes specified above.
	go func() {
		defer irc.wg.Done()
		url := conf.AuthCodeURL("state", oauth2.AccessTypeOffline)
		fmt.Printf("Visit the URL for the auth dialog: %v\n", url)
		// wait for auth code
		for irc.authCode == "" {
			time.Sleep(1 * time.Second)
		}
		var err error
		irc.tok, err = conf.Exchange(ctx, irc.authCode)
		if err != nil {
			// print until we have ctx.done
			fmt.Println(fmt.Errorf("failed to get token with auth code: %w", err))
		}
	}()
	irc.wg.Wait()
	return nil
}
