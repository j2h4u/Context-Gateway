# Context Gateway

LLM proxy that compresses large tool outputs (Bash, Grep, Glob) before they reach the main model. Saves tokens and money. Read, Edit, Write outputs are skipped — they need exact byte matching.

## Links

- **Upstream**: [Compresr-ai/Context-Gateway](https://github.com/Compresr-ai/Context-Gateway)
- **Fork**: [j2h4u/Context-Gateway](https://github.com/j2h4u/Context-Gateway)
- **Port**: 18080
- **Image**: `ghcr.io/compresr-ai/context-gateway:latest`

## How it works

Claude Code sends requests to `localhost:18080` instead of `api.anthropic.com`. The gateway transparently compresses large tool outputs using Haiku and proxies everything else as-is. No API key needed — it reuses the OAuth token from incoming Claude Code requests.

## Updating

```bash
cd /opt/docker/context-gateway
docker compose pull && docker compose up -d
```

## Telemetry

```bash
cd /opt/docker/context-gateway
./telemetry-report.py
```

Shows: dollar savings, ROI, per-model/tool breakdown, threshold analysis with sweet spot. Model prices are fetched live from LiteLLM on each run.

## Key settings (config.yaml)

- `min_bytes` — minimum size for compression. Sweet spot is determined by the telemetry script (wasted=0)
- `skip_tools` — tools excluded from compression: `["read", "edit", "write"]`
- `provider` — which LLM to use for compression (currently `anthropic` / Haiku)

## Toggle on/off

```bash
/opt/docker/context-gateway/claude-toggle-gateway.py
```

Toggles `ANTHROPIC_BASE_URL` in `~/.claude/settings.json`. Run once — gateway on, run again — off. Takes effect on next Claude Code start.

## Troubleshooting

```bash
docker compose ps                  # status
docker compose logs --tail=50      # logs
curl http://localhost:18080/health # healthcheck
```

If the gateway breaks — remove `ANTHROPIC_BASE_URL` from `~/.claude/settings.json`, Claude Code will connect directly to Anthropic.
