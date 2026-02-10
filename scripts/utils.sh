#!/bin/bash
# Common Shell Functions Library
# ===============================
# Shared functions for Context Gateway scripts

# =============================================================================
# COLORS
# =============================================================================

export GREEN='\033[0;32m'
export BLUE='\033[0;34m'
export CYAN='\033[0;36m'
export YELLOW='\033[1;33m'
export RED='\033[0;31m'
export BOLD='\033[1m'
export DIM='\033[2m'
export NC='\033[0m'

# =============================================================================
# PRINT FUNCTIONS
# =============================================================================

print_header() {
    local title="${1:-Context Gateway}"
    echo ""
    echo -e "${BOLD}${CYAN}========================================${NC}"
    echo -e "${BOLD}${CYAN}       $title${NC}"
    echo -e "${BOLD}${CYAN}========================================${NC}"
    echo ""
}

print_success() {
    echo -e "${GREEN}[OK]${NC} $1"
}

print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

print_step() {
    echo -e "${CYAN}>>>${NC} $1"
}

print_banner() {
    local compresr_green='\033[38;2;23;128;68m'  # #178044
    echo -e "${compresr_green}${BOLD}"
    cat << 'EOF'

  ██████╗ ██████╗ ███╗  ██╗████████╗███████╗██╗ ██╗████████╗  ██████╗  █████╗ ████████╗███████╗██╗    ██╗ █████╗ ██╗   ██╗
 ██╔════╝██╔═══██╗████╗ ██║╚══██╔══╝██╔════╝╚██╗██╔╝╚══██╔══╝ ██╔════╝ ██╔══██╗╚══██╔══╝██╔════╝██║    ██║██╔══██╗╚██╗ ██╔╝
 ██║     ██║   ██║██╔██╗██║   ██║   █████╗   ╚███╔╝    ██║    ██║  ███╗███████║   ██║   █████╗  ██║ █╗ ██║███████║ ╚████╔╝ 
 ██║     ██║   ██║██║╚████║   ██║   ██╔══╝   ██╔██╗    ██║    ██║   ██║██╔══██║   ██║   ██╔══╝  ██║███╗██║██╔══██║  ╚██╔╝  
 ╚██████╗╚██████╔╝██║ ╚███║   ██║   ███████╗██╔╝ ██╗   ██║    ╚██████╔╝██║  ██║   ██║   ███████╗╚███╔███╔╝██║  ██║   ██║   
  ╚═════╝ ╚═════╝ ╚═╝  ╚══╝   ╚═╝   ╚══════╝╚═╝  ╚═╝   ╚═╝     ╚═════╝ ╚═╝  ╚═╝   ╚═╝   ╚══════╝ ╚══╝╚══╝ ╚═╝  ╚═╝   ╚═╝   

EOF
    echo -e "${NC}"
}

# =============================================================================
# ENVIRONMENT FUNCTIONS
# =============================================================================

load_env_file() {
    local env_file="$1"

    if [ -f "$env_file" ]; then
        set -a
        source "$env_file" 2>/dev/null || true
        set +a
        print_success "Loaded .env file"
        return 0
    else
        print_warn ".env file not found at $env_file"
        return 1
    fi
}

# =============================================================================
# DIRECTORY FUNCTIONS
# =============================================================================

ensure_directory() {
    local dir="$1"
    mkdir -p "$dir" 2>/dev/null || true
}

# =============================================================================
# SESSION MANAGEMENT
# =============================================================================

create_session_directory() {
    local logs_base="$1"
    local date=$(date +%Y%m%d_%H%M%S)

    # Find next session number
    local session_num=1
    if [ -d "$logs_base" ]; then
        # Use find to avoid glob errors when no sessions exist
        local last_session=$(find "$logs_base" -maxdepth 1 -type d -name "session_*" 2>/dev/null | sort -V | tail -1)
        if [ -n "$last_session" ]; then
            local last_num=$(basename "$last_session" | sed 's/session_\([0-9]*\)_.*/\1/')
            session_num=$((last_num + 1))
        fi
    fi

    local session_dir="$logs_base/session_${session_num}_${date}"
    mkdir -p "$session_dir"
    echo "$session_dir"
}

get_session_log_path() {
    local session_dir="$1"
    local log_type="$2"  # telemetry or compression
    echo "$session_dir/${log_type}.jsonl"
}

