// Package moderation provides a Twitch chat moderation system using LLM-based decision making
package moderation

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/Soypete/twitch-llm-bot/ai"
	"github.com/Soypete/twitch-llm-bot/ai/twitchchat/agent"
	"github.com/Soypete/twitch-llm-bot/database"
	"github.com/Soypete/twitch-llm-bot/logging"
	"github.com/Soypete/twitch-llm-bot/metrics"
	"github.com/Soypete/twitch-llm-bot/twitch/helix"
	"github.com/Soypete/twitch-llm-bot/types"
	"github.com/google/uuid"
	v2 "github.com/gempir/go-twitch-irc/v2"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
)

// Monitor handles chat moderation in parallel to the main chat handler
type Monitor struct {
	config        *ai.ModerationConfig
	llm           llms.Model
	modelName     string
	helixClient   *helix.Client
	db            database.ModActionWriter
	logger        *logging.Logger
	messageCh     chan v2.PrivateMessage
	channelID     string
	channelName   string
	recentMsgs    []types.TwitchMessage
	recentMsgsMu  sync.RWMutex
	maxRecentMsgs int
	ircClient     *v2.Client

	// Rate limiting
	actionCount    int
	actionCountMu  sync.Mutex
	lastResetTime  time.Time
}

// NewMonitor creates a new moderation monitor
func NewMonitor(
	config *ai.ModerationConfig,
	llmPath string,
	modelName string,
	helixClient *helix.Client,
	db database.ModActionWriter,
	channelID string,
	channelName string,
	logger *logging.Logger,
) (*Monitor, error) {
	if logger == nil {
		logger = logging.Default()
	}

	// Set up LLM client
	if llmPath != "" && !strings.HasSuffix(llmPath, "/v1") {
		llmPath = llmPath + "/v1"
	}

	opts := []openai.Option{
		openai.WithBaseURL(llmPath),
		openai.WithModel(modelName),
	}
	llm, err := openai.New(opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create moderation LLM: %w", err)
	}

	return &Monitor{
		config:        config,
		llm:           llm,
		modelName:     modelName,
		helixClient:   helixClient,
		db:            db,
		logger:        logger,
		messageCh:     make(chan v2.PrivateMessage, 100),
		channelID:     channelID,
		channelName:   channelName,
		recentMsgs:    make([]types.TwitchMessage, 0, 20),
		maxRecentMsgs: 20,
		lastResetTime: time.Now(),
	}, nil
}

// SetIRCClient sets the IRC client for sending warning messages
func (m *Monitor) SetIRCClient(client *v2.Client) {
	m.ircClient = client
}

// MessageChannel returns the channel for sending messages to be moderated
func (m *Monitor) MessageChannel() chan<- v2.PrivateMessage {
	return m.messageCh
}

// Start begins the moderation monitoring loop
func (m *Monitor) Start(ctx context.Context, wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		m.logger.Info("moderation monitor started", "channel", m.channelName, "dryRun", m.config.DryRun)

		for {
			select {
			case <-ctx.Done():
				m.logger.Info("moderation monitor shutting down")
				return
			case msg := <-m.messageCh:
				m.processMessage(ctx, msg)
			}
		}
	}()
}

