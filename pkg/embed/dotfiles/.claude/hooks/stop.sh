#!/bin/bash

# Stop hook - logs when Claude Code is stopped
# This hook is called when the user stops Claude Code

LOG_FILE="$HOME/.claude/logs/stop.log"
mkdir -p "$(dirname "$LOG_FILE")"

# Add timestamp and separator
echo "==================== STOP HOOK ====================" >> "$LOG_FILE"
echo "Timestamp: $(date -u +"%Y-%m-%d %H:%M:%S UTC")" >> "$LOG_FILE"
echo "Arguments: $@" >> "$LOG_FILE"

# Append all inputs
echo "--- Hook Input Start ---" >> "$LOG_FILE"
cat >> "$LOG_FILE"
echo -e "\n--- Hook Input End ---\n" >> "$LOG_FILE"