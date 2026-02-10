package external_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/compresr/context-gateway/external"
)

// TestAnthropic_Integration tests external provider with Anthropic format.
// Run: go test -v -run TestAnthropic ./tests/external/...
func TestAnthropic_Integration(t *testing.T) {

	t.Run("compresses_bash_tool_result", func(t *testing.T) {
		server := newAnthropicMockServer(t)
		defer server.Close()

		provider := newAnthropicTestProvider(t, server.URL)
		resp, err := provider.Compress(&external.CompressRequest{
			Content:        "$ ls -la\ntotal 128\ndrwxr-xr-x 15 user staff 480 main.go",
			ToolName:       "bash",
			UserQuery:      "run ls in project dir",
			SourceProvider: "anthropic",
		})

		require.NoError(t, err)
		assert.NotEmpty(t, resp.Content)
		assert.Greater(t, resp.OriginalSize, 0)
	})

	t.Run("compresses_str_replace_editor_output", func(t *testing.T) {
		server := newAnthropicMockServer(t)
		defer server.Close()

		provider := newAnthropicTestProvider(t, server.URL)
		resp, err := provider.Compress(&external.CompressRequest{
			Content:        "File: handler.go\n\npackage main\n\nfunc Handle() {\n\t// code\n}",
			ToolName:       "str_replace_editor",
			SourceProvider: "anthropic",
		})

		require.NoError(t, err)
		assert.NotEmpty(t, resp.Content)
	})

	t.Run("sends_anthropic_headers", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "anthropic", r.Header.Get("X-Source-Provider"))
			assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

			var req map[string]interface{}
			json.NewDecoder(r.Body).Decode(&req)
			assert.Equal(t, "anthropic", req["source_provider"])

			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": true,
				"data": map[string]interface{}{
					"content":           "compressed",
					"original_size":     100,
					"compressed_size":   50,
					"compression_ratio": 0.5,
				},
			})
		}))
		defer server.Close()

		provider := newAnthropicTestProvider(t, server.URL)
		_, err := provider.Compress(&external.CompressRequest{
			Content:        "test",
			ToolName:       "bash",
			SourceProvider: "anthropic",
		})
		require.NoError(t, err)
	})

	t.Run("uses_anthropic_model", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var req map[string]interface{}
			json.NewDecoder(r.Body).Decode(&req)
			assert.Equal(t, "TO_CMPRSR_AN_V1", req["model"])

			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": true,
				"data": map[string]interface{}{
					"content":           "compressed",
					"original_size":     100,
					"compressed_size":   50,
					"compression_ratio": 0.5,
					"model":             "TO_CMPRSR_AN_V1",
				},
			})
		}))
		defer server.Close()

		cfg := external.Config{
			BaseURL: server.URL,
			Timeout: 5 * time.Second,
			Model:   "TO_CMPRSR_AN_V1",
		}
		provider, _ := external.NewExternalProvider(cfg)

		resp, err := provider.Compress(&external.CompressRequest{
			Content:        "test",
			ToolName:       "bash",
			SourceProvider: "anthropic",
		})
		require.NoError(t, err)
		assert.Equal(t, "TO_CMPRSR_AN_V1", resp.Model)
	})

	t.Run("handles_empty_tool_result", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": true,
				"data": map[string]interface{}{
					"content":           "",
					"original_size":     0,
					"compressed_size":   0,
					"compression_ratio": 1.0,
				},
			})
		}))
		defer server.Close()

		provider := newAnthropicTestProvider(t, server.URL)
		resp, err := provider.Compress(&external.CompressRequest{
			Content:        "",
			ToolName:       "bash",
			SourceProvider: "anthropic",
		})
		require.NoError(t, err)
		assert.Empty(t, resp.Content)
	})

	t.Run("handles_tool_error_result", func(t *testing.T) {
		server := newAnthropicMockServer(t)
		defer server.Close()

		provider := newAnthropicTestProvider(t, server.URL)
		resp, err := provider.Compress(&external.CompressRequest{
			Content:        "Error: command not found: invalid_cmd\nexit status 127",
			ToolName:       "bash",
			SourceProvider: "anthropic",
		})
		require.NoError(t, err)
		assert.NotEmpty(t, resp.Content)
	})

	t.Run("handles_sequential_tool_results", func(t *testing.T) {
		server := newAnthropicMockServer(t)
		defer server.Close()

		provider := newAnthropicTestProvider(t, server.URL)

		// Anthropic uses sequential tool_use -> tool_result
		tools := []string{"bash", "str_replace_editor", "grep"}
		for _, tool := range tools {
			resp, err := provider.Compress(&external.CompressRequest{
				Content:        "result for " + tool,
				ToolName:       tool,
				SourceProvider: "anthropic",
			})
			require.NoError(t, err)
			assert.NotEmpty(t, resp.Content)
		}
	})
}

func newAnthropicMockServer(t *testing.T) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "anthropic", r.Header.Get("X-Source-Provider"))

		var req map[string]interface{}
		json.NewDecoder(r.Body).Decode(&req)

		content := ""
		if c, ok := req["content"].(string); ok {
			content = c
		}
		originalSize := len(content)
		compressedSize := originalSize / 2
		if compressedSize < 5 {
			compressedSize = originalSize
		}

		preview := content
		if len(preview) > 20 {
			preview = preview[:20]
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"data": map[string]interface{}{
				"content":           "anthropic compressed: " + preview,
				"original_size":     originalSize,
				"compressed_size":   compressedSize,
				"compression_ratio": 0.5,
				"cache_hit":         false,
			},
		})
	}))
}

func newAnthropicTestProvider(t *testing.T, baseURL string) *external.ExternalProvider {
	cfg := external.Config{
		BaseURL: baseURL,
		Timeout: 5 * time.Second,
		Model:   "TO_CMPRSR_AN_V1",
	}
	provider, err := external.NewExternalProvider(cfg)
	require.NoError(t, err)
	return provider
}
