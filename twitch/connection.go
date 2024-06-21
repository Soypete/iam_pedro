package twitchirc

import (
	"context"
	"log"
	"sync"

	"github.com/Soypete/twitch-llm-bot/database"
	"github.com/Soypete/twitch-llm-bot/langchain"
	v2 "github.com/gempir/go-twitch-irc/v2"
	"github.com/pkg/errors"
	"golang.org/x/oauth2"
)

const peteTwitchChannel = "soypetetech"

// IRC Connection to the twitch IRC server.
type IRC struct {
	db       database.MessageWriter
	wg       sync.WaitGroup
	Client   *v2.Client
	tok      *oauth2.Token
	llm      *langchain.Client
	authCode string
}

// SetupTwitchIRC sets up the IRC, configures oauth, and inits connection functions.
func SetupTwitchIRC(wg sync.WaitGroup, llm *langchain.Client, db database.Postgres) (*IRC, error) {
	irc := &IRC{
		db:  &db,
		wg:  wg,
		llm: llm,
	}
	// using a separate context here because it needs human interaction
	ctx := context.Background()
	err := irc.AuthTwitch(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to authenticate with twitch")
	}

	return irc, nil
}

// connectIRC gets the auth and connects to the twitch IRC server for channel.
func (irc *IRC) ConnectIRC(ctx context.Context) error {
	log.Println("Connecting to twitch IRC")
	c := v2.NewClient(peteTwitchChannel, "oauth:"+irc.tok.AccessToken)
	c.Join(peteTwitchChannel)
	// TODO: This is reconnecting repeatedly and causing a flood of messages to the channel
	c.OnConnect(func() {
		log.Println("connection to twitch IRC established")
	})
	c.OnPrivateMessage(func(msg v2.PrivateMessage) {
		irc.HandleChat(ctx, msg)
	})

	c.Say(peteTwitchChannel, "Hello, my name is Pedro_el_asistente I am here to help you.")
	irc.Client = c
	return nil
}
