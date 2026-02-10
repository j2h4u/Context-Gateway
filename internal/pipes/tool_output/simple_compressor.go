// Simple compressor for testing expand_context behavior.
//
// Strategy: Keep only first N words to simulate aggressive compression.
// This makes it easy to trigger expand_context calls when LLM needs more detail.
package tooloutput

import (
	"strings"
)

// compressSimple keeps only the first N words from content.
// This is a VERY aggressive compression strategy meant for testing expand_context.
func (p *Pipe) compressSimple(content string, maxWords int) string {
	if maxWords <= 0 {
		maxWords = 10 // Default: keep first 10 words
	}

	words := strings.Fields(content)
	if len(words) <= maxWords {
		return content // Already short enough
	}

	// Keep first N words and add ellipsis
	truncated := strings.Join(words[:maxWords], " ")
	return truncated + "..."
}

// CompressSimpleContent is the public wrapper for simple compression.
// Used by Process() when strategy = "simple".
func (p *Pipe) CompressSimpleContent(content string) string {
	// Configurable word count - default 10
	maxWords := 10
	if p.minBytes > 0 {
		// Use minBytes as word count if set (hacky but simple)
		maxWords = p.minBytes / 100 // e.g., minBytes=500 → 5 words
		if maxWords < 5 {
			maxWords = 5
		}
		if maxWords > 50 {
			maxWords = 50
		}
	}

	return p.compressSimple(content, maxWords)
}
