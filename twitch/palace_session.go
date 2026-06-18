package twitchirc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/Soypete/twitch-llm-bot/logging"
)

type PalaceSession struct {
	StreamID    string
	PalacePath  string
	TempDir     string
	mu          sync.Mutex
	logger      *logging.Logger
	messageBuf  []string
	flushTicker *time.Ticker
	stopCh      chan struct{}
	httpClient  *http.Client
	wrapperURL  string
}

type SearchResult struct {
	Text       string  `json:"text"`
	Wing       string  `json:"wing"`
	Room       string  `json:"room"`
	Similarity float64 `json:"similarity"`
}

type SearchResponse struct {
	Query   string         `json:"query"`
	Results []SearchResult `json:"results"`
}

type mineRequest struct {
	Messages []string `json:"messages"`
	Palace   string   `json:"palace"`
	Wing     string   `json:"wing"`
	Room     string   `json:"room"`
}

type searchRequest struct {
	Query   string `json:"query"`
	Palace  string `json:"palace"`
	Wing    string `json:"wing"`
	Room    string `json:"room"`
	Results int    `json:"results"`
}

type httpResponse struct {
	Success bool   `json:"success"`
	Output  string `json:"output"`
	Result  string `json:"result"`
	Error   string `json:"error"`
}

func NewPalaceSession(streamID string, dataDir string, logger *logging.Logger) (*PalaceSession, error) {
	wrapperURL := os.Getenv("MEMPALACE_WRAPPER_URL")
	if wrapperURL == "" {
		wrapperURL = "http://localhost:8082"
	}

	session := &PalaceSession{
		StreamID:   streamID,
		PalacePath: filepath.Join(dataDir, "palaces", streamID),
		logger:     logger,
		messageBuf: make([]string, 0),
		stopCh:     make(chan struct{}),
		httpClient: &http.Client{Timeout: 2 * time.Minute},
		wrapperURL: wrapperURL,
	}

	if err := session.init(); err != nil {
		return nil, err
	}

	return session, nil
}

func (s *PalaceSession) init() error {
	s.TempDir = filepath.Join(s.PalacePath, "temp")

	if err := os.MkdirAll(s.TempDir, 0755); err != nil {
		return fmt.Errorf("failed to create palace directory: %w", err)
	}

	if err := s.writeConfig(); err != nil {
		return fmt.Errorf("failed to write mempalace config: %w", err)
	}

	if err := s.registerDefaultRoutes(); err != nil {
		s.logger.Warn("failed to register default routes", "error", err.Error())
	}

	s.flushTicker = time.NewTicker(30 * time.Second)
	go s.flushLoop()

	return nil
}

func (s *PalaceSession) writeConfig() error {
	config := `wing: stream
rooms:
  - general
  - questions
  - answers
  - off_topic
`
	configPath := filepath.Join(s.PalacePath, "mempalace.yaml")
	return os.WriteFile(configPath, []byte(config), 0644)
}

func (s *PalaceSession) registerDefaultRoutes() error {
	routes := []struct {
		entity   string
		location string
	}{
		{"stream|question", "stream/questions"},
		{"stream|ask", "stream/questions"},
		{"stream|help", "stream/questions"},
		{"stream|thanks", "stream/answers"},
		{"stream|thanks_pedro", "stream/answers"},
		{"stream|off_topic", "stream/off_topic"},
	}

	for _, route := range routes {
		reqBody, _ := json.Marshal(map[string]string{
			"entity_path": route.entity,
			"location":    route.location,
			"palace":      s.PalacePath,
		})

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		httpReq, err := http.NewRequestWithContext(ctx, "POST", s.wrapperURL+"/route/register", bytes.NewBuffer(reqBody))
		if err != nil {
			s.logger.Debug("failed to create route request", "entity", route.entity, "error", err.Error())
			continue
		}
		httpReq.Header.Set("Content-Type", "application/json")

		resp, err := s.httpClient.Do(httpReq)
		if err != nil {
			s.logger.Debug("failed to register route", "entity", route.entity, "error", err.Error())
			continue
		}
		_ = resp.Body.Close()
	}

	return nil
}

func (s *PalaceSession) determineRoom(username, message string) string {
	entityPath := fmt.Sprintf("stream|%s", sanitizeForEntity(message))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(ctx, "GET", s.wrapperURL+"/route/resolve?entity_path="+entityPath+"&palace="+s.PalacePath, nil)
	if err != nil {
		s.logger.Debug("failed to create resolve request", "error", err.Error())
		return s.keywordRoom(message)
	}

	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		s.logger.Debug("route resolve failed, using keyword detection", "error", err.Error())
		return s.keywordRoom(message)
	}
	defer resp.Body.Close()

	var result httpResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return s.keywordRoom(message)
	}

	if !result.Success || result.Result == "" {
		return s.keywordRoom(message)
	}

	resultStr := strings.TrimSpace(result.Result)
	if resultStr == "" || strings.Contains(resultStr, "No route found") {
		return s.keywordRoom(message)
	}

	parts := strings.Split(resultStr, "/")
	if len(parts) >= 2 {
		return parts[1]
	}

	return s.keywordRoom(message)
}

func (s *PalaceSession) keywordRoom(message string) string {
	lowerMsg := strings.ToLower(message)

	questionWords := []string{"?", "what", "how", "why", "when", "where", "who", "which", "can you", "could you", "would you", "is it possible", "do you know"}
	for _, q := range questionWords {
		if strings.Contains(lowerMsg, q) {
			return "questions"
		}
	}

	thanksWords := []string{"thanks", "thank you", "appreciate", "thx"}
	for _, t := range thanksWords {
		if strings.Contains(lowerMsg, t) {
			return "answers"
		}
	}

	offtopicWords := []string{"weather", "cats", "dogs", "food", "movie", "game", " weekend", "holiday"}
	for _, o := range offtopicWords {
		if strings.Contains(lowerMsg, o) {
			return "off_topic"
		}
	}

	return "general"
}

