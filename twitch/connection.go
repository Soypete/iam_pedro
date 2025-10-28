package twitchirc

import (
	"context"
	"sync"
	"time"

	"github.com/Soypete/twitch-llm-bot/ai"
	"github.com/Soypete/twitch-llm-bot/database"
	"github.com/Soypete/twitch-llm-bot/logging"
	"github.com/Soypete/twitch-llm-bot/metrics"
	"github.com/Soypete/twitch-llm-bot/types"
	v2 "github.com/gempir/go-twitch-irc/v2"
	"github.com/pkg/errors"
	"golang.org/x/oauth2"
)

const peteTwitchChannel = "soypetetech"

// IRC Connection to the twitch IRC server.
type IRC struct {
	db               database.ChatResponseWriter
	modelName        string
	wg               *sync.WaitGroup
	Client           *v2.Client
	tok              *oauth2.Token
	tokenRefreshTime time.Time        // Time when the token was last refreshed
	llm              ai.Chatter
	authCode         string
	logger           *logging.Logger
	asyncResponseCh  chan types.TwitchMessage
}

// SetupTwitchIRC sets up the IRC, configures oauth, and inits connection functions.
func SetupTwitchIRC(wg *sync.WaitGroup, llm ai.Chatter, modelName string, db database.ChatResponseWriter, logger *logging.Logger) (*IRC, error) {
	if logger == nil {
		logger = logging.Default()
	}

	irc := &IRC{
		db:              db,
		wg:              wg,
		llm:             llm,
		modelName:       modelName,
		logger:          logger,
		asyncResponseCh: make(chan types.TwitchMessage, 10),
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

	// Start async response handler
	go irc.handleAsyncResponses(ctx)

	irc.Client = c
	return nil
}

// handleAsyncResponses listens for async responses (like web search results) and sends them to chat
func (irc *IRC) handleAsyncResponses(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			irc.logger.Info("shutting down async response handler")
			return
		case response := <-irc.asyncResponseCh:
			irc.logger.Debug("received async response", "messageID", response.UUID, "responseLength", len(response.Text))
			
			// Store the response in the database
			err := irc.db.InsertResponse(ctx, response, irc.modelName)
			if err != nil {
				irc.logger.Error("failed to insert async response into database", "error", err.Error(), "messageID", response.UUID)
			} else {
				irc.logger.Debug("async web search response stored in database", "messageID", response.UUID)
			}
			
			// Send the response to Twitch chat
			irc.Client.Say(peteTwitchChannel, response.Text)
			metrics.TwitchMessageSentCount.Add(1)
		}
	}
}
