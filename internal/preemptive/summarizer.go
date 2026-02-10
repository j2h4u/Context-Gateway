// Summarization service for preemptive summarization.
package preemptive

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"
)

// Summarizer generates conversation summaries.
type Summarizer struct {
	config     SummarizerConfig
	httpClient *http.Client
}

// NewSummarizer creates a new summarizer.
func NewSummarizer(cfg SummarizerConfig) *Summarizer {
	return &Summarizer{
		config:     cfg,
		httpClient: &http.Client{Timeout: cfg.Timeout},
	}
}

// SummarizeInput contains input for summarization.
type SummarizeInput struct {
	Messages         []json.RawMessage
	TriggerThreshold float64 // e.g., 80% → keep 20% of context as recent
	KeepRecentTokens int     // Fixed token count (override)
	KeepRecentCount  int     // Message-based (legacy fallback)
	Model            string  // Used to look up context window
	ContextWindow    int     // Override context window (for testing)
}

// SummarizeOutput contains the result.
type SummarizeOutput struct {
	Summary             string
	SummaryTokens       int
	LastSummarizedIndex int
	Duration            time.Duration
	InputTokens         int
	OutputTokens        int
}

// Summarize generates a summary.
func (s *Summarizer) Summarize(ctx context.Context, input SummarizeInput) (*SummarizeOutput, error) {
	startTime := time.Now()
	total := len(input.Messages)
	if total == 0 {
		return nil, fmt.Errorf("no messages to summarize")
	}

	// Determine cutoff point using token-based or message-based approach
	lastIndex, err := s.findSummarizationCutoff(input)
	if err != nil {
		return nil, err
	}

	toSummarize := input.Messages[:lastIndex+1]

	// Build request
	prompt := s.config.SystemPrompt
	if prompt == "" {
		prompt = DefaultClaudeSystemPrompt
	}

	formatted := FormatMessages(toSummarize)
	req := apiRequest{
		Model:     s.config.Model,
		MaxTokens: s.config.MaxTokens,
		System:    prompt,
		Messages: []apiMessage{{
			Role:    "user",
			Content: fmt.Sprintf("Please summarize the following conversation:\n\n%s", formatted),
		}},
	}

	resp, err := s.callAPI(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("API call failed: %w", err)
	}

	summary := extractResponseText(resp)
	if summary == "" {
		return nil, fmt.Errorf("empty summary returned")
	}

	tokens := len(summary) / 4
	if resp.Usage.OutputTokens > 0 {
		tokens = resp.Usage.OutputTokens
	}

	return &SummarizeOutput{
		Summary:             summary,
		SummaryTokens:       tokens,
		LastSummarizedIndex: lastIndex,
		Duration:            time.Since(startTime),
		InputTokens:         resp.Usage.InputTokens,
		OutputTokens:        resp.Usage.OutputTokens,
	}, nil
}

type apiRequest struct {
	Model     string       `json:"model"`
	MaxTokens int          `json:"max_tokens"`
	System    string       `json:"system,omitempty"`
	Messages  []apiMessage `json:"messages"`
}

type apiMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type apiResponse struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text,omitempty"`
	} `json:"content"`
	Usage struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

func (s *Summarizer) findSummarizationCutoff(input SummarizeInput) (int, error) {
	total := len(input.Messages)

	// Priority 1: Fixed token override (explicit config takes precedence)
	keepTokens := input.KeepRecentTokens
	if keepTokens <= 0 {
		keepTokens = s.config.KeepRecentTokens
	}
	if keepTokens > 0 {
		return s.findCutoffByTokens(input.Messages, keepTokens)
	}

	// Priority 2: Derive from trigger_threshold
	// If trigger is 80%, we keep 20% of context as recent messages
	triggerThreshold := input.TriggerThreshold
	if triggerThreshold <= 0 {
		triggerThreshold = 80.0 // default
	}

	if triggerThreshold > 0 && triggerThreshold < 100 {
		// Get context window size
		contextWindow := input.ContextWindow
		if contextWindow <= 0 && input.Model != "" {
			modelCtx := GetModelContextWindow(input.Model)
			contextWindow = modelCtx.EffectiveMax
		}
		if contextWindow <= 0 {
			contextWindow = 100000 // fallback: 100K
		}

		// keep_percent = 100 - trigger_threshold
		// If trigger at 80%, keep 20% of context window
		keepPercent := 100.0 - triggerThreshold
		keepTokensCalc := int(float64(contextWindow) * keepPercent / 100.0)
		return s.findCutoffByTokens(input.Messages, keepTokensCalc)
	}

	// Priority 3: Message-based (legacy fallback)
	keepCount := input.KeepRecentCount
	if keepCount <= 0 {
		keepCount = s.config.KeepRecentCount
	}
	if keepCount <= 0 {
		keepCount = 2 // absolute fallback
	}

	if total <= keepCount {
		return -1, fmt.Errorf("not enough messages: have %d, keeping %d", total, keepCount)
	}

	return total - keepCount - 1, nil
}

// findCutoffByTokens walks backwards through messages, accumulating tokens.
// Returns the last index to summarize (everything after is kept).
func (s *Summarizer) findCutoffByTokens(messages []json.RawMessage, keepTokens int) (int, error) {
	total := len(messages)
	if total == 0 {
		return -1, fmt.Errorf("no messages")
	}

	// Estimate tokens per message (bytes / 4 is a rough approximation)
	ratio := s.config.TokenEstimateRatio
	if ratio <= 0 {
		ratio = 4
	}

	// Walk backwards, accumulating tokens
	accumulatedTokens := 0
	cutoffIndex := -1

	for i := total - 1; i >= 0; i-- {
		msgTokens := len(messages[i]) / ratio
		accumulatedTokens += msgTokens

		// Once we've accumulated enough "recent" tokens, everything before is summarizable
		if accumulatedTokens >= keepTokens && i > 0 {
			cutoffIndex = i - 1
			break
		}
	}

	// If we went through all messages without hitting threshold,
	// check if we have at least 2 messages (need something to summarize + something to keep)
	if cutoffIndex < 0 {
		if total >= 2 {
			// Summarize all but the last message
			cutoffIndex = total - 2
		} else {
			return -1, fmt.Errorf("not enough content to summarize: %d tokens in %d messages", accumulatedTokens, total)
		}
	}

	return cutoffIndex, nil
}

func (s *Summarizer) callAPI(ctx context.Context, req apiRequest) (*apiResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	endpoint := s.config.Endpoint
	if endpoint == "" {
		endpoint = "https://api.anthropic.com/v1/messages"
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", s.config.APIKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	log.Debug().Str("model", req.Model).Int("max_tokens", req.MaxTokens).Msg("Calling summarization API")

	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned %d: %s", resp.StatusCode, string(respBody))
	}

	var response apiResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, err
	}

	return &response, nil
}

// extractResponseText extracts text from the apiResponse (local type, stays here)
func extractResponseText(resp *apiResponse) string {
	for _, block := range resp.Content {
		if block.Type == "text" {
			return block.Text
		}
	}
	return ""
}
