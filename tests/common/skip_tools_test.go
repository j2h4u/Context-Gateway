package common

import (
	"testing"

	"github.com/compresr/context-gateway/internal/adapters"
	tooloutput "github.com/compresr/context-gateway/internal/pipes/tool_output"
	"github.com/stretchr/testify/assert"
)

func TestBuildSkipSet_Anthropic(t *testing.T) {
	set := tooloutput.BuildSkipSet([]string{"read", "edit"}, adapters.ProviderAnthropic)
	assert.True(t, set["Read"])
	assert.True(t, set["Edit"])
	assert.False(t, set["Bash"])
	assert.False(t, set["file_read"])
}

func TestBuildSkipSet_OpenAI_Fallback(t *testing.T) {
	// OpenAI has no verified mapping — falls back to all known tool names for category
	set := tooloutput.BuildSkipSet([]string{"read"}, adapters.ProviderOpenAI)
	// Gets all known names (Anthropic + Bedrock both have "Read")
	assert.True(t, set["Read"])
	assert.Len(t, set, 1) // Only "Read" since Anthropic and Bedrock share the same name
}

func TestBuildSkipSet_CaseInsensitive(t *testing.T) {
	set := tooloutput.BuildSkipSet([]string{"READ", "Edit", "bAsH"}, adapters.ProviderAnthropic)
	assert.True(t, set["Read"])
	assert.True(t, set["Edit"])
	assert.True(t, set["Bash"])
}

func TestBuildSkipSet_ExactMatchFallback(t *testing.T) {
	// Unknown category treated as exact tool name (backward compat)
	set := tooloutput.BuildSkipSet([]string{"MyCustomTool"}, adapters.ProviderAnthropic)
	assert.True(t, set["MyCustomTool"])
	assert.Len(t, set, 1)
}

func TestBuildSkipSet_BackwardCompat(t *testing.T) {
	// Old-style config: skip_tools: ["Read", "Edit"]
	// "Read" lowercase = "read" category, matches; "Edit" lowercase = "edit" category, matches
	set := tooloutput.BuildSkipSet([]string{"Read", "Edit"}, adapters.ProviderAnthropic)
	assert.True(t, set["Read"])
	assert.True(t, set["Edit"])
}

func TestBuildSkipSet_UnknownProvider(t *testing.T) {
	// Unknown provider gets all tool names from all providers for the category
	set := tooloutput.BuildSkipSet([]string{"read"}, adapters.ProviderUnknown)
	assert.True(t, set["Read"])
	assert.Len(t, set, 1) // Only "Read" — Anthropic and Bedrock share the same name
}

func TestBuildSkipSet_Empty(t *testing.T) {
	set := tooloutput.BuildSkipSet(nil, adapters.ProviderAnthropic)
	assert.Empty(t, set)

	set = tooloutput.BuildSkipSet([]string{}, adapters.ProviderAnthropic)
	assert.Empty(t, set)
}

func TestBuildSkipSet_MixedCategoriesAndExact(t *testing.T) {
	set := tooloutput.BuildSkipSet([]string{"read", "MyTool"}, adapters.ProviderAnthropic)
	assert.True(t, set["Read"])
	assert.True(t, set["MyTool"])
	assert.Len(t, set, 2)
}