# =============================================================================
# GATEWAY FUNCTIONS
# =============================================================================

check_gateway_running() {
    local port="${1:-${GATEWAY_PORT:-18080}}"
    curl -sf "http://localhost:$port/health" > /dev/null 2>&1
}

wait_for_gateway() {
    local port="${1:-${GATEWAY_PORT:-18080}}"
    local max_wait="${2:-30}"
    local waited=0

    print_step "Waiting for gateway to be ready..."

    while ! check_gateway_running "$port"; do
        if [ $waited -ge $max_wait ]; then
            print_error "Gateway failed to start after ${max_wait}s"
            return 1
        fi
        sleep 1
        waited=$((waited + 1))
    done

    print_success "Gateway is healthy!"
    return 0
}

stop_gateway() {
    local pid_file="$1"

    if [ -f "$pid_file" ]; then
        local pid=$(cat "$pid_file")
        if ps -p "$pid" > /dev/null 2>&1; then
            print_info "Stopping gateway (PID: $pid)..."
            kill "$pid" 2>/dev/null || true
            rm -f "$pid_file"
            return 0
        fi
    fi
    return 1
}

# =============================================================================
# PROCESS FUNCTIONS
# =============================================================================

# Check if port is in use and by what process
# Returns: 0 if port is free, 1 if used by gateway (safe to kill), 2 if used by other process
check_port_usage() {
    local port="$1"
    local gateway_binary="${2:-gateway}"
    
    # Check if port is in use
    local pid=$(lsof -ti:"$port" 2>/dev/null | head -1)
    
    if [ -z "$pid" ]; then
        # Port is free
        return 0
    fi
    
    # Get the process command
    local proc_cmd=$(ps -p "$pid" -o comm= 2>/dev/null)
    local proc_full=$(ps -p "$pid" -o command= 2>/dev/null)
    
    # Check if it's our gateway process
    if [[ "$proc_cmd" == *"gateway"* ]] || [[ "$proc_full" == *"bin/gateway"* ]] || [[ "$proc_full" == *"$gateway_binary"* ]]; then
        # It's our gateway - safe to kill
        echo "$pid"
        return 1
    else
        # It's another process - NOT safe to kill
        echo "$pid:$proc_cmd"
        return 2
    fi
}

# Kill gateway on port if it's our process, error if it's not
kill_gateway_on_port() {
    local port="$1"
    local gateway_binary="${2:-gateway}"
    
    local result
    result=$(check_port_usage "$port" "$gateway_binary")
    local status=$?
    
    case $status in
        0)
            # Port is free
            return 0
            ;;
        1)
            # Our gateway is using the port - kill it
            local pid="$result"
            print_info "Stopping existing gateway (PID: $pid) on port $port..."
            kill "$pid" 2>/dev/null || true
            sleep 1
            # Verify it's stopped
            if lsof -ti:"$port" > /dev/null 2>&1; then
                kill -9 "$pid" 2>/dev/null || true
                sleep 1
            fi
            print_success "Previous gateway stopped"
            return 0
            ;;
        2)
            # Another process is using the port
            local pid="${result%%:*}"
            local proc="${result#*:}"
            print_error "Port $port is in use by another process!"
            print_error "  PID: $pid"
            print_error "  Process: $proc"
            print_error "Please stop that process or use a different port (-p flag)"
            return 1
            ;;
    esac
}

kill_process_by_path() {
    local binary_path="$1"

    if pgrep -f "$binary_path" > /dev/null 2>&1; then
        print_info "Stopping running processes..."
        pkill -f "$binary_path" 2>/dev/null || true
        sleep 1
        return 0
    fi
    return 1
}

# =============================================================================
# YAML PARSING
# =============================================================================

parse_yaml() {
    local yaml_file="$1"
    local key_path="$2"
    local scripts_dir="${3:-$(dirname "$(dirname "${BASH_SOURCE[0]}")")}"

    "$scripts_dir/parse_agent_yaml.sh" "$yaml_file" "$key_path" 2>/dev/null || echo ""
}

# =============================================================================
# VALIDATION FUNCTIONS
# =============================================================================

