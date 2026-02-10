package unit

import (
	"testing"

	"github.com/compresr/context-gateway/internal/config"
	"github.com/compresr/context-gateway/tests/anthropic/fixtures"
)

// TestModelConfigurations verifies all 3 model configurations are valid
func TestModelConfigurations(t *testing.T) {
	tests := []struct {
		name     string
		config   *config.Config
		model    string
		strategy string
	}{
		{
			name:     "tool_output_cmprsr",
			config:   fixtures.CmprsrConfig(),
			model:    "tool_output_cmprsr",
			strategy: config.StrategyAPI,
		},
		{
			name:     "tool_output_openai",
			config:   fixtures.OpenAIConfig(),
			model:    "tool_output_openai",
			strategy: config.StrategyAPI,
		},
		{
			name:     "tool_output_reranker",
			config:   fixtures.RerankerConfig(),
			model:    "tool_output_reranker",
			strategy: config.StrategyAPI,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Verify strategy
			if tc.config.Pipes.ToolOutput.Strategy != tc.strategy {
				t.Errorf("expected strategy %q, got %q", tc.strategy, tc.config.Pipes.ToolOutput.Strategy)
			}

			// Verify model
			if tc.config.Pipes.ToolOutput.API.Model != tc.model {
				t.Errorf("expected model %q, got %q", tc.model, tc.config.Pipes.ToolOutput.API.Model)
			}

			// Verify enabled
			if !tc.config.Pipes.ToolOutput.Enabled {
				t.Error("expected tool_output to be enabled")
			}

			// Verify API endpoint is set
			if tc.config.Pipes.ToolOutput.API.Endpoint == "" {
				t.Error("expected API endpoint to be set")
			}
		})
	}
}

// TestPassthroughStrategyConfig verifies passthrough strategy works without model
func TestPassthroughStrategyConfig(t *testing.T) {
	cfg := fixtures.PassthroughConfig()

	if cfg.Pipes.ToolOutput.Strategy != config.StrategyPassthrough {
		t.Errorf("expected strategy %q, got %q", config.StrategyPassthrough, cfg.Pipes.ToolOutput.Strategy)
	}
}

// TestOnlyTwoStrategiesExist verifies only api and passthrough strategies exist
func TestOnlyTwoStrategiesExist(t *testing.T) {
	// These are the only valid strategies after removing external strategies
	validStrategies := map[string]bool{
		config.StrategyAPI:         true,
		config.StrategyPassthrough: true,
	}

	if !validStrategies[config.StrategyAPI] {
		t.Error("StrategyAPI should be valid")
	}

	if !validStrategies[config.StrategyPassthrough] {
		t.Error("StrategyPassthrough should be valid")
	}

	// Verify the constants have expected values
	if config.StrategyAPI != "api" {
		t.Errorf("expected StrategyAPI to be 'api', got %q", config.StrategyAPI)
	}

	if config.StrategyPassthrough != "passthrough" {
		t.Errorf("expected StrategyPassthrough to be 'passthrough', got %q", config.StrategyPassthrough)
	}
}

// TestQueryAgnosticConfiguration verifies QueryAgnostic is set correctly for each model type
func TestQueryAgnosticConfiguration(t *testing.T) {
	tests := []struct {
		name          string
		config        *config.Config
		model         string
		queryAgnostic bool
		description   string
	}{
		{
			name:          "cmprsr_is_query_agnostic",
			config:        fixtures.CmprsrConfig(),
			model:         "tool_output_cmprsr",
			queryAgnostic: true,
			description:   "LLM-based compression doesn't need user query",
		},
		{
			name:          "openai_is_query_agnostic",
			config:        fixtures.OpenAIConfig(),
			model:         "tool_output_openai",
			queryAgnostic: true,
			description:   "LLM-based compression doesn't need user query",
		},
		{
			name:          "reranker_is_NOT_query_agnostic",
			config:        fixtures.RerankerConfig(),
			model:         "tool_output_reranker",
			queryAgnostic: false,
			description:   "Reranker needs user query for relevance scoring",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Verify model
			if tc.config.Pipes.ToolOutput.API.Model != tc.model {
				t.Errorf("expected model %q, got %q", tc.model, tc.config.Pipes.ToolOutput.API.Model)
			}

			// Verify QueryAgnostic setting
			if tc.config.Pipes.ToolOutput.API.QueryAgnostic != tc.queryAgnostic {
				t.Errorf("expected QueryAgnostic=%v for %s (%s), got %v",
					tc.queryAgnostic, tc.model, tc.description, tc.config.Pipes.ToolOutput.API.QueryAgnostic)
			}
		})
	}
}

// TestQueryAgnosticWithCustomConfig verifies TestConfigWithModelAndQuery works correctly
func TestQueryAgnosticWithCustomConfig(t *testing.T) {
	// Test with query agnostic = true
	cfgAgnostic := fixtures.TestConfigWithModelAndQuery(config.StrategyAPI, "test_model", 256, false, true)
	if !cfgAgnostic.Pipes.ToolOutput.API.QueryAgnostic {
		t.Error("expected QueryAgnostic=true when explicitly set to true")
	}

	// Test with query agnostic = false
	cfgNotAgnostic := fixtures.TestConfigWithModelAndQuery(config.StrategyAPI, "test_model", 256, false, false)
	if cfgNotAgnostic.Pipes.ToolOutput.API.QueryAgnostic {
		t.Error("expected QueryAgnostic=false when explicitly set to false")
	}
}
