#!/bin/bash

# SubagentStop hook - logs when a subagent is stopped
# This hook is called when a subagent (Task tool) completes

LOG_FILE="$HOME/.claude/logs/subagent-stop.log"
mkdir -p "$(dirname "$LOG_FILE")"

# Add timestamp and separator
echo "==================== SUBAGENT STOP HOOK ====================" >> "$LOG_FILE"
echo "Timestamp: $(date -u +"%Y-%m-%d %H:%M:%S UTC")" >> "$LOG_FILE"
echo "Arguments: $@" >> "$LOG_FILE"

# Append all inputs
echo "--- Hook Input Start ---" >> "$LOG_FILE"
cat >> "$LOG_FILE"
echo -e "\n--- Hook Input End ---\n" >> "$LOG_FILE"