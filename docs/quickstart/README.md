# Context Gateway — Quick Start

Transparent proxy that compresses large tool outputs (Bash, Grep, Glob) before they reach the main model, saving tokens and money. Read and Edit tool outputs are skipped to preserve exact byte matching.

## What you need

- Docker with compose plugin
- Claude Code subscription (Max or Pro) — gateway reuses your existing OAuth token, no API key needed

## Setup

One-liner:

```bash
curl -sL https://raw.githubusercontent.com/j2h4u/Context-Gateway/main/docs/quickstart/install.sh | sh
cd /opt/docker/context-gateway && docker compose up -d
```

Or manually:

```bash
# Create directory
mkdir -p /opt/docker/context-gateway && cd /opt/docker/context-gateway

# Download files (docker-compose.yml and config.yaml from this directory)
# Or copy them manually — see below

# Create logs directory
mkdir -p logs

# Start
docker compose up -d

# Verify
curl http://localhost:18080/health
```

## Point Claude Code at the gateway

Add to `~/.claude/settings.json` (global) or `.claude/settings.json` (per-project):

```json
{
  "env": {
    "ANTHROPIC_BASE_URL": "http://localhost:18080"
  }
}
```

Restart Claude Code. All requests now flow through the gateway.

## How it works

1. Claude Code sends requests to `localhost:18080` instead of Anthropic directly
2. Gateway inspects tool results in the conversation
3. Large outputs (>1KB) get compressed by Haiku before forwarding
4. If the model needs the original, it calls `expand_context` (injected automatically)
5. Read and Edit outputs are never compressed — they need exact bytes for file editing
6. If compression fails, the request passes through unchanged

## Files

- `docker-compose.yml` — pulls the official image, runs the gateway
- `config.yaml` — gateway settings (thresholds, skip_tools, provider)

## Updating

```bash
cd /opt/docker/context-gateway
docker compose pull
docker compose up -d
```

## Troubleshooting

```bash
# Check status
docker compose ps

# View logs
docker compose logs --tail=50

# Health check
curl http://localhost:18080/health
```

If something breaks, remove the `ANTHROPIC_BASE_URL` setting — Claude Code will connect directly to Anthropic as usual.
