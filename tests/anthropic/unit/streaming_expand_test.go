// Tests for streaming expand_context functionality (Alternative 2: Selective Replace).
//
// These tests verify:
// - RewriteHistoryWithExpansion: Replaces compressed tool outputs with full content
// - InvalidateExpandedMappings: Cache invalidation after expansion
// - StreamBuffer detection of expand_context in SSE
package unit

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	tooloutput "github.com/compresr/context-gateway/internal/pipes/tool_output"
	"github.com/compresr/context-gateway/internal/store"
)

// TestRewriteHistoryWithExpansion_Anthropic_SingleTool verifies single tool expansion.
func TestRewriteHistoryWithExpansion_Anthropic_SingleTool(t *testing.T) {
	st := store.NewMemoryStore(60 * time.Second)
	defer st.Close()
	expander := tooloutput.NewExpander(st, nil)

	// Store original content
	shadowID := "shadow_abc123"
	originalContent := "This is the full original content that was compressed"
	st.Set(shadowID, originalContent)

	// Build request with compressed tool result
	request := map[string]interface{}{
		"model": "claude-3-sonnet",
		"messages": []interface{}{
			map[string]interface{}{
				"role":    "user",
				"content": "Read the file",
			},
			map[string]interface{}{
				"role": "assistant",
				"content": []interface{}{
					map[string]interface{}{
						"type": "tool_use",
						"id":   "toolu_123",
						"name": "read_file",
						"input": map[string]interface{}{
							"path": "/test.txt",
						},
					},
				},
			},
			map[string]interface{}{
				"role": "user",
				"content": []interface{}{
					map[string]interface{}{
						"type":        "tool_result",
						"tool_use_id": "toolu_123",
						"content":     "<<<SHADOW:shadow_abc123>>>\nCompressed version",
					},
				},
			},
		},
	}

	body, err := json.Marshal(request)
	require.NoError(t, err)

	expandCalls := []tooloutput.ExpandContextCall{
		{ToolUseID: "expand_123", ShadowID: shadowID},
	}

	// Rewrite history
	rewritten, expandedIDs, err := expander.RewriteHistoryWithExpansion(body, expandCalls)
	require.NoError(t, err)
	require.Len(t, expandedIDs, 1)
	assert.Equal(t, shadowID, expandedIDs[0])

	// Verify the content was replaced
	var result map[string]interface{}
	err = json.Unmarshal(rewritten, &result)
	require.NoError(t, err)

	messages := result["messages"].([]interface{})
	toolResult := messages[2].(map[string]interface{})
	content := toolResult["content"].([]interface{})
	block := content[0].(map[string]interface{})

	assert.Equal(t, originalContent, block["content"])
}

// TestRewriteHistoryWithExpansion_Anthropic_SelectiveMultiple verifies selective expansion.
func TestRewriteHistoryWithExpansion_Anthropic_SelectiveMultiple(t *testing.T) {
	st := store.NewMemoryStore(60 * time.Second)
	defer st.Close()
	expander := tooloutput.NewExpander(st, nil)

	// Store original content for multiple tools
	shadowID1 := "shadow_tool1"
	shadowID2 := "shadow_tool2"
	shadowID3 := "shadow_tool3"
	st.Set(shadowID1, "Original content for tool 1")
	st.Set(shadowID2, "Original content for tool 2")
	st.Set(shadowID3, "Original content for tool 3")

	// Build request with multiple compressed tool results
	request := map[string]interface{}{
		"model": "claude-3-sonnet",
		"messages": []interface{}{
			map[string]interface{}{
				"role": "user",
				"content": []interface{}{
					map[string]interface{}{
						"type":        "tool_result",
						"tool_use_id": "toolu_1",
						"content":     "<<<SHADOW:shadow_tool1>>>\nCompressed 1",
					},
					map[string]interface{}{
						"type":        "tool_result",
						"tool_use_id": "toolu_2",
						"content":     "<<<SHADOW:shadow_tool2>>>\nCompressed 2",
					},
					map[string]interface{}{
						"type":        "tool_result",
						"tool_use_id": "toolu_3",
						"content":     "<<<SHADOW:shadow_tool3>>>\nCompressed 3",
					},
				},
			},
		},
	}

	body, err := json.Marshal(request)
	require.NoError(t, err)

	// Only request expansion for tool2 - tool1 and tool3 should remain compressed
	expandCalls := []tooloutput.ExpandContextCall{
		{ToolUseID: "expand_tool2", ShadowID: shadowID2},
	}

	rewritten, expandedIDs, err := expander.RewriteHistoryWithExpansion(body, expandCalls)
	require.NoError(t, err)
	require.Len(t, expandedIDs, 1)
	assert.Equal(t, shadowID2, expandedIDs[0])

	// Verify only tool2 was expanded
	var result map[string]interface{}
	err = json.Unmarshal(rewritten, &result)
	require.NoError(t, err)

	messages := result["messages"].([]interface{})
	userMsg := messages[0].(map[string]interface{})
	content := userMsg["content"].([]interface{})

	// Tool 1 should still be compressed
	tool1 := content[0].(map[string]interface{})
	assert.Contains(t, tool1["content"], "<<<SHADOW:shadow_tool1>>>")

	// Tool 2 should be expanded
	tool2 := content[1].(map[string]interface{})
	assert.Equal(t, "Original content for tool 2", tool2["content"])

	// Tool 3 should still be compressed
	tool3 := content[2].(map[string]interface{})
	assert.Contains(t, tool3["content"], "<<<SHADOW:shadow_tool3>>>")
}

