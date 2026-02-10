// Expander Tests - Context Expansion via Phantom Tool
//
// Tests the expand_context mechanism for restoring compressed content:
// - ParseExpandContextCalls: Extract shadow IDs from LLM responses
// - CreateExpandResultMessages: Build tool_result with original content
// - Filter functions: Remove phantom tool from client-facing responses
// Supports both Anthropic and OpenAI response formats.
package unit

import (
	"testing"

	"github.com/stretchr/testify/assert"

	tooloutput "github.com/compresr/context-gateway/internal/pipes/tool_output"
	"github.com/compresr/context-gateway/tests/anthropic/fixtures"
)

// TestExpander_ParseExpandContextCalls_Anthropic verifies Anthropic format parsing.
// Anthropic uses content blocks with type:"tool_use" and name:"expand_context".
// Input contains {"id": "shadow_xxx"} with the shadow ID to expand.
func TestExpander_ParseExpandContextCalls_Anthropic(t *testing.T) {
	st := fixtures.TestStore()
	expander := tooloutput.NewExpander(st, nil)

	response := fixtures.AnthropicResponseWithExpandCall("toolu_001", "shadow_abc123")

	calls := expander.ParseExpandContextCalls(response)

	assert.Len(t, calls, 1)
	assert.Equal(t, "toolu_001", calls[0].ToolUseID)
	assert.Equal(t, "shadow_abc123", calls[0].ShadowID)
}

func TestExpander_ParseExpandContextCalls_OpenAI(t *testing.T) {
	st := fixtures.TestStore()
	expander := tooloutput.NewExpander(st, nil)

	response := fixtures.OpenAIResponseWithExpandCall("call_001", "shadow_xyz789")

	calls := expander.ParseExpandContextCalls(response)

	assert.Len(t, calls, 1)
	assert.Equal(t, "call_001", calls[0].ToolUseID)
	assert.Equal(t, "shadow_xyz789", calls[0].ShadowID)
}

func TestExpander_ParseExpandContextCalls_NoExpandCalls(t *testing.T) {
	st := fixtures.TestStore()
	expander := tooloutput.NewExpander(st, nil)

	response := fixtures.AnthropicResponseNoExpand("Here is my response.")

	calls := expander.ParseExpandContextCalls(response)

	assert.Empty(t, calls)
}

func TestExpander_ParseExpandContextCalls_OtherToolCall(t *testing.T) {
	st := fixtures.TestStore()
	expander := tooloutput.NewExpander(st, nil)

	response := fixtures.AnthropicResponseWithOtherToolCall("toolu_002", "read_file")

	calls := expander.ParseExpandContextCalls(response)

	assert.Empty(t, calls, "should not parse non-expand_context tool calls")
}

func TestExpander_ParseExpandContextCalls_EmptyResponse(t *testing.T) {
	st := fixtures.TestStore()
	expander := tooloutput.NewExpander(st, nil)

	calls := expander.ParseExpandContextCalls([]byte{})

	assert.Empty(t, calls)
}

func TestExpander_ParseExpandContextCalls_InvalidJSON(t *testing.T) {
	st := fixtures.TestStore()
	expander := tooloutput.NewExpander(st, nil)

	calls := expander.ParseExpandContextCalls([]byte("not valid json"))

	assert.Empty(t, calls)
}

func TestExpander_CreateExpandResultMessages_Anthropic_Found(t *testing.T) {
	shadowID := "shadow_test123"
	content := "This is the full content"

	st := fixtures.PreloadedStore(map[string]string{shadowID: content})
	expander := tooloutput.NewExpander(st, nil)

	calls := []tooloutput.ExpandContextCall{
		{ToolUseID: "toolu_001", ShadowID: shadowID},
	}

	messages, found, notFound := expander.CreateExpandResultMessages(calls, true)

	assert.Equal(t, 1, found)
	assert.Equal(t, 0, notFound)
	assert.Len(t, messages, 1)
	assert.Equal(t, "user", messages[0]["role"])

	contentBlocks, ok := messages[0]["content"].([]interface{})
	assert.True(t, ok)
	assert.Len(t, contentBlocks, 1)

	block := contentBlocks[0].(map[string]interface{})
	assert.Equal(t, "tool_result", block["type"])
	assert.Equal(t, "toolu_001", block["tool_use_id"])
	assert.Equal(t, content, block["content"])
}

func TestExpander_CreateExpandResultMessages_OpenAI_Found(t *testing.T) {
	shadowID := "shadow_test456"
	content := "OpenAI full content"

	st := fixtures.PreloadedStore(map[string]string{shadowID: content})
	expander := tooloutput.NewExpander(st, nil)

	calls := []tooloutput.ExpandContextCall{
		{ToolUseID: "call_001", ShadowID: shadowID},
	}

	messages, found, notFound := expander.CreateExpandResultMessages(calls, false)

	assert.Equal(t, 1, found)
	assert.Equal(t, 0, notFound)
	assert.Len(t, messages, 1)
	assert.Equal(t, "tool", messages[0]["role"])
	assert.Equal(t, "call_001", messages[0]["tool_call_id"])
	assert.Equal(t, content, messages[0]["content"])
}

func TestExpander_CreateExpandResultMessages_NotFound(t *testing.T) {
	st := fixtures.TestStore()
	expander := tooloutput.NewExpander(st, nil)

	calls := []tooloutput.ExpandContextCall{
		{ToolUseID: "toolu_001", ShadowID: "shadow_nonexistent"},
	}

	messages, found, notFound := expander.CreateExpandResultMessages(calls, true)

	assert.Equal(t, 0, found)
	assert.Equal(t, 1, notFound)
	assert.Len(t, messages, 1)

	contentBlocks := messages[0]["content"].([]interface{})
	block := contentBlocks[0].(map[string]interface{})
	assert.Contains(t, block["content"], "not found or expired")
}

func TestNewExpander(t *testing.T) {
	st := fixtures.TestStore()

	expander := tooloutput.NewExpander(st, nil)

	assert.NotNil(t, expander)
}