// processMessage evaluates a message for moderation
func (m *Monitor) processMessage(ctx context.Context, msg v2.PrivateMessage) {
	// Skip if moderation is disabled
	if !m.config.Enabled {
		return
	}

	// Skip if channel is not in the moderated list
	if !m.config.IsChannelModerated(m.channelName) {
		return
	}

	// Skip messages from known bots
	if m.shouldSkipUser(msg.User.DisplayName) {
		return
	}

	// Convert to TwitchMessage and add to recent messages
	twitchMsg := types.TwitchMessage{
		Username: msg.User.DisplayName,
		Text:     msg.Message,
		Time:     time.Now(),
	}

	m.addRecentMessage(twitchMsg)

	// Quick filter - skip obviously safe messages
	if !m.needsEvaluation(msg.Message) {
		m.logger.Debug("message skipped by quick filter", "user", msg.User.DisplayName)
		return
	}

	m.logger.Debug("evaluating message for moderation", "user", msg.User.DisplayName, "messageID", msg.ID)

	// Build context for LLM
	modContext := types.ModerationContext{
		Message:        twitchMsg,
		MessageID:      msg.ID,
		RecentMessages: m.getRecentMessages(),
		ChannelRules:   m.config.ChannelRules,
		ChannelID:      m.channelID,
		ChannelName:    m.channelName,
	}

	// Get LLM decision
	decision, err := m.evaluateWithLLM(ctx, modContext)
	if err != nil {
		m.logger.Error("failed to evaluate message with LLM", "error", err.Error(), "user", msg.User.DisplayName)
		metrics.FailedLLMGenCount.Add(1)
		return
	}

	metrics.SuccessfulLLMGenCount.Add(1)

	// Log the decision
	m.logger.Debug("moderation decision",
		"user", msg.User.DisplayName,
		"tool", decision.ToolCall,
		"shouldAct", decision.ShouldAct,
		"reasoning", decision.Reasoning,
	)

	// Execute the action if needed
	if decision.ShouldAct && decision.ToolCall != agent.ToolNoAction {
		m.executeAction(ctx, msg, decision)
	} else {
		// Log no_action decisions for audit trail
		m.logModAction(ctx, msg, decision, nil, true, "")
	}
}

// shouldSkipUser returns true if the user should not be moderated
func (m *Monitor) shouldSkipUser(username string) bool {
	skipUsers := []string{
		"Nightbot",
		"StreamElements",
		"Streamlabs",
		"Moobot",
		"Pedro_el_asistente",
		"soypetetech", // Don't moderate the streamer
	}

	for _, skip := range skipUsers {
		if strings.EqualFold(username, skip) {
			return true
		}
	}
	return false
}

// needsEvaluation performs a quick filter to determine if a message needs LLM evaluation
func (m *Monitor) needsEvaluation(message string) bool {
	// Skip very short messages
	if len(message) < 5 {
		return false
	}

	// Skip messages that are just emotes
	if strings.HasPrefix(message, "soypet2") && !strings.Contains(message, " ") {
		return false
	}

	// Keywords that might indicate problematic content
	problematicPatterns := []string{
		"http://", "https://", // Links might be spam
		"@everyone", "@here",  // Mass mentions
	}

	lowerMsg := strings.ToLower(message)
	for _, pattern := range problematicPatterns {
		if strings.Contains(lowerMsg, pattern) {
			return true
		}
	}

	// Check for excessive caps (potential spam/aggression)
	upperCount := 0
	for _, c := range message {
		if c >= 'A' && c <= 'Z' {
			upperCount++
		}
	}
	if len(message) > 10 && float64(upperCount)/float64(len(message)) > 0.7 {
		return true
	}

	// Check for repeated characters (spam indicator)
	if hasRepeatedChars(message, 5) {
		return true
	}

	// Most messages don't need evaluation
	return false
}

// hasRepeatedChars checks if a string has the same character repeated n times
func hasRepeatedChars(s string, n int) bool {
	if len(s) < n {
		return false
	}
	count := 1
	for i := 1; i < len(s); i++ {
		if s[i] == s[i-1] {
			count++
			if count >= n {
				return true
			}
		} else {
			count = 1
		}
	}
	return false
}

// addRecentMessage adds a message to the recent messages buffer
func (m *Monitor) addRecentMessage(msg types.TwitchMessage) {
	m.recentMsgsMu.Lock()
	defer m.recentMsgsMu.Unlock()

	if len(m.recentMsgs) >= m.maxRecentMsgs {
		m.recentMsgs = m.recentMsgs[1:]
	}
	m.recentMsgs = append(m.recentMsgs, msg)
}

// getRecentMessages returns a copy of recent messages
func (m *Monitor) getRecentMessages() []types.TwitchMessage {
	m.recentMsgsMu.RLock()
	defer m.recentMsgsMu.RUnlock()

	result := make([]types.TwitchMessage, len(m.recentMsgs))
	copy(result, m.recentMsgs)
	return result
}

