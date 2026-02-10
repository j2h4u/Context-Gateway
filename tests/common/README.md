# Common Test Utilities

This folder contains shared test utilities, fixtures, and provider-agnostic tests used across Anthropic and OpenAI test suites.

## Structure

```
tests/common/
├── provider_identification_test.go  # Provider detection tests (6 tests)
├── fixtures/
│   └── fixtures.go                   # Shared test data and config helpers
├── integration/                      # (empty - placeholder for shared integration tests)
└── unit/                             # (empty - placeholder for shared unit tests)
```

## Test Files

### provider_identification_test.go (6 tests)

Tests for the adapter registry and provider identification logic.

| Test Name | Description |
|-----------|-------------|
| `TestProviderIdentification_ExplicitHeader` | Detection via X-Provider header (anthropic, openai, gemini, unknown) |
| `TestProviderIdentification_PathBased` | Detection via URL path patterns (/v1/messages, /v1/chat/completions) |
| `TestProviderIdentification_APIHeaders` | Detection via API-specific headers (x-api-key, anthropic-version) |
| `TestProviderIdentification_Precedence` | X-Provider header takes precedence over path/headers |
| `TestAdapterRegistry_BuiltInAdapters` | Verifies both adapters are registered |
| `TestAdapterRegistry_GetByName` | Retrieves specific adapters by name |

## Fixtures (fixtures/fixtures.go)

### Test Data Constants

| Constant | Description |
|----------|-------------|
| `LargeToolOutput` | Large content that triggers compression (error logs, stack traces) |
| `SmallToolOutput` | Small JSON content below compression threshold |

### Store Helpers

| Function | Description |
|----------|-------------|
| `TestStore()` | Creates in-memory store for testing |
| `PreloadedStore(content)` | Creates store with pre-populated shadow content |

### Configuration Helpers

| Function | Description |
|----------|-------------|
| `SimpleCompressionConfig()` | Simple compression strategy with expand_context enabled |
| `APICompressionConfig()` | API-based compression with Compresr service |
| `PassthroughConfig()` | No compression, direct proxy |
| `ExpandDisabledConfig()` | Compression without expand_context feature |

## Running Tests

```bash
# Run all common tests
go test ./tests/common/... -v

# Run provider identification tests
go test ./tests/common/... -v -run TestProviderIdentification

# Run adapter registry tests
go test ./tests/common/... -v -run TestAdapterRegistry
```

## Usage

### Importing Fixtures

```go
import (
    "github.com/compresr/context-gateway/tests/common/fixtures"
)

func TestSomething(t *testing.T) {
    // Use shared test data
    content := fixtures.LargeToolOutput
    
    // Create test store
    store := fixtures.TestStore()
    
    // Get configuration
    cfg := fixtures.SimpleCompressionConfig()
}
```

### Provider Detection Logic

The gateway identifies providers through:

1. **X-Provider header** (highest priority)
   - `X-Provider: anthropic` → Anthropic adapter
   - `X-Provider: openai` → OpenAI adapter

2. **URL path patterns**
   - `/v1/messages` → Anthropic
   - `/v1/chat/completions` → OpenAI

3. **API-specific headers**
   - `x-api-key` + `anthropic-version` → Anthropic
   - Default fallback → OpenAI

## Notes

- Fixtures initialize with `.env` file loading via godotenv
- Logs are silenced during tests (zerolog disabled)
- All configs use in-memory store with 1-hour TTL
- The `integration/` and `unit/` folders are placeholders for future shared tests
