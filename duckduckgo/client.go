package duckduckgo

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

type Client struct {
	BaseURL    string
	HTTPClient *http.Client
}

type Response struct {
	Abstract        string          `json:"Abstract"`
	AbstractSource  string          `json:"AbstractSource"`
	AbstractURL     string          `json:"AbstractURL"`
	Entity          string          `json:"Entity"`
	Heading         string          `json:"Heading"`
	Image           string          `json:"Image"`
	ImageHeight     int             `json:"ImageHeight"`
	ImageIsLogo     int             `json:"ImageIsLogo"`
	ImageWidth      int             `json:"ImageWidth"`
	Infobox         Infobox         `json:"Infobox"`
	RelatedTopics   []RelatedTopic  `json:"RelatedTopics"`
	Results         []Result        `json:"Results"`
	Type            string          `json:"Type"`
	Meta            json.RawMessage `json:"meta"`
}

type Infobox struct {
	Content []InfoboxContent `json:"content"`
	Meta    []InfoboxMeta    `json:"meta"`
}

type InfoboxContent struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

type InfoboxMeta struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

type RelatedTopic struct {
	FirstURL string `json:"FirstURL"`
	Icon     Icon   `json:"Icon"`
	Result   string `json:"Result"`
	Text     string `json:"Text"`
}

type Icon struct {
	Height string `json:"Height"`
	URL    string `json:"URL"`
	Width  string `json:"Width"`
}

type Result struct {
	FirstURL string `json:"FirstURL"`
	Icon     Icon   `json:"Icon"`
	Result   string `json:"Result"`
	Text     string `json:"Text"`
}

func NewClient() *Client {
	return &Client{
		BaseURL: "https://api.duckduckgo.com",
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *Client) Search(query string) (*Response, error) {
	if query == "" {
		return nil, fmt.Errorf("query cannot be empty")
	}

	u, err := url.Parse(c.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}

	params := url.Values{}
	params.Set("q", query)
	params.Set("format", "json")
	params.Set("no_html", "1")
	params.Set("skip_disambig", "1")

	u.RawQuery = params.Encode()

	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "twitch-llm-bot/1.0")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var ddgResponse Response
	if err := json.NewDecoder(resp.Body).Decode(&ddgResponse); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &ddgResponse, nil
}

func (c *Client) SearchWithOptions(query string, options SearchOptions) (*Response, error) {
	if query == "" {
		return nil, fmt.Errorf("query cannot be empty")
	}

	u, err := url.Parse(c.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}

	params := url.Values{}
	params.Set("q", query)
	params.Set("format", "json")

	if options.NoHTML {
		params.Set("no_html", "1")
	}
	if options.SkipDisambig {
		params.Set("skip_disambig", "1")
	}
	if options.NoRedirect {
		params.Set("no_redirect", "1")
	}
	if options.SafeSearch != "" {
		params.Set("safe_search", options.SafeSearch)
	}

	u.RawQuery = params.Encode()

	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "twitch-llm-bot/1.0")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var ddgResponse Response
	if err := json.NewDecoder(resp.Body).Decode(&ddgResponse); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &ddgResponse, nil
}

type SearchOptions struct {
	NoHTML        bool
	SkipDisambig  bool
	NoRedirect    bool
	SafeSearch    string // "strict", "moderate", "off"
}