// evaluateWithLLM uses the LLM to decide on moderation action
func (m *Monitor) evaluateWithLLM(ctx context.Context, modContext types.ModerationContext) (*types.ModerationDecision, error) {
	// Build the system prompt
	rulesStr := strings.Join(modContext.ChannelRules, "\n- ")
	if rulesStr != "" {
		rulesStr = "- " + rulesStr
	}
	systemPrompt := fmt.Sprintf(ai.ModerationPrompt, rulesStr, m.config.SensitivityLevel)

	// Build the user message with context
	var recentContext strings.Builder
	recentContext.WriteString("Recent chat messages:\n")
	for _, msg := range modContext.RecentMessages {
		recentContext.WriteString(fmt.Sprintf("[%s]: %s\n", msg.Username, msg.Text))
	}

	userMessage := fmt.Sprintf(`%s

Message to evaluate:
User: %s
Message ID: %s
Content: %s

Analyze this message and decide if moderation action is needed. Call exactly one tool with your decision.`,
		recentContext.String(),
		modContext.Message.Username,
		modContext.MessageID,
		modContext.Message.Text,
	)

	// Build message history
	messageHistory := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, systemPrompt),
		llms.TextParts(llms.ChatMessageTypeHuman, userMessage),
	}

	// Get available tools based on configuration
	tools := m.getAvailableTools()

	// Call the LLM
	resp, err := m.llm.GenerateContent(ctx, messageHistory,
		llms.WithModel(m.modelName),
		llms.WithCandidateCount(1),
		llms.WithMaxLength(500),
		llms.WithTemperature(0.3), // Lower temperature for more consistent moderation
		llms.WithTools(tools),
	)
	if err != nil {
		return nil, fmt.Errorf("LLM generation failed: %w", err)
	}

	// Parse the response
	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in LLM response")
	}

	choice := resp.Choices[0]

	// Check for tool calls
	if len(choice.ToolCalls) > 0 {
		toolCall := choice.ToolCalls[0]
		parsed, err := agent.ParseModerationToolCall(toolCall)
		if err != nil {
			return nil, fmt.Errorf("failed to parse tool call: %w", err)
		}

		decision := &types.ModerationDecision{
			ShouldAct:  parsed.ToolName != agent.ToolNoAction,
			ToolCall:   parsed.ToolName,
			ToolParams: parsed.Args,
		}

		// Extract reasoning if provided
		if reason, ok := parsed.Args["reason"].(string); ok {
			decision.Reasoning = reason
		}

		return decision, nil
	}

	// If no tool call, default to no action
	return &types.ModerationDecision{
		ShouldAct:  false,
		ToolCall:   agent.ToolNoAction,
		ToolParams: map[string]interface{}{"reason": "No tool call in response"},
		Reasoning:  "No tool call in LLM response",
	}, nil
}

// getAvailableTools returns the tools based on configuration
func (m *Monitor) getAvailableTools() []llms.Tool {
	allTools := agent.GetCoreModerationToolDefinitions()

	if len(m.config.AllowedTools) == 0 {
		return allTools
	}

	var filtered []llms.Tool
	for _, tool := range allTools {
		if m.config.IsToolAllowed(tool.Function.Name) {
			filtered = append(filtered, tool)
		}
	}

	// Always include no_action
	hasNoAction := false
	for _, tool := range filtered {
		if tool.Function.Name == agent.ToolNoAction {
			hasNoAction = true
			break
		}
	}
	if !hasNoAction {
		filtered = append(filtered, agent.GetCoreModerationToolDefinitions()[0]) // no_action is first
	}

	return filtered
}