validate_file_exists() {
    local file="$1"
    local description="${2:-File}"

    if [ ! -f "$file" ]; then
        print_error "$description not found: $file"
        return 1
    fi
    return 0
}

validate_command() {
    local cmd="$1"
    local name="${2:-Command}"

    if ! command -v "$cmd" &> /dev/null; then
        print_error "$name is not installed. Please install $cmd first."
        return 1
    fi
    return 0
}

# =============================================================================
# BUILD FUNCTIONS
# =============================================================================

check_needs_build() {
    local binary_path="$1"
    local source_dir="$2"

    # Binary doesn't exist
    if [ ! -f "$binary_path" ]; then
        echo "true"
        return
    fi

    # Check if any .go file is newer than binary
    local newest_go=$(find "$source_dir" -name "*.go" -newer "$binary_path" 2>/dev/null | head -1)
    if [ -n "$newest_go" ]; then
        echo "true"
        return
    fi

    echo "false"
}

build_go_binary() {
    local project_dir="$1"
    local binary_path="$2"
    local main_file="${3:-./cmd/main.go}"

    print_step "Building binary..."

    if ! validate_command "go" "Go"; then
        return 1
    fi

    mkdir -p "$(dirname "$binary_path")"

    cd "$project_dir"
    if go build -o "$binary_path" "$main_file" 2>&1; then
        print_success "Build successful"
        return 0
    else
        print_error "Build failed"
        return 1
    fi
}

# =============================================================================
# CLEANUP HANDLERS
# =============================================================================

setup_cleanup_handler() {
    local cleanup_func="$1"
    trap "$cleanup_func" SIGINT SIGTERM EXIT
}

# =============================================================================
# AGENT FUNCTIONS
# =============================================================================

