# Anthropic Integration Tests

Real API integration tests for the Context Gateway with Anthropic Claude models.

## Requirements

- `ANTHROPIC_API_KEY` environment variable set in `.env`
- Network connectivity to Anthropic API

## Test Structure

```
tests/anthropic/integration/
├── e2e_test.go                    # End-to-end Claude Code tests (Sonnet)
├── expand_behavior_test.go        # expand_context behavior tests (Haiku)
└── hard_integration_test.go       # Edge case & stress tests (Sonnet)
```

## Test Tree

### e2e_test.go - End-to-End Tests (21 tests)
Basic gateway functionality with real Claude API.

```
TestE2E_ClaudeCode
├── SimpleChat                     # Basic message round-trip
├── UsageExtraction                # Token usage tracking
├── SmallToolResult                # Tool result below compression threshold
├── LargeToolResultCompression     # Large content compression
├── MultipleToolResults            # Multiple tool outputs in one request
├── DirectoryListing               # ls/dir command output
├── BashCommandOutput              # Shell command results
├── SearchResults                  # grep/find output compression
├── ErrorToolResult                # Error handling in tool results
├── LongConversation               # Multi-turn conversation
├── CompareDirectVsProxy           # Direct API vs gateway comparison
├── WriteFileTool                  # File write tool output
├── LargeBashOutputCompression     # Large bash output
├── CacheHit                       # Shadow reference caching
├── GitDiffOutput                  # Git diff compression
├── JSONToolOutput                 # JSON content handling
├── HealthCheck                    # Gateway health endpoint
├── LargeSearchResultsCompression  # Large search results
├── WithSystemPrompt               # System prompt handling
├── TrafficInterception            # Request/response inspection
└── FullWorkflow                   # Complete workflow simulation
```

### expand_behavior_test.go - Expand Context Tests (9 tests)
Tests WITH and WITHOUT `expand_context` enabled (uses Haiku for cost efficiency).

```
TestExpandBehavior
├── WITH expand_context enabled
│   ├── WithExpand_MinimalCompression    # Force LLM to use expand tool
│   ├── WithExpand_QuestionRequiresDetail # Question requiring full content
│   └── WithExpand_ErrorToolResult        # Error handling with expand
├── WITHOUT expand_context enabled
│   ├── NoExpand_CompressedOnly          # LLM only sees compressed
│   └── NoExpand_LargeOutput             # Large content without expand
├── Small content (no compression)
│   └── SmallContent_NoCompression       # Below threshold, no compression
├── Comparison tests
│   └── Compare_DetailedQuestionWithVsWithout # Side-by-side comparison
├── Multi-tool tests
│   └── MultiTool_SomeNeedExpand         # 3 tools, partial expansion
└── Stress tests
    └── StressTest_ManyToolResults       # 5 tool results at once
```

### hard_integration_test.go - Edge Cases (13 tests)
Complex scenarios and edge cases.

```
TestHardIntegration
├── Multiple tools
│   ├── ThreeToolsOneLarge_OneExpandNeeded # Mixed sizes
│   └── ThreeToolsAllLarge                 # All large outputs
├── Error handling
│   ├── ToolResultIsError                  # is_error: true
│   ├── MixedSuccessAndError               # Success + error mix
│   └── LargeErrorMessage                  # Large error content
├── Conversation flow
│   ├── MultiRoundToolUse                  # Multiple tool rounds
│   └── ToolUseInMiddleOfConversation      # Mid-conversation tools
├── Real-world outputs
│   ├── RealWorld_GitLog                   # git log output
│   ├── RealWorld_NPMInstall               # npm install output
│   └── RealWorld_DockerBuild              # docker build output
└── Edge cases
    ├── EmptyToolResult                    # Empty content handling
    ├── SpecialCharactersInOutput          # Unicode, escapes
    └── BinaryLikeContent                  # Binary-like data
```

## Running Tests

```bash
# Run all Anthropic integration tests
go test ./tests/anthropic/integration/... -v

# Run specific test file
go test ./tests/anthropic/integration/... -v -run "TestE2E"
go test ./tests/anthropic/integration/... -v -run "TestExpandBehavior"
go test ./tests/anthropic/integration/... -v -run "TestHardIntegration"

# Run single test
go test ./tests/anthropic/integration/... -v -run "TestExpandBehavior_WithExpand_MinimalCompression"

# Run with timeout (recommended for LLM tests)
go test ./tests/anthropic/integration/... -v -timeout 300s
```

## Models Used

| Test File | Model | Reason |
|-----------|-------|--------|
| e2e_test.go | claude-sonnet-4-20250514 | Quality responses for E2E |
| expand_behavior_test.go | claude-3-haiku-20240307 | Cost-effective for behavior tests |
| hard_integration_test.go | claude-sonnet-4-20250514 | Complex scenarios need better model |

## Test Categories

### Compression Tests
- Small content (no compression)
- Large content (compression triggered)
- Multiple tool results (mixed compression)

### expand_context Tests
- WITH expand enabled (LLM can request full content)
- WITHOUT expand enabled (LLM only sees compressed)
- Comparison between modes

### Error Handling Tests
- Tool result errors (is_error: true)
- Permission denied errors
- Mixed success/error results

### Real-World Scenarios
- Git output (log, diff)
- Package manager output (npm)
- Container output (docker build)
- Search results (grep, find)
