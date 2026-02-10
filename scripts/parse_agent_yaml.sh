#!/bin/bash
# YAML Parser for Agent Configuration Files
# ==========================================
# Extracts values from agent YAML files with environment variable expansion.
#
# Usage:
#   ./parse_agent_yaml.sh <yaml_file> <key_path>
#
# Examples:
#   ./parse_agent_yaml.sh agents/claude_code.yaml "agent.name"
#   ./parse_agent_yaml.sh agents/claude_code.yaml "agent.gateway.config"

set -e

if [ "$#" -lt 2 ]; then
    echo "Usage: $0 <yaml_file> <key_path>" >&2
    echo "" >&2
    echo "Examples:" >&2
    echo '  $0 agents/claude.yaml "agent.name"' >&2
    echo '  $0 agents/claude.yaml "agent.gateway.config"' >&2
    exit 1
fi

YAML_FILE="$1"
KEY_PATH="$2"

if [ ! -f "$YAML_FILE" ]; then
    echo "Error: File not found: $YAML_FILE" >&2
    exit 1
fi

# Parse nested YAML keys
# This is a simplified parser that works for our specific YAML structure
parse_yaml_key() {
    local file="$1"
    local keypath="$2"

    # Split key path by dots
    IFS='.' read -ra KEYS <<< "$keypath"

    local current_indent=0
    local found_section=""
    local result=""

    # For simple two-level keys like "agent.name"
    if [ "${#KEYS[@]}" -eq 2 ]; then
        local section="${KEYS[0]}"
        local key="${KEYS[1]}"

        result=$(awk -v section="$section:" -v key="$key:" '
            $1 == section { in_section=1; next }
            in_section && $1 == key {
                # Extract value after colon, remove quotes
                val=$0
                sub(/^[^:]*:[[:space:]]*/, "", val)
                gsub(/^["'\'']|["'\'']$/, "", val)
                print val
                exit
            }
            in_section && /^[a-zA-Z]/ && $1 != key { exit }
        ' "$file")

    # For three-level keys like "agent.gateway.config"
    elif [ "${#KEYS[@]}" -eq 3 ]; then
        local section="${KEYS[0]}"
        local subsection="${KEYS[1]}"
        local key="${KEYS[2]}"

        result=$(awk -v section="$section:" -v subsection="$subsection:" -v key="$key:" '
            $1 == section { in_section=1; next }
            in_section && $1 == subsection { in_subsection=1; next }
            in_subsection && $1 == key {
                # Extract value after colon, remove quotes
                val=$0
                sub(/^[^:]*:[[:space:]]*/, "", val)
                gsub(/^["'\'']|["'\'']$/, "", val)
                print val
                exit
            }
            in_subsection && /^  [a-zA-Z]/ && $1 != key { in_subsection=0 }
            in_section && /^[a-zA-Z]/ && $1 != section { exit }
        ' "$file")

    # For four-level keys like "agent.command.run"
    elif [ "${#KEYS[@]}" -eq 4 ]; then
        local section="${KEYS[0]}"
        local subsection="${KEYS[1]}"
        local subsubsection="${KEYS[2]}"
        local key="${KEYS[3]}"

        result=$(awk -v section="$section:" -v subsection="$subsection:" -v subsubsection="$subsubsection:" -v key="$key:" '
            $1 == section { in_section=1; next }
            in_section && $1 == subsection { in_subsection=1; next }
            in_subsection && $1 == subsubsection { in_subsubsection=1; next }
            in_subsubsection && $1 == key {
                # Extract value after colon, remove quotes
                val=$0
                sub(/^[^:]*:[[:space:]]*/, "", val)
                gsub(/^["'\'']|["'\'']$/, "", val)
                print val
                exit
            }
        ' "$file")
    fi

    echo "$result"
}

# Get the value
VALUE=$(parse_yaml_key "$YAML_FILE" "$KEY_PATH")

# Expand environment variables (${VAR} syntax)
# Use eval to expand, but safely quote the value first
if [[ "$VALUE" == *'${'* ]]; then
    VALUE=$(echo "$VALUE" | envsubst)
fi

echo "$VALUE"
