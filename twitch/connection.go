package twitchirc

import (
	"context"
	"sync"

	"github.com/Soypete/twitch-llm-bot/ai"
	"github.com/Soypete/twitch-llm-bot/database"
	"github.com/Soypete/twitch-llm-bot/logging"
	"github.com/Soypete/twitch-llm-bot/metrics"
	v2 "github.com/gempir/go-twitch-irc/v2"
	"github.com/pkg/errors"
	"golang.org/x/oauth2"
)

const peteTwitchChannel = "soypetetech"

// IRC Connection to the twitch IRC server.
type IRC struct {
	db       database.MessageWriter
	wg       *sync.WaitGroup
	Client   *v2.Client
	tok      *oauth2.Token
	llm      ai.Chatter
	authCode string
	logger   *logging.Logger
}

// SetupTwitchIRC sets up the IRC, configures oauth, and inits connection functions.
func SetupTwitchIRC(wg *sync.WaitGroup, llm ai.Chatter, db database.MessageWriter, logger *logging.Logger) (*IRC, error) {
	if logger == nil {
		logger = logging.Default()
	}
	
	irc := &IRC{
		db:     db,
		wg:     wg,
		llm:    llm,
		logger: logger,
	}
	
	// using a separate context here because it needs human interaction
	ctx := context.Background()
	err := irc.AuthTwitch(ctx)
	if err != nil {
		logger.Error("failed to authenticate with twitch", "error", err.Error())
		return nil, errors.Wrap(err, "failed to authenticate with twitch")
	}

	logger.Info("authenticating with twitch IRC")

	return irc, nil
}

// connectIRC gets the auth and connects to the twitch IRC server for channel.
func (irc *IRC) ConnectIRC(ctx context.Context, wg *sync.WaitGroup) error {
	irc.logger.Info("connecting to twitch IRC")
	c := v2.NewClient(peteTwitchChannel, "oauth:"+irc.tok.AccessToken)
	c.Join(peteTwitchChannel)
	c.OnConnect(func() {
		metrics.TwitchConnectionCount.Add(1)
		irc.logger.Info("connection to twitch IRC established")
	})
	c.OnPrivateMessage(func(msg v2.PrivateMessage) {
		metrics.TwitchMessageRecievedCount.Add(1)
		irc.logger.Debug("received message", "user", msg.User.Name, "message", msg.Message)
		irc.HandleChat(ctx, msg)
	})

	c.Say(peteTwitchChannel, "Hello, my name is Pedro_el_asistente I am here to help you.")

	irc.Client = c
	return nil
}