export_agent_env() {
    local yaml_file="$1"
    local in_env_section=false
    local current_var=""

    while IFS= read -r line; do
        if [[ "$line" =~ ^[[:space:]]*environment:[[:space:]]*$ ]]; then
            in_env_section=true
            continue
        fi

        if $in_env_section && [[ "$line" =~ ^[[:space:]]{0,2}[a-z_]+:[[:space:]]*$ ]] && [[ ! "$line" =~ ^[[:space:]]*- ]]; then
            break
        fi

        if $in_env_section; then
            if [[ "$line" =~ ^[[:space:]]*-[[:space:]]*name:[[:space:]]*\"(.+)\" ]]; then
                current_var="${BASH_REMATCH[1]}"
            elif [[ "$line" =~ ^[[:space:]]*value:[[:space:]]*\"(.+)\" ]]; then
                local value="${BASH_REMATCH[1]}"
                value=$(eval echo "$value")
                export "$current_var=$value"
                print_info "Exported: $current_var"
                current_var=""
            fi
        fi
    done < "$yaml_file"
}

validate_agent() {
    local yaml_file="$1"
    local scripts_dir="$2"

    if [ ! -f "$yaml_file" ]; then
        print_error "Agent configuration not found: $yaml_file"
        return 1
    fi

    local check_cmd=$(parse_yaml "$yaml_file" "agent.command.check" "$scripts_dir")
    local fallback_msg=$(parse_yaml "$yaml_file" "agent.command.fallback_message" "$scripts_dir")
    local install_cmd=$(parse_yaml "$yaml_file" "agent.command.install" "$scripts_dir")
    local agent_name=$(parse_yaml "$yaml_file" "agent.display_name" "$scripts_dir")
    [ -z "$agent_name" ] && agent_name=$(basename "$yaml_file" .yaml)

    if [ -n "$check_cmd" ]; then
        if ! eval "$check_cmd"; then
            echo ""
            print_warn "Agent '$agent_name' is not installed"
            [ -n "$fallback_msg" ] && echo -e "${YELLOW}  $fallback_msg${NC}"
            echo ""
            
            if [ -n "$install_cmd" ]; then
                echo -e "${CYAN}Would you like to install it now? [Y/n]${NC}"
                echo -e "${DIM}  Command: $install_cmd${NC}"
                echo ""
                read -rsn1 response
                
                case "$response" in
                    n|N)
                        echo ""
                        print_info "Installation skipped. Returning to menu..."
                        sleep 1
                        return 2  # Special code to indicate "go back to menu"
                        ;;
                    *)
                        echo ""
                        print_step "Installing $agent_name..."
                        echo ""
                        if eval "$install_cmd"; then
                            echo ""
                            print_success "$agent_name installed successfully!"
                            sleep 1
                            return 0
                        else
                            echo ""
                            print_error "Installation failed"
                            echo -e "${YELLOW}You can try installing manually: $install_cmd${NC}"
                            echo ""
                            echo -e "${CYAN}Press any key to return to menu...${NC}"
                            read -rsn1
                            return 2
                        fi
                        ;;
                esac
            else
                # No install command available
                echo -e "${YELLOW}No automatic installation available.${NC}"
                echo -e "${CYAN}Press any key to return to menu...${NC}"
                read -rsn1
                return 2
            fi
        fi
    fi
    return 0
}

discover_agents() {
    local agents_dir="$1"
    local agents=()

    for yaml_file in "$agents_dir"/*.yaml; do
        [ -f "$yaml_file" ] || continue
        local agent_name=$(basename "$yaml_file" .yaml)
        [ "$agent_name" != "template" ] && agents+=("$agent_name")
    done

    echo "${agents[@]}"
}

list_agents() {
    local agents_dir="$1"
    local scripts_dir="$2"

    print_header "Start Agent with Gateway"
    echo -e "${BOLD}${CYAN}Available Agents:${NC}"
    echo ""

    local i=1
    for yaml_file in "$agents_dir"/*.yaml; do
        [ -f "$yaml_file" ] || continue
        local agent_name=$(basename "$yaml_file" .yaml)
        [[ "$agent_name" == template* ]] && continue

        local display_name=$(parse_yaml "$yaml_file" "agent.display_name" "$scripts_dir")
        local description=$(parse_yaml "$yaml_file" "agent.description" "$scripts_dir")

        echo -e "  ${GREEN}[$i]${NC} ${BOLD}$agent_name${NC}"
        [ -n "$display_name" ] && echo -e "      ${CYAN}$display_name${NC}"
        [ -n "$description" ] && echo -e "      $description"
        echo ""
        i=$((i + 1))
    done
}

# =============================================================================
# OPENCLAW SETUP FUNCTIONS
# =============================================================================

# Parse models from agent YAML and return as array
# Usage: parse_agent_models yaml_file scripts_dir
parse_agent_models() {
    local yaml_file="$1"
    local scripts_dir="$2"
    
    # Extract model IDs from YAML using awk
    awk '
        /^  models:/ { in_models=1; next }
        in_models && /- id:/ { 
            gsub(/.*- id: *"/, "")
            gsub(/".*/, "")
            print
        }
        in_models && /^  [a-z]/ && !/- / { exit }
    ' "$yaml_file"
}

# Get default model from agent YAML
get_default_model() {
    local yaml_file="$1"
    local scripts_dir="$2"
    
    grep "^  default_model:" "$yaml_file" 2>/dev/null | \
        sed 's/^  default_model: *"\?\([^"]*\)"\?/\1/'
}

# Create OpenClaw config file with selected model and proxy settings
# Usage: create_openclaw_config model gateway_port
create_openclaw_config() {
    local model="$1"
    local gateway_port="$2"
    local config_dir="$HOME/.openclaw"
    local config_file="$config_dir/openclaw.json"
    
    mkdir -p "$config_dir"
    
    # OpenClaw 2026.x config format:
    # - agents.defaults.model.primary for the model selection
    # - models.providers for custom baseUrl (proxy routing) - requires models array
    cat > "$config_file" << EOF
{
  "agents": {
    "defaults": {
      "model": {
        "primary": "$model"
      }
    }
  },
  "models": {
    "providers": {
      "anthropic": {
        "baseUrl": "http://localhost:$gateway_port",
        "models": []
      },
      "openai": {
        "baseUrl": "http://localhost:$gateway_port/v1",
        "models": []
      }
    }
  }
}
EOF
    
    print_success "Created OpenClaw config with model: $model"
    print_info "API calls routed through Context Gateway on port $gateway_port"
}

