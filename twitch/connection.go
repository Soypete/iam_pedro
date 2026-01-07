package twitchirc

import (
	"context"
	"os"
	"sync"
	"time"

	"github.com/Soypete/twitch-llm-bot/ai"
	"github.com/Soypete/twitch-llm-bot/database"
	"github.com/Soypete/twitch-llm-bot/logging"
	"github.com/Soypete/twitch-llm-bot/metrics"
	"github.com/Soypete/twitch-llm-bot/twitch/helix"
	"github.com/Soypete/twitch-llm-bot/twitch/moderation"
	"github.com/Soypete/twitch-llm-bot/types"
	v2 "github.com/gempir/go-twitch-irc/v2"
	"github.com/pkg/errors"
	"golang.org/x/oauth2"
)

const peteTwitchChannel = "soypetetech"

// IRC Connection to the twitch IRC server.
type IRC struct {
	db               database.ChatResponseWriter
	modDB            database.ModActionWriter
	modelName        string
	wg               *sync.WaitGroup
	Client           *v2.Client
	tok              *oauth2.Token
	tokenRefreshTime time.Time // Time when the token was last refreshed
	llm              ai.Chatter
	authCode         string
	logger           *logging.Logger
	asyncResponseCh  chan types.TwitchMessage

	// Moderation system
	modMonitor    *moderation.Monitor
	modConfig     *ai.ModerationConfig
	helixClient   *helix.Client
	broadcasterID string
	moderatorID   string
}

// SetupTwitchIRC sets up the IRC, configures oauth, and inits connection functions.
func SetupTwitchIRC(wg *sync.WaitGroup, llm ai.Chatter, modelName string, db database.ChatResponseWriter, logger *logging.Logger) (*IRC, error) {
	return SetupTwitchIRCWithModeration(wg, llm, modelName, db, nil, nil, logger)
}

// SetupTwitchIRCWithModeration sets up the IRC with optional moderation support
func SetupTwitchIRCWithModeration(wg *sync.WaitGroup, llm ai.Chatter, modelName string, db database.ChatResponseWriter, modDB database.ModActionWriter, modConfig *ai.ModerationConfig, logger *logging.Logger) (*IRC, error) {
	if logger == nil {
		logger = logging.Default()
	}

	irc := &IRC{
		db:              db,
		modDB:           modDB,
		wg:              wg,
		llm:             llm,
		modelName:       modelName,
		logger:          logger,
		asyncResponseCh: make(chan types.TwitchMessage, 10),
		modConfig:       modConfig,
	}

	// using a separate context here because it needs human interaction
	ctx := context.Background()
	err := irc.AuthTwitch(ctx)
	if err != nil {
		logger.Error("failed to authenticate with twitch", "error", err.Error())
		return nil, errors.Wrap(err, "failed to authenticate with twitch")
	}

	logger.Info("authenticating with twitch IRC")

	// Set up Helix client if moderation is enabled
	if modConfig != nil && modConfig.Enabled {
		if err := irc.setupModeration(ctx); err != nil {
			logger.Error("failed to setup moderation", "error", err.Error())
			// Continue without moderation - don't fail the whole bot
			irc.modConfig = nil
		}
	}

	return irc, nil
}

// setupModeration initializes the moderation system
func (irc *IRC) setupModeration(ctx context.Context) error {
	clientID := os.Getenv("TWITCH_ID")
	if clientID == "" {
		return errors.New("TWITCH_ID environment variable not set")
	}

	// Get broadcaster and moderator IDs
	irc.helixClient = helix.NewClient(clientID, irc.tok.AccessToken, "", "", irc.logger)

	// Get the bot's user ID (moderator ID)
	botUserID, err := irc.helixClient.GetUserIDByLogin(ctx, "pedro_el_asistente")
	if err != nil {
		// Try with soy_llm_bot as fallback
		botUserID, err = irc.helixClient.GetUserIDByLogin(ctx, "soy_llm_bot")
		if err != nil {
			return errors.Wrap(err, "failed to get bot user ID")
		}
	}
	irc.moderatorID = botUserID

	// Get broadcaster ID
	broadcasterID, err := irc.helixClient.GetUserIDByLogin(ctx, peteTwitchChannel)
	if err != nil {
		return errors.Wrap(err, "failed to get broadcaster ID")
	}
	irc.broadcasterID = broadcasterID

	// Update helix client with proper IDs
	irc.helixClient = helix.NewClient(clientID, irc.tok.AccessToken, broadcasterID, botUserID, irc.logger)

	irc.logger.Info("moderation system initialized",
		"broadcasterID", broadcasterID,
		"moderatorID", botUserID,
		"dryRun", irc.modConfig.DryRun,
	)

	return nil
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

	// Start moderation monitor if enabled
	if irc.modConfig != nil && irc.modConfig.Enabled && irc.helixClient != nil && irc.modDB != nil {
		llmPath := os.Getenv("LLAMA_CPP_PATH")
		monitor, err := moderation.NewMonitor(
			irc.modConfig,
			llmPath,
			irc.modelName,
			irc.helixClient,
			irc.modDB,
			irc.broadcasterID,
			peteTwitchChannel,
			irc.logger,
		)
		if err != nil {
			irc.logger.Error("failed to create moderation monitor", "error", err.Error())
		} else {
			irc.modMonitor = monitor
			monitor.SetIRCClient(c)
			monitor.Start(ctx, wg)
			irc.logger.Info("moderation monitor started")
		}
	}

	c.OnPrivateMessage(func(msg v2.PrivateMessage) {
		metrics.TwitchMessageRecievedCount.Add(1)
		irc.logger.Debug("received message", "user", msg.User.Name, "message", msg.Message)

		// Send to moderation monitor (non-blocking)
		if irc.modMonitor != nil {
			select {
			case irc.modMonitor.MessageChannel() <- msg:
			default:
				irc.logger.Debug("moderation channel full, skipping message")
			}
		}

		// Handle normal chat
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
