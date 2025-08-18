#!/bin/bash

# PreToolUse hook - logs all tool use attempts before execution
# This hook is called before each tool use in Claude

LOG_FILE="$HOME/.claude/logs/pretooluse.log"
mkdir -p "$(dirname "$LOG_FILE")"

# Add timestamp and separator
echo "==================== PRETOOLUSE HOOK ====================" >> "$LOG_FILE"
echo "Timestamp: $(date -u +"%Y-%m-%d %H:%M:%S UTC")" >> "$LOG_FILE"
echo "Arguments: $@" >> "$LOG_FILE"

# Append all inputs
echo "--- Hook Input Start ---" >> "$LOG_FILE"
cat >> "$LOG_FILE"
echo -e "\n--- Hook Input End ---\n" >> "$LOG_FILE"