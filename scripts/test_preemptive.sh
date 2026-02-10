#!/bin/bash
# =============================================================================
# test_preemptive.sh - Test preemptive summarization with low thresholds
# =============================================================================
# 
# This script tests the preemptive summarization feature by:
# 1. Sending multiple messages to build up context
# 2. Checking that preemptive summarization triggers
# 3. Sending a compaction request and verifying X-Was-Precomputed header
#
# USAGE:
#   ./scripts/test_preemptive.sh
#
# PREREQUISITES:
#   - Gateway running with configs/preemptive_test.yaml
#   - ANTHROPIC_API_KEY set in environment
# =============================================================================

set -e

GATEWAY_URL="${GATEWAY_URL:-http://localhost:18080}"
API_KEY="${ANTHROPIC_API_KEY:-}"

if [ -z "$API_KEY" ]; then
    echo "ERROR: ANTHROPIC_API_KEY not set"
    exit 1
fi

echo "=========================================="
echo "Testing Preemptive Summarization"
echo "=========================================="
echo "Gateway: $GATEWAY_URL"
echo ""

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# -----------------------------------------------------------------------------
# Step 1: Send first message (establish session)
# -----------------------------------------------------------------------------
echo -e "${YELLOW}Step 1: Sending first message (establish session)${NC}"

RESPONSE1=$(curl -s -w "\n%{http_code}" -X POST "$GATEWAY_URL/v1/messages" \
  -H "Content-Type: application/json" \
  -H "x-api-key: $API_KEY" \
  -H "anthropic-version: 2023-06-01" \
  -d '{
    "model": "claude-haiku-4-5",
    "max_tokens": 100,
    "messages": [
      {"role": "user", "content": "Hello! I am testing preemptive summarization. This is the first message to establish a session."}
    ]
  }')

HTTP_CODE1=$(echo "$RESPONSE1" | tail -n 1)
BODY1=$(echo "$RESPONSE1" | head -n -1)

if [ "$HTTP_CODE1" -eq 200 ]; then
    echo -e "${GREEN}✓ First message sent successfully${NC}"
else
    echo -e "${RED}✗ First message failed: $HTTP_CODE1${NC}"
    echo "$BODY1"
    exit 1
fi

echo ""
sleep 1

# -----------------------------------------------------------------------------
# Step 2: Send multiple messages to build up context (~800 tokens)
# This should trigger preemptive summarization at 5% of 10K = 500 tokens
# -----------------------------------------------------------------------------
echo -e "${YELLOW}Step 2: Building up context to trigger preemptive summarization${NC}"

# Generate a longer conversation with ~800 tokens total
LONG_CONTENT="Let me tell you about a complex software project. We are building a context compression proxy that sits between AI coding assistants and LLM APIs. The main features include:

1. Tool Output Compression: When tools like file readers or search return large outputs, we compress them using semantic reranking to keep only the most relevant parts.

2. Preemptive Summarization: This is what we are testing now. When the conversation reaches 80% of the context window, we start generating a summary in the background. When the user eventually needs compaction, the summary is already ready.

3. Session Management: We track conversations by hashing the first few messages to create a stable session ID. This allows us to maintain state across requests.

4. Multiple Detection Methods: We can detect compaction requests through prompt patterns, specific headers, or tool calls that indicate the client wants to compress the conversation.

The architecture uses a Go backend with clean separation between the gateway handler, compression pipes, and the preemptive summarization manager. We also have comprehensive tests covering unit tests, integration tests, and end-to-end scenarios."

RESPONSE2=$(curl -s -w "\n%{http_code}" -X POST "$GATEWAY_URL/v1/messages" \
  -H "Content-Type: application/json" \
  -H "x-api-key: $API_KEY" \
  -H "anthropic-version: 2023-06-01" \
  -d "{
    \"model\": \"claude-haiku-4-5\",
    \"max_tokens\": 100,
    \"messages\": [
      {\"role\": \"user\", \"content\": \"Hello! I am testing preemptive summarization.\"},
      {\"role\": \"assistant\", \"content\": \"Hello! I understand you are testing preemptive summarization. How can I help?\"},
      {\"role\": \"user\", \"content\": \"$LONG_CONTENT\"}
    ]
  }")

HTTP_CODE2=$(echo "$RESPONSE2" | tail -n 1)
BODY2=$(echo "$RESPONSE2" | head -n -1)

if [ "$HTTP_CODE2" -eq 200 ]; then
    echo -e "${GREEN}✓ Long message sent successfully${NC}"
    echo "  (This should have triggered preemptive summarization in the background)"
else
    echo -e "${RED}✗ Long message failed: $HTTP_CODE2${NC}"
    echo "$BODY2"
    exit 1
fi

echo ""
echo "Waiting 3 seconds for background summarization to complete..."
sleep 3

# -----------------------------------------------------------------------------
# Step 3: Send compaction request
# -----------------------------------------------------------------------------
echo -e "${YELLOW}Step 3: Sending compaction request${NC}"

# Include response headers in output
RESPONSE3=$(curl -s -w "\n%{http_code}\nHEADERS:\n" -D - -X POST "$GATEWAY_URL/v1/messages" \
  -H "Content-Type: application/json" \
  -H "x-api-key: $API_KEY" \
  -H "anthropic-version: 2023-06-01" \
  -H "X-Request-Compaction: true" \
  -d "{
    \"model\": \"claude-haiku-4-5\",
    \"max_tokens\": 100,
    \"messages\": [
      {\"role\": \"user\", \"content\": \"Hello! I am testing preemptive summarization.\"},
      {\"role\": \"assistant\", \"content\": \"Hello! I understand you are testing preemptive summarization. How can I help?\"},
      {\"role\": \"user\", \"content\": \"$LONG_CONTENT\"},
      {\"role\": \"assistant\", \"content\": \"That is a comprehensive project. The preemptive summarization feature sounds very useful.\"},
      {\"role\": \"user\", \"content\": \"Please summarize this conversation for me.\"}
    ]
  }")

echo "Response Headers:"
echo "$RESPONSE3" | grep -E "^X-" || echo "(No X- headers found)"
echo ""

# Check for the precomputed header
if echo "$RESPONSE3" | grep -q "X-Was-Precomputed: true"; then
    echo -e "${GREEN}✓ SUCCESS: Summary was precomputed!${NC}"
    echo "  The preemptive summarization worked - the summary was ready before you asked for it."
elif echo "$RESPONSE3" | grep -q "X-Was-Precomputed: false"; then
    echo -e "${YELLOW}⚠ Summary was computed synchronously${NC}"
    echo "  The preemptive summarization didn't complete in time, fell back to sync."
else
    echo -e "${RED}✗ Could not determine if summary was precomputed${NC}"
    echo "  Check the gateway logs for details."
fi

echo ""
echo "=========================================="
echo "Test complete. Check gateway logs for detailed flow."
echo "=========================================="
