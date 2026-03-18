#!/usr/bin/env python3
"""Fix corrupted Claude Code sessions with empty thinking blocks.

Root cause: gateway SSE conversion bug stripped both thinking content AND
signature from thinking blocks. Claude Code normally handles empty thinking
(converts to redacted_thinking using the signature), but blocks with BOTH
fields empty are unfixable and cause API 400 errors.

Fix: remove thinking blocks where BOTH thinking and signature are empty.
These blocks contain no recoverable data and are the sole cause of the
"each thinking block must contain thinking" error.

Usage:
  python3 fix-thinking-blocks.py                    # dry-run: show affected files
  python3 fix-thinking-blocks.py --fix               # fix all affected files
  python3 fix-thinking-blocks.py --fix <file.jsonl>  # fix specific file
"""

import json
import os
import sys
from pathlib import Path


def fix_line(line: str) -> tuple[str, int]:
    """Fix a single JSONL line. Returns (fixed_line, removed_count)."""
    stripped = line.strip()
    if not stripped:
        return line, 0

    try:
        data = json.loads(stripped)
    except json.JSONDecodeError:
        return line, 0

    if data.get("type") != "assistant":
        return line, 0

    message = data.get("message")
    if not isinstance(message, dict):
        return line, 0

    content = message.get("content")
    if not isinstance(content, list):
        return line, 0

    # Remove thinking blocks where BOTH thinking AND signature are empty.
    # Blocks with empty thinking but valid signature are NORMAL —
    # Claude Code converts those to redacted_thinking.
    original_len = len(content)
    filtered = [
        block for block in content
        if not (
            isinstance(block, dict)
            and block.get("type") == "thinking"
            and block.get("thinking") == ""
            and not block.get("signature")  # empty or missing signature
        )
    ]

    removed = original_len - len(filtered)
    if removed == 0:
        return line, 0

    message["content"] = filtered
    ending = "\n" if line.endswith("\n") else ""
    return json.dumps(data, ensure_ascii=False, separators=(",", ":")) + ending, removed


def fix_file(filepath: str, dry_run: bool = True) -> tuple[int, int]:
    """Fix a single JSONL file. Returns (lines_fixed, blocks_removed)."""
    with open(filepath, "r") as f:
        lines = f.readlines()

    new_lines = []
    total_lines_fixed = 0
    total_blocks_removed = 0

    for line in lines:
        fixed, removed = fix_line(line)
        new_lines.append(fixed)
        if removed > 0:
            total_lines_fixed += 1
            total_blocks_removed += removed

    if total_blocks_removed > 0 and not dry_run:
        backup = filepath + ".bak"
        if not os.path.exists(backup):
            os.rename(filepath, backup)
        with open(filepath, "w") as f:
            f.writelines(new_lines)

    return total_lines_fixed, total_blocks_removed


def find_affected_files(base_dir: str) -> list[str]:
    """Find JSONL files with truly corrupt thinking blocks (empty thinking + empty signature)."""
    affected = []
    for root, _dirs, files in os.walk(base_dir):
        for name in files:
            if not name.endswith(".jsonl") or name == "history.jsonl":
                continue
            filepath = os.path.join(root, name)
            try:
                with open(filepath, "r") as f:
                    for line in f:
                        if '"thinking":""' not in line and '"thinking": ""' not in line:
                            continue
                        data = json.loads(line.strip())
                        if data.get("type") != "assistant":
                            continue
                        content = data.get("message", {}).get("content", [])
                        if not isinstance(content, list):
                            continue
                        for block in content:
                            if (
                                isinstance(block, dict)
                                and block.get("type") == "thinking"
                                and block.get("thinking") == ""
                                and not block.get("signature")
                            ):
                                affected.append(filepath)
                                raise StopIteration
            except StopIteration:
                pass
            except (OSError, UnicodeDecodeError, json.JSONDecodeError):
                pass
    return affected


def main():
    base_dir = os.path.expanduser("~/.claude")
    do_fix = "--fix" in sys.argv
    specific_file = None

    for arg in sys.argv[1:]:
        if arg != "--fix" and arg.endswith(".jsonl"):
            specific_file = arg

    if specific_file:
        files = [specific_file]
    else:
        files = find_affected_files(base_dir)

    if not files:
        print("No affected files found.")
        return

    print(f"{'FIXING' if do_fix else 'DRY RUN'}: {len(files)} affected file(s)\n")

    total_files = 0
    total_lines = 0
    total_blocks = 0

    for filepath in sorted(files):
        lines_fixed, blocks_removed = fix_file(filepath, dry_run=not do_fix)
        if blocks_removed > 0:
            total_files += 1
            total_lines += lines_fixed
            total_blocks += blocks_removed
            rel = os.path.relpath(filepath, base_dir)
            print(f"  {rel}: {lines_fixed} lines, {blocks_removed} corrupt thinking blocks")

    print(f"\nTotal: {total_files} files, {total_lines} lines, {total_blocks} corrupt blocks removed")

    if not do_fix and total_blocks > 0:
        print("\nRun with --fix to apply changes (backups created as .bak)")


if __name__ == "__main__":
    main()
