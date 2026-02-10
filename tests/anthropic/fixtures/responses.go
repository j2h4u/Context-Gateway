package fixtures

import (
	"encoding/json"
	"fmt"
	"time"
)

// AnthropicResponseWithExpandCall creates an Anthropic response with expand_context tool call
func AnthropicResponseWithExpandCall(toolUseID, shadowID string) []byte {
	resp := map[string]interface{}{
		"id":   "msg_001",
		"type": "message",
		"role": "assistant",
		"content": []interface{}{
			map[string]interface{}{
				"type": "text",
				"text": "I need to see the full content.",
			},
			map[string]interface{}{
				"type":  "tool_use",
				"id":    toolUseID,
				"name":  "expand_context",
				"input": map[string]interface{}{"id": shadowID},
			},
		},
		"stop_reason": "tool_use",
	}
	data, _ := json.Marshal(resp)
	return data
}

// OpenAIResponseWithExpandCall creates an OpenAI response with expand_context tool call
func OpenAIResponseWithExpandCall(toolCallID, shadowID string) []byte {
	resp := map[string]interface{}{
		"id":      "chatcmpl-001",
		"object":  "chat.completion",
		"created": time.Now().Unix(),
		"model":   "gpt-4",
		"choices": []interface{}{
			map[string]interface{}{
				"index": 0,
				"message": map[string]interface{}{
					"role":    "assistant",
					"content": nil,
					"tool_calls": []interface{}{
						map[string]interface{}{
							"id":   toolCallID,
							"type": "function",
							"function": map[string]interface{}{
								"name":      "expand_context",
								"arguments": fmt.Sprintf(`{"id":"%s"}`, shadowID),
							},
						},
					},
				},
				"finish_reason": "tool_calls",
			},
		},
	}
	data, _ := json.Marshal(resp)
	return data
}

// AnthropicResponseNoExpand creates an Anthropic response without expand calls
func AnthropicResponseNoExpand(text string) []byte {
	resp := map[string]interface{}{
		"id":   "msg_002",
		"type": "message",
		"role": "assistant",
		"content": []interface{}{
			map[string]interface{}{
				"type": "text",
				"text": text,
			},
		},
		"stop_reason": "end_turn",
	}
	data, _ := json.Marshal(resp)
	return data
}

// OpenAIResponseNoExpand creates an OpenAI response without expand calls
func OpenAIResponseNoExpand(text string) []byte {
	resp := map[string]interface{}{
		"id":      "chatcmpl-002",
		"object":  "chat.completion",
		"created": time.Now().Unix(),
		"model":   "gpt-4",
		"choices": []interface{}{
			map[string]interface{}{
				"index": 0,
				"message": map[string]interface{}{
					"role":    "assistant",
					"content": text,
				},
				"finish_reason": "stop",
			},
		},
	}
	data, _ := json.Marshal(resp)
	return data
}

// AnthropicResponseWithOtherToolCall creates a response with a non-expand tool call
func AnthropicResponseWithOtherToolCall(toolUseID, toolName string) []byte {
	resp := map[string]interface{}{
		"id":   "msg_003",
		"type": "message",
		"role": "assistant",
		"content": []interface{}{
			map[string]interface{}{
				"type":  "tool_use",
				"id":    toolUseID,
				"name":  toolName,
				"input": map[string]interface{}{"path": "/some/file.txt"},
			},
		},
		"stop_reason": "tool_use",
	}
	data, _ := json.Marshal(resp)
	return data
}