// TestRewriteHistoryWithExpansion_OpenAI verifies OpenAI format support.
func TestRewriteHistoryWithExpansion_OpenAI(t *testing.T) {
	st := store.NewMemoryStore(60 * time.Second)
	defer st.Close()
	expander := tooloutput.NewExpander(st, nil)

	shadowID := "shadow_openai123"
	originalContent := "Full content for OpenAI tool"
	st.Set(shadowID, originalContent)

	// OpenAI format request
	request := map[string]interface{}{
		"model": "gpt-4",
		"messages": []interface{}{
			map[string]interface{}{
				"role":    "user",
				"content": "Read the file",
			},
			map[string]interface{}{
				"role": "assistant",
				"tool_calls": []interface{}{
					map[string]interface{}{
						"id":   "call_123",
						"type": "function",
						"function": map[string]interface{}{
							"name":      "read_file",
							"arguments": `{"path": "/test.txt"}`,
						},
					},
				},
			},
			map[string]interface{}{
				"role":         "tool",
				"tool_call_id": "call_123",
				"content":      "<<<SHADOW:shadow_openai123>>>\nCompressed content",
			},
		},
	}

	body, err := json.Marshal(request)
	require.NoError(t, err)

	expandCalls := []tooloutput.ExpandContextCall{
		{ToolUseID: "expand_456", ShadowID: shadowID},
	}

	rewritten, expandedIDs, err := expander.RewriteHistoryWithExpansion(body, expandCalls)
	require.NoError(t, err)
	require.Len(t, expandedIDs, 1)

	var result map[string]interface{}
	err = json.Unmarshal(rewritten, &result)
	require.NoError(t, err)

	messages := result["messages"].([]interface{})
	toolMsg := messages[2].(map[string]interface{})

	assert.Equal(t, originalContent, toolMsg["content"])
}

// TestRewriteHistoryWithExpansion_NoShadowMatch verifies graceful handling.
func TestRewriteHistoryWithExpansion_NoShadowMatch(t *testing.T) {
	st := store.NewMemoryStore(60 * time.Second)
	defer st.Close()
	expander := tooloutput.NewExpander(st, nil)

	request := map[string]interface{}{
		"model": "claude-3-sonnet",
		"messages": []interface{}{
			map[string]interface{}{
				"role": "user",
				"content": []interface{}{
					map[string]interface{}{
						"type":        "tool_result",
						"tool_use_id": "toolu_123",
						"content":     "<<<SHADOW:shadow_missing>>>\nCompressed version",
					},
				},
			},
		},
	}

	body, err := json.Marshal(request)
	require.NoError(t, err)

	expandCalls := []tooloutput.ExpandContextCall{
		{ToolUseID: "expand_123", ShadowID: "shadow_missing"},
	}

	// Should not fail, just not expand
	rewritten, expandedIDs, err := expander.RewriteHistoryWithExpansion(body, expandCalls)
	require.NoError(t, err)
	assert.Empty(t, expandedIDs)

	// Content should remain unchanged
	var result map[string]interface{}
	err = json.Unmarshal(rewritten, &result)
	require.NoError(t, err)

	messages := result["messages"].([]interface{})
	userMsg := messages[0].(map[string]interface{})
	content := userMsg["content"].([]interface{})
	block := content[0].(map[string]interface{})
	assert.Contains(t, block["content"], "<<<SHADOW:shadow_missing>>>")
}

