# SSH Commands and PATH Dependency Issues

**Date**: 2025-10-29

## Issue

When executing commands via SSH non-interactively (e.g., `ssh host "command"`), the remote shell doesn't source `.zshrc` or set up PATH properly. This causes "command not found" errors even if the command exists in `~/.local/bin`.

## Root Cause

Non-interactive SSH sessions don't run login shells by default:
- `.zshrc` is not sourced
- PATH doesn't include `~/.local/bin` or other custom directories
- Environment variables are minimal

## Solution

Use fully qualified paths instead of relying on PATH:
- **Wrong**: `ssh host "team list"`
- **Right**: `ssh host "~/.local/bin/team list"`

The `~` is expanded by SSH automatically, so this works reliably.

## What Doesn't Work

Avoid wrapping commands in `zsh -l -c`:
- **Unnecessary**: Fully qualified paths work without it
- **Complicated**: Adds extra quoting complexity
- **Fragile**: Can break with complex arguments

## Implementation Pattern

In L8s command handlers, use fully qualified paths for scripts:
```go
cmd := []string{"~/.local/bin/team", "list"}
```

## Related Files
- `pkg/cli/handlers.go` - Command handlers using SSH execution
