package tooloutput

import (
	"strings"

	"github.com/compresr/context-gateway/internal/adapters"
	"github.com/rs/zerolog/log"
)

// categoryMapping maps generic skip_tools categories to provider-specific tool names.
// Users configure categories (lowercase): skip_tools: ["read", "edit"]
// Only verified provider mappings are included. Other providers (OpenAI, Gemini, Ollama)
// fall back to using all known tool names for the category.
// TODO: Add verified mappings for OpenAI, Gemini, Ollama when tool names are confirmed.
var categoryMapping = map[string]map[adapters.Provider][]string{
	"read": {
		adapters.ProviderAnthropic: {"Read"},
		adapters.ProviderBedrock:   {"Read"},
	},
	"edit": {
		adapters.ProviderAnthropic: {"Edit"},
		adapters.ProviderBedrock:   {"Edit"},
	},
	"write": {
		adapters.ProviderAnthropic: {"Write"},
		adapters.ProviderBedrock:   {"Write"},
	},
	"bash": {
		adapters.ProviderAnthropic: {"Bash"},
		adapters.ProviderBedrock:   {"Bash"},
	},
	"glob": {
		adapters.ProviderAnthropic: {"Glob"},
		adapters.ProviderBedrock:   {"Glob"},
	},
	"grep": {
		adapters.ProviderAnthropic: {"Grep"},
		adapters.ProviderBedrock:   {"Grep"},
	},
}

// BuildSkipSet resolves skip_tools categories to a set of provider-specific tool names.
// Categories are case-insensitive. Unknown categories are treated as exact tool names
// (backward compatible with skip_tools: ["Read", "Edit"]).
func BuildSkipSet(categories []string, provider adapters.Provider) map[string]bool {
	result := make(map[string]bool, len(categories)*2)
	for _, cat := range categories {
		lower := strings.ToLower(cat)
		if providerMap, ok := categoryMapping[lower]; ok {
			if toolNames, ok := providerMap[provider]; ok {
				for _, name := range toolNames {
					result[name] = true
				}
				continue
			}
			// Provider not in mapping — add all known tool names for this category
			log.Debug().Str("category", cat).Str("provider", string(provider)).Msg("skip_tools: provider not in mapping, using all known tool names")
			for _, toolNames := range providerMap {
				for _, name := range toolNames {
					result[name] = true
				}
			}
		} else {
			// Unknown category — use as exact tool name (backward compat)
			result[cat] = true
		}
	}
	return result
}
