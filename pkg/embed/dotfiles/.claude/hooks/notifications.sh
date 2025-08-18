#!/bin/bash

# Default notifications hook - appends all hook inputs to a log file
# This hook is called for various notification events in Claude

LOG_FILE="$HOME/.claude/logs/notifications.log"
mkdir -p "$(dirname "$LOG_FILE")"

# Add timestamp and separator
echo "==================== NOTIFICATIONS HOOK ====================" >> "$LOG_FILE"
echo "Timestamp: $(date -u +"%Y-%m-%d %H:%M:%S UTC")" >> "$LOG_FILE"
echo "Arguments: $@" >> "$LOG_FILE"

# Append all inputs
echo "--- Hook Input Start ---" >> "$LOG_FILE"
cat >> "$LOG_FILE"
echo -e "\n--- Hook Input End ---\n" >> "$LOG_FILE"