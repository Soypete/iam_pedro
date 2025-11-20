package agent

import (
	"context"
	"testing"

	"github.com/Soypete/twitch-llm-bot/duckduckgo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tmc/langchaingo/llms"
)

func TestNewWebSearchTool(t *testing.T) {
	ddgClient := duckduckgo.NewClient()
	tool := NewWebSearchTool(ddgClient)

	assert.NotNil(t, tool)
	assert.Equal(t, "web_search", tool.Name())
	assert.NotEmpty(t, tool.Description())
}

func TestGetWebSearchToolDefinition(t *testing.T) {
	toolDef := GetWebSearchToolDefinition()

	assert.Equal(t, "function", toolDef.Type)
	assert.NotNil(t, toolDef.Function)
	assert.Equal(t, "web_search", toolDef.Function.Name)
	assert.NotEmpty(t, toolDef.Function.Description)
	assert.NotNil(t, toolDef.Function.Parameters)

	// Verify parameters structure
	params, ok := toolDef.Function.Parameters.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "object", params["type"])

	// Verify required fields
	required, ok := params["required"].([]string)
	require.True(t, ok)
	assert.Contains(t, required, "query")
}

func TestParseWebSearchToolCall_Success(t *testing.T) {
	toolCall := llms.ToolCall{
		FunctionCall: &llms.FunctionCall{
			Name:      "web_search",
			Arguments: `{"query":"golang best practices"}`,
		},
	}

	query, err := ParseWebSearchToolCall(toolCall)

	require.NoError(t, err)
	assert.Equal(t, "golang best practices", query)
}

func TestParseWebSearchToolCall_WrongTool(t *testing.T) {
	toolCall := llms.ToolCall{
		FunctionCall: &llms.FunctionCall{
			Name:      "other_tool",
			Arguments: `{"query":"test"}`,
		},
	}

	_, err := ParseWebSearchToolCall(toolCall)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected tool call")
}

func TestParseWebSearchToolCall_InvalidJSON(t *testing.T) {
	toolCall := llms.ToolCall{
		FunctionCall: &llms.FunctionCall{
			Name:      "web_search",
			Arguments: `{invalid json}`,
		},
	}

	_, err := ParseWebSearchToolCall(toolCall)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse tool call arguments")
}

func TestParseWebSearchToolCall_EmptyQuery(t *testing.T) {
	toolCall := llms.ToolCall{
		FunctionCall: &llms.FunctionCall{
			Name:      "web_search",
			Arguments: `{"query":""}`,
		},
	}

	_, err := ParseWebSearchToolCall(toolCall)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "query cannot be empty")
}

func TestCreateWebSearchAgent(t *testing.T) {
	ddgClient := duckduckgo.NewClient()
	agent := CreateWebSearchAgent(ddgClient)

	assert.NotNil(t, agent)
	assert.Equal(t, "web_search", agent.Name())
	assert.NotEmpty(t, agent.Description())

	// Verify it's the same type as NewWebSearchTool
	tool := NewWebSearchTool(ddgClient)
	assert.IsType(t, tool, agent)
}

func TestWebSearchTool_Call(t *testing.T) {
	// Skip this test if running in CI without network access
	t.Skip("Skipping integration test - requires network access")

	// Note: This test requires a real DuckDuckGo client
	// In a real test suite, you might want to mock this
	ddgClient := duckduckgo.NewClient()
	tool := NewWebSearchTool(ddgClient)

	// For now, just verify the tool can be called without panicking
	// A full integration test would verify the actual search results
	ctx := context.Background()

	result, err := tool.Call(ctx, "golang")
	assert.NoError(t, err)
	assert.NotEmpty(t, result)
}
