// Inject/Filter Tests - Phantom Tool Management
//
// Tests the expand_context tool injection and filtering:
// - InjectExpandContextTool: Add phantom tool to request when shadow refs exist
// - FilterExpandContextFromResponse: Remove phantom tool calls from LLM response
// These enable transparent context expansion without client awareness.
// Critical for C7 (transparent proxy) constraint compliance.
package unit

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	tooloutput "github.com/compresr/context-gateway/internal/pipes/tool_output"
	"github.com/compresr/context-gateway/tests/anthropic/fixtures"
)

// TestInjectExpandContextTool_WithShadowRefs verifies tool injection.
// When request has shadow refs, inject expand_context tool definition.
// LLM can then call this tool to request original content.
func TestInjectExpandContextTool_WithShadowRefs(t *testing.T) {
	body := []byte(`{"model": "claude-3", "messages": []}`)
	shadowRefs := map[string]string{
		"shadow_123": "some content",
	}

	result, err := tooloutput.InjectExpandContextTool(body, shadowRefs, "anthropic")

	require.NoError(t, err)

	var request map[string]interface{}
	err = json.Unmarshal(result, &request)
	require.NoError(t, err)

	tools, ok := request["tools"].([]interface{})
	assert.True(t, ok)
	assert.Len(t, tools, 1)

	tool := tools[0].(map[string]interface{})
	assert.Equal(t, "expand_context", tool["name"])
}

func TestInjectExpandContextTool_NoShadowRefs(t *testing.T) {
	body := []byte(`{"model": "claude-3", "messages": []}`)
	shadowRefs := map[string]string{}

	result, err := tooloutput.InjectExpandContextTool(body, shadowRefs, "anthropic")

	require.NoError(t, err)
	assert.Equal(t, body, result, "should return unchanged body")
}

func TestInjectExpandContextTool_ExistingTools(t *testing.T) {
	body := []byte(`{
		"model": "claude-3",
		"messages": [],
		"tools": [{"name": "read_file", "description": "Read a file"}]
	}`)
	shadowRefs := map[string]string{"shadow_123": "content"}

	result, err := tooloutput.InjectExpandContextTool(body, shadowRefs, "anthropic")

	require.NoError(t, err)

	var request map[string]interface{}
	err = json.Unmarshal(result, &request)
	require.NoError(t, err)

	tools := request["tools"].([]interface{})
	assert.Len(t, tools, 2, "should append expand_context to existing tools")

	expandTool := tools[1].(map[string]interface{})
	assert.Equal(t, "expand_context", expandTool["name"])
}

func TestInjectExpandContextTool_AlreadyExists(t *testing.T) {
	body := []byte(`{
		"model": "claude-3",
		"messages": [],
		"tools": [{"name": "expand_context", "description": "Already exists"}]
	}`)
	shadowRefs := map[string]string{"shadow_123": "content"}

	result, err := tooloutput.InjectExpandContextTool(body, shadowRefs, "anthropic")

	require.NoError(t, err)

	var request map[string]interface{}
	err = json.Unmarshal(result, &request)
	require.NoError(t, err)

	tools := request["tools"].([]interface{})
	assert.Len(t, tools, 1, "should not duplicate expand_context tool")
}

func TestExpander_FilterExpandContextFromResponse_Anthropic(t *testing.T) {
	st := fixtures.TestStore()
	expander := tooloutput.NewExpander(st, nil)

	response := fixtures.AnthropicResponseWithExpandCall("toolu_001", "shadow_123")

	filtered, modified := expander.FilterExpandContextFromResponse(response)

	assert.True(t, modified)

	var result map[string]interface{}
	err := json.Unmarshal(filtered, &result)
	require.NoError(t, err)

	content := result["content"].([]interface{})
	for _, block := range content {
		b := block.(map[string]interface{})
		if b["type"] == "tool_use" {
			name, _ := b["name"].(string)
			assert.NotEqual(t, "expand_context", name, "expand_context should be filtered")
		}
	}
}

func TestExpander_FilterExpandContextFromResponse_OpenAI(t *testing.T) {
	st := fixtures.TestStore()
	expander := tooloutput.NewExpander(st, nil)

	response := fixtures.OpenAIResponseWithExpandCall("call_001", "shadow_123")

	filtered, modified := expander.FilterExpandContextFromResponse(response)

	assert.True(t, modified)

	var result map[string]interface{}
	err := json.Unmarshal(filtered, &result)
	require.NoError(t, err)

	choices := result["choices"].([]interface{})
	choice := choices[0].(map[string]interface{})
	message := choice["message"].(map[string]interface{})

	// tool_calls should be empty or nil after filtering the only expand_context call
	toolCalls, ok := message["tool_calls"].([]interface{})
	if ok {
		assert.Empty(t, toolCalls, "expand_context tool calls should be filtered")
	}
}

func TestExpander_FilterExpandContextFromResponse_NoExpand(t *testing.T) {
	st := fixtures.TestStore()
	expander := tooloutput.NewExpander(st, nil)

	response := fixtures.AnthropicResponseNoExpand("Just text response")

	filtered, modified := expander.FilterExpandContextFromResponse(response)

	assert.False(t, modified)
	assert.Equal(t, response, filtered)
}

func TestExpander_AppendMessagesToRequest_Anthropic(t *testing.T) {
	st := fixtures.TestStore()
	expander := tooloutput.NewExpander(st, nil)

	body := []byte(`{"model": "claude-3", "messages": [{"role": "user", "content": "Hi"}]}`)
	assistantResponse := fixtures.AnthropicResponseWithExpandCall("toolu_001", "shadow_123")
	toolResults := []map[string]interface{}{
		{
			"role": "user",
			"content": []interface{}{
				map[string]interface{}{
					"type":        "tool_result",
					"tool_use_id": "toolu_001",
					"content":     "expanded content",
				},
			},
		},
	}

	result, err := expander.AppendMessagesToRequest(body, assistantResponse, toolResults)

	require.NoError(t, err)

	var request map[string]interface{}
	err = json.Unmarshal(result, &request)
	require.NoError(t, err)

	messages := request["messages"].([]interface{})
	assert.Len(t, messages, 3)
}