// executeAction executes a moderation action
func (m *Monitor) executeAction(ctx context.Context, msg v2.PrivateMessage, decision *types.ModerationDecision) {
	// Check rate limits
	if !m.checkRateLimit() {
		m.logger.Warn("rate limit exceeded, skipping action",
			"tool", decision.ToolCall,
			"user", msg.User.DisplayName,
		)
		return
	}

	// Check if tool is allowed
	if !m.config.IsToolAllowed(decision.ToolCall) {
		m.logger.Warn("tool not in allowed list",
			"tool", decision.ToolCall,
			"user", msg.User.DisplayName,
		)
		return
	}

	var apiResponse []byte
	var err error
	var success bool
	var errorMsg string

	// In dry run mode, just log what would happen
	if m.config.DryRun {
		m.logger.Info("DRY RUN: would execute moderation action",
			"tool", decision.ToolCall,
			"params", decision.ToolParams,
			"user", msg.User.DisplayName,
		)
		m.logModAction(ctx, msg, decision, nil, true, "dry run - no action taken")
		return
	}

	// Execute the action based on tool type
	switch decision.ToolCall {
	case agent.ToolWarnUser:
		err = m.executeWarnUser(ctx, msg, decision)
	case agent.ToolTimeoutUser:
		apiResponse, err = m.executeTimeoutUser(ctx, msg, decision)
	case agent.ToolBanUser:
		apiResponse, err = m.executeBanUser(ctx, msg, decision)
	case agent.ToolUnbanUser:
		apiResponse, err = m.executeUnbanUser(ctx, msg, decision)
	case agent.ToolDeleteMessage:
		apiResponse, err = m.executeDeleteMessage(ctx, msg, decision)
	case agent.ToolClearChat:
		apiResponse, err = m.executeClearChat(ctx)
	case agent.ToolEmoteOnlyMode:
		apiResponse, err = m.executeEmoteOnlyMode(ctx, decision)
	case agent.ToolSubscriberOnlyMode:
		apiResponse, err = m.executeSubscriberOnlyMode(ctx, decision)
	case agent.ToolFollowerOnlyMode:
		apiResponse, err = m.executeFollowerOnlyMode(ctx, decision)
	case agent.ToolSlowMode:
		apiResponse, err = m.executeSlowMode(ctx, decision)
	default:
		m.logger.Warn("unknown moderation tool", "tool", decision.ToolCall)
		return
	}

	if err != nil {
		success = false
		errorMsg = err.Error()
		m.logger.Error("failed to execute moderation action",
			"error", err.Error(),
			"tool", decision.ToolCall,
			"user", msg.User.DisplayName,
		)
	} else {
		success = true
		m.logger.Info("moderation action executed",
			"tool", decision.ToolCall,
			"user", msg.User.DisplayName,
			"reasoning", decision.Reasoning,
		)
	}

	// Log to database
	m.logModAction(ctx, msg, decision, apiResponse, success, errorMsg)
}

// checkRateLimit checks and updates rate limiting
func (m *Monitor) checkRateLimit() bool {
	m.actionCountMu.Lock()
	defer m.actionCountMu.Unlock()

	// Reset counter every minute
	if time.Since(m.lastResetTime) > time.Minute {
		m.actionCount = 0
		m.lastResetTime = time.Now()
	}

	if m.actionCount >= m.config.RateLimits.ActionsPerMinute {
		return false
	}

	m.actionCount++
	return true
}

// logModAction logs a moderation action to the database
func (m *Monitor) logModAction(ctx context.Context, msg v2.PrivateMessage, decision *types.ModerationDecision, apiResponse []byte, success bool, errorMsg string) {
	paramsJSON, _ := json.Marshal(decision.ToolParams)

	targetUsername := ""
	if username, ok := decision.ToolParams["username"].(string); ok {
		targetUsername = username
	} else {
		targetUsername = msg.User.DisplayName
	}

	action := types.ModAction{
		ID:                    uuid.New(),
		CreatedAt:             time.Now(),
		TriggerMessageID:      msg.ID,
		TriggerUsername:       msg.User.DisplayName,
		TriggerMessageContent: msg.Message,
		LLMModel:              m.modelName,
		LLMReasoning:          decision.Reasoning,
		ToolCallName:          decision.ToolCall,
		ToolCallParams:        paramsJSON,
		TargetUsername:        targetUsername,
		TargetUserID:          decision.TargetUserID,
		TwitchAPIResponse:     apiResponse,
		Success:               success,
		ErrorMessage:          errorMsg,
		ChannelID:             m.channelID,
		ChannelName:           m.channelName,
	}

	if _, err := m.db.InsertModAction(ctx, action); err != nil {
		m.logger.Error("failed to log mod action to database", "error", err.Error())
	}
}

// Action execution methods

func (m *Monitor) executeWarnUser(ctx context.Context, msg v2.PrivateMessage, decision *types.ModerationDecision) error {
	if m.ircClient == nil {
		return fmt.Errorf("IRC client not set")
	}

	message := "Please follow the channel rules."
	if msg, ok := decision.ToolParams["message"].(string); ok && msg != "" {
		message = msg
	}

	warningMsg := fmt.Sprintf("@%s %s soypet2Peace", msg.User.DisplayName, message)
	m.ircClient.Say(m.channelName, warningMsg)
	return nil
}