# Create OpenClaw config file WITHOUT proxy (direct API calls)
# Usage: create_openclaw_config_direct model
create_openclaw_config_direct() {
    local model="$1"
    local config_dir="$HOME/.openclaw"
    local config_file="$config_dir/openclaw.json"
    
    mkdir -p "$config_dir"
    
    # OpenClaw 2026.x config format - minimal config with just the model
    cat > "$config_file" << EOF
{
  "agents": {
    "defaults": {
      "model": {
        "primary": "$model"
      }
    }
  }
}
EOF
    
    print_success "Created OpenClaw config with model: $model"
    print_info "API calls go directly to providers (no proxy)"
}

# Start OpenClaw gateway
# Usage: start_openclaw_gateway
start_openclaw_gateway() {
    # Stop any existing gateway
    openclaw gateway stop 2>/dev/null || true
    sleep 1
    
    # Start fresh gateway
    print_info "Starting OpenClaw gateway..."
    openclaw gateway --port 18789 --allow-unconfigured --token localdev --force &
    sleep 2
    
    print_success "OpenClaw gateway started on port 18789"
}

# Interactive model selection for OpenClaw
# Usage: select_model_interactive yaml_file scripts_dir
# Sets SELECTED_MODEL variable
select_model_interactive() {
    local yaml_file="$1"
    local scripts_dir="$2"
    
    # Get models from YAML - parse both id and name
    local model_ids=()
    local model_names=()
    
    # Parse model data using awk (returns id|name pairs)
    local awk_output
    awk_output=$(awk '
        /^  models:/ { in_models=1; next }
        in_models && /- id:/ { 
            gsub(/.*- id: *"/, "")
            gsub(/".*/, "")
            current_id=$0
        }
        in_models && /name:/ && !/display_name/ { 
            gsub(/.*name: *"/, "")
            gsub(/".*/, "")
            print current_id "|" $0
        }
        in_models && /^  [a-z]/ && !/- / { exit }
    ' "$yaml_file")
    
    # Read awk output line by line
    while IFS='|' read -r model_id model_name; do
        if [ -n "$model_id" ]; then
            model_ids+=("$model_id")
            model_names+=("$model_name")
        fi
    done <<< "$awk_output"
    
    # Get default
    local default_model=$(get_default_model "$yaml_file" "$scripts_dir")
    
    local total=${#model_ids[@]}
    if [ "$total" -eq 0 ]; then
        SELECTED_MODEL="$default_model"
        return 0
    fi
    
    # Find default index
    local default_idx=0
    local i=0
    while [ $i -lt $total ]; do
        if [ "${model_ids[$i]}" = "$default_model" ]; then
            default_idx=$i
            break
        fi
        i=$((i + 1))
    done
    
    local selected=$default_idx
    tput civis  # Hide cursor
    trap 'tput cnorm' RETURN
    
    while true; do
        clear
        print_header "Select LLM Model"
        echo -e "${BOLD}${CYAN}Choose which model OpenClaw should use:${NC}"
        echo ""
        
        i=0
        while [ $i -lt $total ]; do
            local model_id="${model_ids[$i]}"
            local model_name="${model_names[$i]}"
            local default_marker=""
            [ "$model_id" = "$default_model" ] && default_marker=" ${DIM}(default)${NC}"
            
            if [ "$i" -eq "$selected" ]; then
                echo -e "  ${BOLD}${GREEN}➤ $model_name${NC}$default_marker"
                echo -e "      ${CYAN}$model_id${NC}"
            else
                echo -e "    $model_name$default_marker"
            fi
            i=$((i + 1))
        done
        
        echo ""
        echo -e "${CYAN}Use ↑/↓ arrows, Enter to select${NC}"
        
        read -rsn1 key
        case "$key" in
            $'\x1b')
                read -rsn2 key
                case "$key" in
                    '[A') ((selected--)); [ "$selected" -lt 0 ] && selected=$((total - 1)) ;;
                    '[B') ((selected++)); [ "$selected" -ge "$total" ] && selected=0 ;;
                esac
                ;;
            '')
                tput cnorm
                echo ""
                SELECTED_MODEL="${model_ids[$selected]}"
                print_success "Selected: ${model_names[$selected]} ($SELECTED_MODEL)"
                echo ""
                return 0
                ;;
        esac
    done
}

# =============================================================================
# ARGUMENT PARSING HELPERS
# =============================================================================

parse_flag() {
    local flag="$1"
    local short="$2"
    local long="$3"

    [ "$flag" = "$short" ] || [ "$flag" = "$long" ]
}
