#!/bin/bash

# Claude Code status line script for l8s containers
# Shows container name, team session, working directory, and output style

# ANSI color codes
RED='\033[0;91m'      # Bright red for container name
CYAN='\033[0;96m'     # Bright cyan for team session  
YELLOW='\033[0;93m'   # Bright yellow for path
PURPLE='\033[0;95m'   # Bright purple for output style
GRAY='\033[0;90m'     # Gray for separators
GREEN='\033[0;92m'    # Bright green for model name
RESET='\033[0m'       # Reset color

# Read JSON input from stdin
json_input=$(cat)

# Log the latest statusline input for debugging (silently, don't affect output)
echo "$json_input" > "$HOME/.debug-statusline.json" 2>/dev/null

# Extract working directory from JSON input using jq if available
# Falls back to PWD if jq not available or parsing fails
if command -v jq &> /dev/null 2>&1; then
    cwd=$(echo "$json_input" | jq -r '.cwd // empty' 2>/dev/null)
fi
if [ -z "$cwd" ]; then
    cwd="$PWD"
fi

# Extract model display name from JSON input using jq
model_display=""
if command -v jq &> /dev/null 2>&1; then
    model_display=$(echo "$json_input" | jq -r '.model.display_name // empty' 2>/dev/null)
fi

# Format model display with brackets if available
if [ -n "$model_display" ]; then
    model_display=" ${GREEN}[${model_display}]${RESET}"
fi

# Simplify path - handle both home and root directories properly
if [ "$cwd" = "$HOME" ]; then
    path_display="~"
elif [[ "$cwd" == "$HOME/"* ]]; then
    # Path is under home directory
    path_display="~/${cwd#$HOME/}"
else
    # Path is not under home (like /workspace)
    path_display="$cwd"
fi

# Truncate to last 2 directories if path is long
IFS='/' read -ra PARTS <<< "$path_display"
# Count actual parts (non-empty)
count=0
for part in "${PARTS[@]}"; do
    if [ -n "$part" ] || [ "$part" = "~" ]; then
        ((count++))
    fi
done
if [ $count -gt 3 ]; then
    # Get last 2 parts
    if [[ "$path_display" == "~/"* ]]; then
        path_display="~/.../$(basename "$(dirname "$cwd")")/$(basename "$cwd")"
    else
        path_display=".../$(basename "$(dirname "$cwd")")/$(basename "$cwd")"
    fi
fi

# Extract container name from hostname (remove 'dev-' prefix)
# Use HOSTNAME env var if available (more reliable in containers)
if [ -n "$HOSTNAME" ]; then
    container_name=$(echo "$HOSTNAME" | sed 's/^dev-//')
elif command -v hostname &> /dev/null; then
    container_name=$(hostname | sed 's/^dev-//')
else
    container_name="container"
fi

# Check if in a team session (dtach)
team_session=""
if [ -n "$DTACH_SESSION" ]; then
    team_session="${CYAN}⚒${DTACH_SESSION}${RESET}"
fi

# Get output style name from Claude settings if available
output_style=""
if [ -f "$HOME/.claude/output_styles/current" ]; then
    output_style=$(cat "$HOME/.claude/output_styles/current" 2>/dev/null)
fi
if [ -n "$output_style" ]; then
    output_style=" ${PURPLE}[${output_style}]${RESET}"
fi

# Output the status line with colors matching oh-my-posh
# Format: 🤖 [Model] container[⚒team]:path [style]
# Using printf with -e to interpret escape sequences
printf "🤖${model_display} ${RED}${container_name}${RESET}${team_session}${GRAY}:${RESET}${YELLOW}${path_display}${RESET}${output_style}\n"