// TestRewriteHistoryWithExpansion_EmptyExpandCalls verifies no-op behavior.
func TestRewriteHistoryWithExpansion_EmptyExpandCalls(t *testing.T) {
	st := store.NewMemoryStore(60 * time.Second)
	defer st.Close()
	expander := tooloutput.NewExpander(st, nil)

	request := map[string]interface{}{
		"model":    "claude-3-sonnet",
		"messages": []interface{}{},
	}

	body, err := json.Marshal(request)
	require.NoError(t, err)

	rewritten, expandedIDs, err := expander.RewriteHistoryWithExpansion(body, nil)
	require.NoError(t, err)
	assert.Empty(t, expandedIDs)
	assert.Equal(t, body, rewritten)
}

// TestInvalidateExpandedMappings verifies cache invalidation.
func TestInvalidateExpandedMappings(t *testing.T) {
	st := store.NewMemoryStore(60 * time.Second)
	defer st.Close()
	expander := tooloutput.NewExpander(st, nil)

	// Store compressed content
	shadowID := "shadow_toclear"
	st.SetCompressed(shadowID, "compressed version")

	// Verify it exists
	_, found := st.GetCompressed(shadowID)
	assert.True(t, found)

	// Invalidate
	expander.InvalidateExpandedMappings([]string{shadowID})

	// Verify it's gone
	_, found = st.GetCompressed(shadowID)
	assert.False(t, found)
}

// TestStreamBuffer_DetectExpandContext verifies SSE parsing.
func TestStreamBuffer_DetectExpandContext(t *testing.T) {
	buffer := tooloutput.NewStreamBuffer()

	// Simulate Anthropic streaming chunks with expand_context
	contentBlockStart := `data: {"type":"content_block_start","index":0,"content_block":{"type":"tool_use","id":"toolu_expand","name":"expand_context"}}` + "\n\n"
	contentBlockDelta := `data: {"type":"content_block_delta","index":0,"delta":{"type":"input_json_delta","partial_json":"{\"id\":\"shadow_test123\"}"}}` + "\n\n"
	contentBlockStop := `data: {"type":"content_block_stop","index":0}` + "\n\n"

	// Process chunks
	buffer.ProcessChunk([]byte(contentBlockStart))
	buffer.ProcessChunk([]byte(contentBlockDelta))
	buffer.ProcessChunk([]byte(contentBlockStop))

	// Should have detected and suppressed the expand_context call
	calls := buffer.GetSuppressedCalls()
	require.Len(t, calls, 1)
	assert.Equal(t, "toolu_expand", calls[0].ToolUseID)
	assert.True(t, buffer.HasSuppressedCalls())
}

// TestStreamBuffer_NoExpandContext verifies passthrough for normal tools.
func TestStreamBuffer_NoExpandContext(t *testing.T) {
	buffer := tooloutput.NewStreamBuffer()

	// Normal tool call (not expand_context)
	contentBlockStart := `data: {"type":"content_block_start","index":0,"content_block":{"type":"tool_use","id":"toolu_normal","name":"read_file"}}` + "\n\n"
	contentBlockStop := `data: {"type":"content_block_stop","index":0}` + "\n\n"

	// Process chunks - should pass through
	out1, _ := buffer.ProcessChunk([]byte(contentBlockStart))
	out2, _ := buffer.ProcessChunk([]byte(contentBlockStop))

	// Should have output (not suppressed)
	assert.NotNil(t, out1)
	assert.NotNil(t, out2)

	// No suppressed calls
	assert.False(t, buffer.HasSuppressedCalls())
}
