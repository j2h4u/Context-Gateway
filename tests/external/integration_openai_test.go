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

// TestOpenAI_Integration tests external provider with OpenAI format.
// Run: go test -v -run TestOpenAI ./tests/external/...
func TestOpenAI_Integration(t *testing.T) {

	t.Run("compresses_bash_output", func(t *testing.T) {
		server := newMockServer(t, "openai")
		defer server.Close()

		provider := newTestProvider(t, server.URL)
		resp, err := provider.Compress(&external.CompressRequest{
			Content:        "drwxr-xr-x 15 user staff 480 main.go\n-rw-r--r-- 1 user staff 1234 utils.go",
			ToolName:       "bash",
			UserQuery:      "list files",
			SourceProvider: "openai",
		})

		require.NoError(t, err)
		assert.NotEmpty(t, resp.Content)
		assert.Greater(t, resp.OriginalSize, 0)
	})

	t.Run("compresses_read_file_output", func(t *testing.T) {
		server := newMockServer(t, "openai")
		defer server.Close()

		provider := newTestProvider(t, server.URL)
		resp, err := provider.Compress(&external.CompressRequest{
			Content:        "package main\n\nimport \"fmt\"\n\nfunc main() {\n\tfmt.Println(\"hello\")\n}",
			ToolName:       "read_file",
			SourceProvider: "openai",
		})

		require.NoError(t, err)
		assert.NotEmpty(t, resp.Content)
	})

	t.Run("sends_correct_headers", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "openai", r.Header.Get("X-Source-Provider"))
			assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

			var req map[string]interface{}
			json.NewDecoder(r.Body).Decode(&req)
			assert.Equal(t, "openai", req["source_provider"])

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

		provider := newTestProvider(t, server.URL)
		_, err := provider.Compress(&external.CompressRequest{
			Content:        "test",
			ToolName:       "bash",
			SourceProvider: "openai",
		})
		require.NoError(t, err)
	})

	t.Run("uses_openai_model", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var req map[string]interface{}
			json.NewDecoder(r.Body).Decode(&req)
			assert.Equal(t, "TO_CMPRSR_OA_V1", req["model"])

			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": true,
				"data": map[string]interface{}{
					"content":           "compressed",
					"original_size":     100,
					"compressed_size":   50,
					"compression_ratio": 0.5,
					"model":             "TO_CMPRSR_OA_V1",
				},
			})
		}))
		defer server.Close()

		cfg := external.Config{
			BaseURL: server.URL,
			Timeout: 5 * time.Second,
			Model:   "TO_CMPRSR_OA_V1",
		}
		provider, _ := external.NewExternalProvider(cfg)

		resp, err := provider.Compress(&external.CompressRequest{
			Content:        "test",
			ToolName:       "bash",
			SourceProvider: "openai",
		})
		require.NoError(t, err)
		assert.Equal(t, "TO_CMPRSR_OA_V1", resp.Model)
	})

	t.Run("handles_empty_output", func(t *testing.T) {
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

		provider := newTestProvider(t, server.URL)
		resp, err := provider.Compress(&external.CompressRequest{
			Content:        "",
			ToolName:       "bash",
			SourceProvider: "openai",
		})
		require.NoError(t, err)
		assert.Empty(t, resp.Content)
	})

	t.Run("handles_multiple_tool_calls", func(t *testing.T) {
		server := newMockServer(t, "openai")
		defer server.Close()

		provider := newTestProvider(t, server.URL)

		tools := []string{"bash", "read_file", "list_dir"}
		for _, tool := range tools {
			resp, err := provider.Compress(&external.CompressRequest{
				Content:        "content for " + tool,
				ToolName:       tool,
				SourceProvider: "openai",
			})
			require.NoError(t, err)
			assert.NotEmpty(t, resp.Content)
		}
	})
}

func newMockServer(t *testing.T, expectedSource string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, expectedSource, r.Header.Get("X-Source-Provider"))

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
				"content":           "compressed: " + preview,
				"original_size":     originalSize,
				"compressed_size":   compressedSize,
				"compression_ratio": 0.5,
				"cache_hit":         false,
			},
		})
	}))
}

func newTestProvider(t *testing.T, baseURL string) *external.ExternalProvider {
	cfg := external.Config{
		BaseURL: baseURL,
		Timeout: 5 * time.Second,
	}
	provider, err := external.NewExternalProvider(cfg)
	require.NoError(t, err)
	return provider
}
