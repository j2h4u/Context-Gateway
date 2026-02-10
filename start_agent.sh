#!/bin/bash
# Start Agent with Gateway Proxy
#
# USAGE:
#   ./start_agent.sh                    # Interactive menu (recommended)
#   ./start_agent.sh [AGENT]            # Select agent, then config menu
#   ./start_agent.sh [AGENT] [OPTIONS]  # Direct mode with specific config
#
# EXAMPLES:
#   ./start_agent.sh                                # Interactive mode
#   ./start_agent.sh claude_code                    # Interactive config selection
#   ./start_agent.sh claude_code -c test_openai_direct.yaml
#   ./start_agent.sh cursor -c test.yaml -d
#   ./start_agent.sh -l                             # List agents
#
# FLAGS:
#   -c, --config FILE    Gateway config (optional - shows menu if not specified)
#   -p, --port PORT      Gateway port override (default: 18080)
#   -d, --debug          Enable debug logging
#   --proxy MODE         auto (default), start, skip
#   --log-dir DIR        Override log directory (default: ./logs)
#   -l, --list           List available agents
#   -h, --help           Show help

set -e

# =============================================================================
# CONFIGURATION
# =============================================================================

SCRIPT_PATH="${BASH_SOURCE[0]}"
while [ -L "$SCRIPT_PATH" ]; do
    SCRIPT_DIR="$(cd "$(dirname "$SCRIPT_PATH")" && pwd -P)"
    SCRIPT_PATH="$(readlink "$SCRIPT_PATH")"
    [[ "$SCRIPT_PATH" != /* ]] && SCRIPT_PATH="$SCRIPT_DIR/$SCRIPT_PATH"
done
SCRIPT_DIR="$(cd "$(dirname "$SCRIPT_PATH")" && pwd -P)"

AGENTS_DIR="$SCRIPT_DIR/agents"
CONFIGS_DIR="$SCRIPT_DIR/configs"
SCRIPTS_DIR="$SCRIPT_DIR/scripts"
ENV_FILE="$SCRIPT_DIR/.env"
LOGS_DIR="${LOG_DIR:-$SCRIPT_DIR/logs}"
PID_FILE="$LOGS_DIR/gateway.pid"

source "$SCRIPTS_DIR/utils.sh"

# Defaults
DEBUG_MODE=false
PROXY_MODE="auto"
GATEWAY_CONFIG=""
GATEWAY_PORT="18080"
AGENT_ARG=""
WE_STARTED_GATEWAY=false

# =============================================================================
# HELPER FUNCTIONS
# =============================================================================

show_help() {
    cat << EOF
Start Agent with Gateway Proxy

Usage: $0 [AGENT] [OPTIONS]
       $0                    # Interactive menu for both agent and config
       $0 AGENT              # Select agent, then choose config interactively
       $0 AGENT -c FILE      # Specify both agent and config

Options:
  -c, --config FILE    Gateway config (optional - shows menu if not specified)
  -p, --port PORT      Gateway port (default: 18080)
  -d, --debug          Enable debug logging
  --proxy MODE         auto (default), start, skip
  --log-dir DIR        Log directory (default: ./logs)
  -l, --list           List available agents
  -h, --help           Show this help

Examples:
  $0                                        # Interactive mode
  $0 claude_code                            # Interactive config selection
  $0 claude_code -c test_openai_direct.yaml
  $0 cursor -c test.yaml -d

EOF
}

get_config_metadata() {
    local config_file="$1"
    local field="$2"
    grep -A 10 "^metadata:" "$config_file" 2>/dev/null | \
        grep "^  ${field}:" | \
        sed 's/^  [^:]*: *"\(.*\)"/\1/' | \
        sed 's/^  [^:]*: *\(.*\)/\1/'
}

# Generic numbered-list selector.
# Usage: select_from_list "prompt" item1 item2 ... itemN
# Sets SELECTED_INDEX (0-based) and SELECTED_ITEM.
select_from_list() {
    local prompt="$1"; shift
    local items=("$@")
    local total=${#items[@]}

    echo ""
    echo -e "${BOLD}${CYAN}${prompt}${NC}"
    echo ""
    for i in "${!items[@]}"; do
        echo -e "  ${GREEN}[$((i + 1))]${NC} ${items[$i]}"
    done
    echo -e "  ${YELLOW}[0]${NC} Cancel"
    echo ""

    while true; do
        read -rp "Enter number: " choice
        if [[ "$choice" == "0" ]]; then
            print_warn "Cancelled"
            exit 0
        fi
        if [[ "$choice" =~ ^[0-9]+$ ]] && [ "$choice" -ge 1 ] && [ "$choice" -le "$total" ]; then
            SELECTED_INDEX=$((choice - 1))
            SELECTED_ITEM="${items[$SELECTED_INDEX]}"
            return 0
        fi
        echo "Invalid choice. Enter 1-${total} or 0 to cancel."
    done
}

start_gateway_background() {
    local config="$1" port="$2" debug="$3"

    local args=("-c" "$config")
    [ -n "$port" ] && args+=("-p" "$port")
    [ "$debug" = "true" ] && args+=("-d")

    print_info "Starting gateway in background..."
    SESSION_TELEMETRY_LOG="$SESSION_TELEMETRY_LOG" \
    SESSION_COMPRESSION_LOG="$SESSION_COMPRESSION_LOG" \
    SESSION_COMPACTION_LOG="$SESSION_COMPACTION_LOG" \
    SESSION_TRAJECTORY_LOG="$SESSION_TRAJECTORY_LOG" \
    SESSION_GATEWAY_LOG="$SESSION_GATEWAY_LOG" \
    SESSION_GATEWAY_PID="$SESSION_GATEWAY_PID" \
    "$SCRIPTS_DIR/start_gateway.sh" "${args[@]}" > /dev/null 2>&1 &
}

cleanup_on_exit() {
    echo ""
    print_warn "Shutting down..."

    # Only stop gateway if this script started it
    if [ "$WE_STARTED_GATEWAY" = "true" ]; then
        stop_gateway "$PID_FILE"
    fi

    echo ""
    if [ -n "$SESSION_DIR" ]; then
        echo -e "${CYAN}Session logs:${NC} $SESSION_DIR"
    fi
    echo ""
    exit 0
}

# =============================================================================
# ARGUMENT PARSING
# =============================================================================

while [[ $# -gt 0 ]]; do
    case "$1" in
        -h|--help) show_help; exit 0 ;;
        -l|--list) list_agents "$AGENTS_DIR" "$SCRIPTS_DIR"; exit 0 ;;
        -d|--debug) DEBUG_MODE=true; shift ;;
        -c|--config) GATEWAY_CONFIG="$2"; shift 2 ;;
        --proxy) PROXY_MODE="$2"; shift 2 ;;
        -p|--port) GATEWAY_PORT="$2"; shift 2 ;;
        --log-dir) LOGS_DIR="$2"; shift 2 ;;
        -*) print_error "Unknown option: $1"; show_help; exit 1 ;;
        *) AGENT_ARG="$1"; shift ;;
    esac
done

# =============================================================================
# MAIN EXECUTION
# =============================================================================

print_banner

# Load environment
[ -f "$ENV_FILE" ] && load_env_file "$ENV_FILE"

# Create logs directory
ensure_directory "$LOGS_DIR"

# Check if gateway is already running
GATEWAY_ALREADY_RUNNING=false
if [ "$PROXY_MODE" != "skip" ] && check_gateway_running "$GATEWAY_PORT"; then
    GATEWAY_ALREADY_RUNNING=true
    print_success "Reusing existing gateway on port $GATEWAY_PORT"
fi

# Only create session directory if starting a NEW gateway
if [ "$GATEWAY_ALREADY_RUNNING" = "false" ]; then
    SESSION_DIR=$(create_session_directory "$LOGS_DIR")
    [[ "$SESSION_DIR" != /* ]] && SESSION_DIR="$(cd "$SESSION_DIR" && pwd -P)"
    print_success "Session: $(basename "$SESSION_DIR")"

    export SESSION_TELEMETRY_LOG="$SESSION_DIR/telemetry.jsonl"
    export SESSION_COMPRESSION_LOG="$SESSION_DIR/compression.jsonl"
    export SESSION_COMPACTION_LOG="$SESSION_DIR/compaction.jsonl"
    export SESSION_TRAJECTORY_LOG="$SESSION_DIR/trajectory.json"
    export SESSION_GATEWAY_LOG="$SESSION_DIR/gateway.log"
    export SESSION_GATEWAY_PID="$SESSION_DIR/gateway.pid"

    PID_FILE="$SESSION_DIR/gateway.pid"
fi

# Interactive agent menu if none specified
if [ -z "$AGENT_ARG" ]; then
    agents=()
    for yaml_file in "$AGENTS_DIR"/*.yaml; do
        [ -f "$yaml_file" ] || continue
        name=$(basename "$yaml_file" .yaml)
        [[ "$name" == template* ]] && continue
        agents+=("$name")
    done
    [ ${#agents[@]} -eq 0 ] && { print_error "No agents found"; exit 1; }

    select_from_list "Select an agent:" "${agents[@]}"
    AGENT_ARG="$SELECTED_ITEM"
fi

# Validate agent
AGENT_YAML="$AGENTS_DIR/${AGENT_ARG}.yaml"
if [ ! -f "$AGENT_YAML" ]; then
    print_error "Agent not found: $AGENT_ARG"
    echo ""
    list_agents "$AGENTS_DIR" "$SCRIPTS_DIR"
    exit 1
fi

validate_agent "$AGENT_YAML" "$SCRIPTS_DIR"
VALIDATE_RESULT=$?
if [ $VALIDATE_RESULT -eq 2 ]; then
    exec "$0"
    exit 0
elif [ $VALIDATE_RESULT -ne 0 ]; then
    exit 1
fi

# Get agent configuration
AGENT_CMD=$(parse_yaml "$AGENT_YAML" "agent.command.run" "$SCRIPTS_DIR")
AGENT_DISPLAY=$(parse_yaml "$AGENT_YAML" "agent.display_name" "$SCRIPTS_DIR")

if [ -z "$AGENT_CMD" ]; then
    print_error "Failed to parse agent command from $AGENT_YAML"
    exit 1
fi

# Interactive config menu if needed and not specified
if [ "$PROXY_MODE" != "skip" ] && [ -z "$GATEWAY_CONFIG" ]; then
    configs=() cfg_labels=()
    for f in "$CONFIGS_DIR"/*.yaml; do
        [ -f "$f" ] || continue
        local_name=$(get_config_metadata "$f" "name")
        local_desc=$(get_config_metadata "$f" "description")
        configs+=("$(basename "$f")")
        cfg_labels+=("${local_name:-$(basename "$f")} - ${local_desc}")
    done
    if [ ${#configs[@]} -eq 0 ]; then
        print_error "No configuration files found in $CONFIGS_DIR"
        exit 1
    fi

    select_from_list "Select a gateway configuration:" "${cfg_labels[@]}"
    GATEWAY_CONFIG="${configs[$SELECTED_INDEX]}"
    print_success "Selected: $GATEWAY_CONFIG"
fi

export GATEWAY_PORT

# =============================================================================
# STEP 1: Start Gateway
# =============================================================================

echo ""
print_header "Step 1: Gateway Setup"

if [ "$PROXY_MODE" != "skip" ]; then
    if [ "$GATEWAY_ALREADY_RUNNING" = "true" ]; then
        print_success "Gateway running on port $GATEWAY_PORT (reusing existing)"
    else
        print_step "Starting gateway in background..."
        start_gateway_background "$GATEWAY_CONFIG" "$GATEWAY_PORT" "$DEBUG_MODE"
        sleep 3

        if ! wait_for_gateway "$GATEWAY_PORT"; then
            print_error "Gateway failed to start"
            [ -n "$SESSION_DIR" ] && echo "Check logs: tail -f $SESSION_DIR/gateway.log"
            echo -e "${CYAN}Continue anyway? [y/N]${NC}"
            read -rsn1 response
            case "$response" in
                y|Y) print_warn "Continuing without healthy gateway..." ;;
                *)   print_error "Exiting due to gateway failure"; exit 1 ;;
            esac
        else
            print_success "Gateway started on port $GATEWAY_PORT"
            WE_STARTED_GATEWAY=true
        fi
    fi
else
    print_info "Skipping gateway (--proxy skip)"
fi

# Export environment variables
export_agent_env "$AGENT_YAML"

# =============================================================================
# STEP 2: OpenClaw Model Selection (only for OpenClaw)
# =============================================================================

if [ "$AGENT_ARG" = "openclaw" ]; then
    echo ""
    print_header "Step 2: OpenClaw Model Selection"
    select_model_interactive "$AGENT_YAML" "$SCRIPTS_DIR"
    create_openclaw_config "$SELECTED_MODEL" "$GATEWAY_PORT"
    start_openclaw_gateway
fi

# Setup cleanup
trap cleanup_on_exit SIGINT SIGTERM EXIT

# =============================================================================
# STEP 3: Start Agent
# =============================================================================

echo ""
print_header "Step 3: Start Agent"

print_step "Launching ${AGENT_DISPLAY:-$AGENT_ARG}..."
echo ""
[ -n "$SESSION_DIR" ] && echo -e "${CYAN}Session logs: $(basename "$SESSION_DIR")${NC}"
echo ""

# Clean up stale IDE lock files before launching
rm -f ~/.claude/ide/*.lock 2>/dev/null || true

# Run agent as child process so cleanup trap fires on exit
$AGENT_CMD
AGENT_EXIT_CODE=$?

echo ""
print_info "Agent exited with code: $AGENT_EXIT_CODE"
exit $AGENT_EXIT_CODE
