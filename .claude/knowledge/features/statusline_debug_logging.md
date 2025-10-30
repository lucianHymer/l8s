# Statusline Debug Logging

L8s containers include debug logging for the Claude Code statusline to help troubleshoot and understand the data flow.

## Overview

The statusline script silently logs the latest JSON input from Claude Code to a debug file for inspection purposes.

## Implementation Details

### Debug File Location
- **Path**: `~/.debug-statusline.json`
- **Update Mode**: Overwritten each time (not appended)
- **Creation**: Line 19 of statusline.sh script

### Data Captured
The debug file contains the complete JSON payload from Claude Code, including:
- Session ID
- Model information
- Cost metrics
- Workspace paths
- Current working directory
- Output style settings
- Any other data Claude Code provides

### Silent Operation
- Logging happens with `2>/dev/null` to suppress errors
- No impact on statusline output
- File size remains bounded (overwrite mode)

## Usage

To inspect what Claude Code is sending to the statusline:
```bash
cat ~/.debug-statusline.json | jq .
```

This is particularly useful for:
- Understanding Claude Code's data structure
- Debugging statusline display issues
- Developing new statusline features

## Related Files
- `pkg/embed/dotfiles/.claude/statusline.sh` - Statusline script with debug logging