func (m *Monitor) executeTimeoutUser(ctx context.Context, msg v2.PrivateMessage, decision *types.ModerationDecision) ([]byte, error) {
	duration := 60 // Default 60 seconds
	if d, ok := decision.ToolParams["duration_seconds"].(float64); ok {
		duration = int(d)
	}

	reason := "Violation of channel rules"
	if r, ok := decision.ToolParams["reason"].(string); ok && r != "" {
		reason = r
	}

	// Get user ID
	userID, err := m.helixClient.GetUserIDByLogin(ctx, msg.User.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to get user ID: %w", err)
	}

	decision.TargetUserID = userID
	return m.helixClient.BanUser(ctx, userID, duration, reason)
}

func (m *Monitor) executeBanUser(ctx context.Context, msg v2.PrivateMessage, decision *types.ModerationDecision) ([]byte, error) {
	reason := "Severe violation of channel rules"
	if r, ok := decision.ToolParams["reason"].(string); ok && r != "" {
		reason = r
	}

	// Get user ID
	userID, err := m.helixClient.GetUserIDByLogin(ctx, msg.User.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to get user ID: %w", err)
	}

	decision.TargetUserID = userID
	return m.helixClient.BanUser(ctx, userID, 0, reason) // 0 = permanent
}

func (m *Monitor) executeUnbanUser(ctx context.Context, msg v2.PrivateMessage, decision *types.ModerationDecision) ([]byte, error) {
	username := msg.User.Name
	if u, ok := decision.ToolParams["username"].(string); ok && u != "" {
		username = u
	}

	userID, err := m.helixClient.GetUserIDByLogin(ctx, username)
	if err != nil {
		return nil, fmt.Errorf("failed to get user ID: %w", err)
	}

	decision.TargetUserID = userID
	return m.helixClient.UnbanUser(ctx, userID)
}

func (m *Monitor) executeDeleteMessage(ctx context.Context, msg v2.PrivateMessage, decision *types.ModerationDecision) ([]byte, error) {
	messageID := msg.ID
	if id, ok := decision.ToolParams["message_id"].(string); ok && id != "" {
		messageID = id
	}

	return m.helixClient.DeleteMessage(ctx, messageID)
}

func (m *Monitor) executeClearChat(ctx context.Context) ([]byte, error) {
	return m.helixClient.ClearChat(ctx)
}

func (m *Monitor) executeEmoteOnlyMode(ctx context.Context, decision *types.ModerationDecision) ([]byte, error) {
	enabled := false
	if e, ok := decision.ToolParams["enabled"].(bool); ok {
		enabled = e
	}

	settings := helix.ChatSettings{
		EmoteMode: &enabled,
	}
	return m.helixClient.UpdateChatSettings(ctx, settings)
}

func (m *Monitor) executeSubscriberOnlyMode(ctx context.Context, decision *types.ModerationDecision) ([]byte, error) {
	enabled := false
	if e, ok := decision.ToolParams["enabled"].(bool); ok {
		enabled = e
	}

	settings := helix.ChatSettings{
		SubscriberMode: &enabled,
	}
	return m.helixClient.UpdateChatSettings(ctx, settings)
}

func (m *Monitor) executeFollowerOnlyMode(ctx context.Context, decision *types.ModerationDecision) ([]byte, error) {
	enabled := false
	if e, ok := decision.ToolParams["enabled"].(bool); ok {
		enabled = e
	}

	settings := helix.ChatSettings{
		FollowerMode: &enabled,
	}

	if enabled {
		duration := 0
		if d, ok := decision.ToolParams["duration_minutes"].(float64); ok {
			duration = int(d)
		}
		settings.FollowerModeDuration = &duration
	}

	return m.helixClient.UpdateChatSettings(ctx, settings)
}

func (m *Monitor) executeSlowMode(ctx context.Context, decision *types.ModerationDecision) ([]byte, error) {
	enabled := false
	if e, ok := decision.ToolParams["enabled"].(bool); ok {
		enabled = e
	}

	settings := helix.ChatSettings{
		SlowMode: &enabled,
	}

	if enabled {
		delay := 30 // Default 30 seconds
		if d, ok := decision.ToolParams["delay_seconds"].(float64); ok {
			delay = int(d)
		}
		settings.SlowModeWaitTime = &delay
	}

	return m.helixClient.UpdateChatSettings(ctx, settings)
}
