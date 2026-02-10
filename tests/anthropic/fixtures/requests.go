// Package fixtures provides test data generators for gateway tests.
package fixtures

import (
	"strings"
)

// LargeToolOutput generates a tool output of specified size.
func LargeToolOutput(sizeBytes int) string {
	// Generate a realistic looking file content
	line := "func processData(input string) (string, error) { return input, nil }\n"
	var sb strings.Builder
	sb.WriteString("package main\n\n")
	for sb.Len() < sizeBytes {
		sb.WriteString(line)
	}
	return sb.String()[:sizeBytes]
}

// SmallToolOutput generates a small tool output.
func SmallToolOutput() string {
	return "package main\n\nfunc main() {}"
}

// MediumToolOutput generates a medium-sized tool output (1KB).
func MediumToolOutput() string {
	return LargeToolOutput(1024)
}

// RequestWithLargeToolOutput creates a request with a large tool output.
func RequestWithLargeToolOutput(sizeBytes int) []byte {
	return SingleToolOutputRequest(LargeToolOutput(sizeBytes))
}

// RequestWithSmallToolOutput creates a request with a small tool output.
func RequestWithSmallToolOutput() []byte {
	return SingleToolOutputRequest(SmallToolOutput())
}

// RequestWithSingleToolOutput creates a request with a single tool output (alias for SingleToolOutputRequest).
func RequestWithSingleToolOutput(content string) []byte {
	return SingleToolOutputRequest(content)
}
