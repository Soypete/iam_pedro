package duckduckgo

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

type Client struct {
	BaseURL    string
	HTTPClient *http.Client
}

type Response struct {
	Abstract       string          `json:"Abstract,omitempty"`
	AbstractSource string          `json:"AbstractSource,omitempty"`
	AbstractURL    string          `json:"AbstractURL,omitempty"`
	Entity         string          `json:"Entity,omitempty"`
	Heading        string          `json:"Heading,omitempty"`
	Image          string          `json:"Image,omitempty"`
	ImageHeight    any             `json:"ImageHeight,omitempty"` // types seem to vary
	ImageIsLogo    any             `json:"ImageIsLogo,omitempty"` // types seem to vary
	ImageWidth     any             `json:"ImageWidth,omitempty"`  // types seem to vary
	Infobox        any                 `json:"Infobox,omitempty"`     // this is a string or an object, we are not going to use it in search
	RelatedTopics  []RelatedTopic  `json:"RelatedTopics,omitempty"`
	Results        []Result        `json:"Results,omitempty"`
	Type           string          `json:"Type,omitempty"`
	Meta           json.RawMessage `json:"meta,omitempty"`
}

type Infobox struct {
	Content []InfoboxContent `json:"content,omitempty"`
	Meta    []InfoboxMeta    `json:"meta,omitempty"`
}

type InfoboxContent struct {
	DataType  string `json:"data_type,omitempty"`
	Label     string `json:"label,omitempty"`
	Value     any    `json:"value,omitempty"`      // can be string or an object
	WikiOrder any    `json:"wiki_order,omitempty"` // I can see value of 1,2, or "101","102"
}

type InfoboxMeta struct {
	DataType string `json:"data_type,omitempty"`
	Label    string `json:"label,omitempty"`
	Value    string `json:"value,omitempty"`
}

type RelatedTopic struct {
	FirstURL string `json:"FirstURL,omitempty"`
	Icon     Icon   `json:"Icon,omitempty"`
	Result   string `json:"Result,omitempty"`
	Text     string `json:"Text,omitempty"`
}

type Icon struct {
	Height string `json:"Height,omitempty"`
	URL    string `json:"URL,omitempty"`
	Width  string `json:"Width,omitempty"`
}

type Result struct {
	FirstURL string `json:"FirstURL,omitempty"`
	Icon     Icon   `json:"Icon,omitempty"`
	Result   string `json:"Result,omitempty"`
	Text     string `json:"Text,omitempty"`
}

func NewClient() *Client {
	return &Client{
		BaseURL: "https://api.duckduckgo.com",
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Search calls duckduckgo api and return the json as a an unmasharlled []byte.
func (c *Client) Search(query string) ([]byte, error) {
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

	if resp.StatusCode >=300 {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse body")
	}
	return body, nil
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
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse body| %w", err)
	}
	var ddgResponse Response

	fmt.Println(string(body))
	err = json.Unmarshal(body, &ddgResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarsharl payload | %w", err)
	}

	return &ddgResponse, nil
}

type SearchOptions struct {
	NoHTML       bool
	SkipDisambig bool
	NoRedirect   bool
	SafeSearch   string // "strict", "moderate", "off"
}