func sanitizeForEntity(message string) string {
	sanitized := strings.ToLower(message)
	sanitized = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == ' ' {
			return r
		}
		return -1
	}, sanitized)

	words := strings.Fields(sanitized)
	if len(words) > 6 {
		words = words[:6]
	}
	return strings.Join(words, " ")
}

func (s *PalaceSession) flushLoop() {
	for {
		select {
		case <-s.flushTicker.C:
			s.flush()
		case <-s.stopCh:
			s.flush()
			return
		}
	}
}

func (s *PalaceSession) flush() {
	s.mu.Lock()
	if len(s.messageBuf) == 0 {
		s.mu.Unlock()
		return
	}
	buf := s.messageBuf
	s.messageBuf = make([]string, 0)
	s.mu.Unlock()

	_ = s.writeAndMine(buf)
}

func (s *PalaceSession) writeAndMine(messages []string) error {
	roomMsgs := make(map[string][]string)

	for _, msg := range messages {
		var username, text string
		if strings.HasPrefix(msg, "[") {
			parts := strings.SplitN(msg, "]: ", 2)
			if len(parts) == 2 {
				username = strings.TrimPrefix(parts[0], "[")
				text = parts[1]
			}
		} else {
			text = msg
		}

		room := s.determineRoom(username, text)
		roomMsgs[room] = append(roomMsgs[room], msg)
	}

	for room, msgs := range roomMsgs {
		reqBody, _ := json.Marshal(mineRequest{
			Messages: msgs,
			Palace:   s.PalacePath,
			Wing:     "stream",
			Room:     room,
		})

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		httpReq, err := http.NewRequestWithContext(ctx, "POST", s.wrapperURL+"/mine", bytes.NewBuffer(reqBody))
		if err != nil {
			s.logger.Error("failed to create mine request", "room", room, "error", err.Error())
			continue
		}
		httpReq.Header.Set("Content-Type", "application/json")

		resp, err := s.httpClient.Do(httpReq)
		if err != nil {
			s.logger.Error("failed to mine messages", "room", room, "error", err.Error())
			continue
		}
		defer func() { _ = resp.Body.Close() }()

		var result httpResponse
		_ = json.NewDecoder(resp.Body).Decode(&result)

		s.logger.Debug("indexed messages to palace", "room", room, "count", len(msgs))
	}

	return nil
}

func (s *PalaceSession) IndexMessage(username, message string) error {
	entry := fmt.Sprintf("[%s]: %s", username, message)

	s.mu.Lock()
	s.messageBuf = append(s.messageBuf, entry)
	count := len(s.messageBuf)
	s.mu.Unlock()

	if count >= 10 {
		s.flush()
	}

	return nil
}

func (s *PalaceSession) GetContext(query string) (string, error) {
	rooms := []string{"general", "questions", "answers", "off_topic"}
	var allResults string

	for _, room := range rooms {
		reqBody, _ := json.Marshal(searchRequest{
			Query:   query,
			Palace:  s.PalacePath,
			Wing:    "stream",
			Room:    room,
			Results: 3,
		})

		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		httpReq, err := http.NewRequestWithContext(ctx, "POST", s.wrapperURL+"/search", bytes.NewBuffer(reqBody))
		if err != nil {
			s.logger.Debug("failed to create search request", "room", room, "error", err.Error())
			continue
		}
		httpReq.Header.Set("Content-Type", "application/json")

		resp, err := s.httpClient.Do(httpReq)
		if err != nil {
			s.logger.Debug("search room failed", "room", room, "error", err.Error())
			continue
		}
		defer resp.Body.Close()

		var result httpResponse
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			continue
		}

		if result.Success && result.Output != "" {
			results := s.parseSearchOutput(result.Output)
			if results != "" {
				allResults += fmt.Sprintf("[%s]\n%s\n---\n", room, results)
			}
		}
	}

	if allResults == "" {
		return "", nil
	}

	return allResults, nil
}

func (s *PalaceSession) parseSearchOutput(output string) string {
	var results SearchResponse
	if err := json.Unmarshal([]byte(output), &results); err == nil {
		var context string
		for _, r := range results.Results {
			context += r.Text + "\n---\n"
		}
		return context
	}

	lines := make([]string, 0)
	for _, line := range outputLines(output) {
		if line == "Results for:" || line == "" {
			continue
		}
		if len(line) > 2 && line[:2] == "  " && !contains(line, []string{"Wing:", "Room:", "Source:", "Match:", "─", "="}) {
			lines = append(lines, line)
		}
	}

	return joinLines(lines)
}

func (s *PalaceSession) End() error {
	close(s.stopCh)
	s.flushTicker.Stop()

	if err := os.RemoveAll(s.PalacePath); err != nil {
		s.logger.Error("failed to cleanup palace directory", "error", err.Error())
		return err
	}

	return nil
}

func outputLines(s string) []string {
	return splitLines(s)
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

func joinLines(lines []string) string {
	result := ""
	for _, line := range lines {
		trimmed := trimPrefix(line, "  ")
		if trimmed != "" {
			result += trimmed + "\n"
		}
	}
	return result
}

func trimPrefix(s, prefix string) string {
	if len(s) >= len(prefix) && s[:len(prefix)] == prefix {
		return s[len(prefix):]
	}
	return s
}

func contains(s string, substrs []string) bool {
	for _, sub := range substrs {
		if len(s) >= len(sub) && containsSubstring(s, sub) {
			return true
		}
	}
	return false
}

func containsSubstring(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
