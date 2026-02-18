#!/usr/bin/env python3
"""Toggle Context Gateway proxy for Claude Code.

Adds or removes ANTHROPIC_BASE_URL from ~/.claude/settings.json.
Run without args to toggle. The setting takes effect on next Claude Code start."""

import json
from pathlib import Path

SETTINGS = Path.home() / ".claude" / "settings.json"
BACKUP = SETTINGS.with_suffix(".json.bak")
KEY = "ANTHROPIC_BASE_URL"
VALUE = "http://localhost:18080"

settings = json.loads(SETTINGS.read_text())
BACKUP.write_text(SETTINGS.read_text())
env = settings.setdefault("env", {})

if KEY in env:
    del env[KEY]
    print(f"Gateway OFF — Claude Code will connect directly to Anthropic")
else:
    env[KEY] = VALUE
    print(f"Gateway ON  — Claude Code will use {VALUE}")

SETTINGS.write_text(json.dumps(settings, indent=2) + "\n")
