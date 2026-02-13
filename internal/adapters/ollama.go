package adapters

import (
	"encoding/json"
)

// OllamaAdapter handles Ollama API format requests.
// Ollama uses the OpenAI Chat Completions format for requests (messages[], tool_calls[], role: tool),
// so this adapter embeds OpenAIAdapter and delegates all request-side methods.
// The only difference is the response format — Ollama uses prompt_eval_count/eval_count
// instead of OpenAI's prompt_tokens/completion_tokens.
type OllamaAdapter struct {
	BaseAdapter
	*OpenAIAdapter
}

// NewOllamaAdapter creates a new Ollama adapter.
func NewOllamaAdapter() *OllamaAdapter {
	return &OllamaAdapter{
		BaseAdapter: BaseAdapter{
			name:     "ollama",
			provider: ProviderOllama,
		},
		OpenAIAdapter: NewOpenAIAdapter(),
	}
}

// Name returns the adapter name (overrides embedded OpenAIAdapter.Name).
func (a *OllamaAdapter) Name() string {
	return a.BaseAdapter.Name()
}

// Provider returns the provider type (overrides embedded OpenAIAdapter.Provider).
func (a *OllamaAdapter) Provider() Provider {
	return a.BaseAdapter.Provider()
}

// ExtractUsage extracts token usage from Ollama API response.
// Ollama format: {"prompt_eval_count": N, "eval_count": N}
// Also supports OpenAI format as fallback (some Ollama versions return it).
func (a *OllamaAdapter) ExtractUsage(responseBody []byte) UsageInfo {
	if len(responseBody) == 0 {
		return UsageInfo{}
	}

	// Try Ollama-native format first
	var resp struct {
		PromptEvalCount int `json:"prompt_eval_count"`
		EvalCount       int `json:"eval_count"`
	}
	if err := json.Unmarshal(responseBody, &resp); err != nil {
		return UsageInfo{}
	}

	if resp.PromptEvalCount > 0 || resp.EvalCount > 0 {
		return UsageInfo{
			InputTokens:  resp.PromptEvalCount,
			OutputTokens: resp.EvalCount,
			TotalTokens:  resp.PromptEvalCount + resp.EvalCount,
		}
	}

	// Fallback to OpenAI format (some Ollama versions return it)
	return a.OpenAIAdapter.ExtractUsage(responseBody)
}

// Ensure OllamaAdapter implements Adapter
var _ Adapter = (*OllamaAdapter)(nil)
