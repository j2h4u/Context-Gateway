<p align="center">
  <img src="https://compresr.ai/logo.png" alt="Compresr" width="200"/>
</p>

<p align="center">
  <b>Instant history compaction and context optimization for AI agents</b>
</p>


<p align="center">
  <a href="https://compresr.ai">Website</a> •
  <a href="https://compresr.ai/docs">Docs</a> •
  <a href="https://discord.gg/PeaVWNjT">Discord</a>
</p>

---

# Context Gateway

**[Compresr](https://compresr.ai)** is a YC-backed company building LLM prompt compression and context optimization.

Context Gateway sits between your AI agent (Claude Code, Cursor, etc.) and the LLM API. When your conversation gets too long, it compresses history **in the background** so you never wait for compaction.

## Quick Start

```bash
# 1. Install (from GitHub)
curl -fsSL https://compresr.ai/install_gateway_cli | sh

# 2. Add API key
echo 'ANTHROPIC_API_KEY=sk-ant-...' >> ~/.config/context-gateway/.env

# 3. Run (auto-detects config)
context-gateway
```

Then point your agent to the gateway:

```bash
export ANTHROPIC_API_URL=http://localhost:18080
```

## What you'll notice

- **No more waiting** when conversation hits context limits
- Compaction happens instantly (summary was pre-computed in background)
- Check `logs/compaction.jsonl` to see what's happening

## Contributing

We welcome contributions! Please join our [Discord](https://discord.gg/PeaVWNjT) to contribute.
