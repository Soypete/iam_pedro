package duckduckgo

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewClient(t *testing.T) {
	client := NewClient()
	if client == nil {
		t.Fatal("NewClient() returned nil")
	}
	if client.BaseURL != "https://api.duckduckgo.com" {
		t.Errorf("Expected BaseURL to be 'https://api.duckduckgo.com', got '%s'", client.BaseURL)
	}
	if client.HTTPClient == nil {
		t.Error("Expected HTTPClient to be initialized")
	}
}

func TestSearch(t *testing.T) {
	mockResponse := Response{
		Abstract:       "Go is a programming language.",
		AbstractSource: "Wikipedia",
		Heading:        "Go (programming language)",
		Type:           "A",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("q") != "golang" {
			t.Errorf("Expected query 'golang', got '%s'", r.URL.Query().Get("q"))
		}
		if r.URL.Query().Get("format") != "json" {
			t.Errorf("Expected format 'json', got '%s'", r.URL.Query().Get("format"))
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(mockResponse); err != nil {
			t.Fatalf("Failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	client := &Client{
		BaseURL:    server.URL,
		HTTPClient: &http.Client{},
	}

	response, err := client.Search("golang")
	if err != nil {
		t.Fatalf("Search() returned error: %v", err)
	}

	if response.Abstract != mockResponse.Abstract {
		t.Errorf("Expected Abstract '%s', got '%s'", mockResponse.Abstract, response.Abstract)
	}
	if response.Heading != mockResponse.Heading {
		t.Errorf("Expected Heading '%s', got '%s'", mockResponse.Heading, response.Heading)
	}
}

func TestSearchEmptyQuery(t *testing.T) {
	client := NewClient()
	_, err := client.Search("")
	if err == nil {
		t.Error("Expected error for empty query, got nil")
	}
	if err.Error() != "query cannot be empty" {
		t.Errorf("Expected error 'query cannot be empty', got '%s'", err.Error())
	}
}

func TestSearchWithOptions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		if query.Get("no_html") != "1" {
			t.Error("Expected no_html=1")
		}
		if query.Get("skip_disambig") != "1" {
			t.Error("Expected skip_disambig=1")
		}
		if query.Get("safe_search") != "strict" {
			t.Error("Expected safe_search=strict")
		}

		mockResponse := Response{Type: "A"}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(mockResponse); err != nil {
			t.Fatalf("Failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	client := &Client{
		BaseURL:    server.URL,
		HTTPClient: &http.Client{},
	}

	options := SearchOptions{
		NoHTML:       true,
		SkipDisambig: true,
		SafeSearch:   "strict",
	}

	_, err := client.SearchWithOptions("test query", options)
	if err != nil {
		t.Fatalf("SearchWithOptions() returned error: %v", err)
	}
}

func TestSearchHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := &Client{
		BaseURL:    server.URL,
		HTTPClient: &http.Client{},
	}

	_, err := client.Search("test")
	if err == nil {
		t.Error("Expected error for HTTP 500, got nil")
	}
}

func TestSearchInvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write([]byte("invalid json")); err != nil {
			t.Fatalf("Failed to write response: %v", err)
		}
	}))
	defer server.Close()

	client := &Client{
		BaseURL:    server.URL,
		HTTPClient: &http.Client{},
	}

	_, err := client.Search("test")
	if err == nil {
		t.Error("Expected error for invalid JSON, got nil")
	}
}