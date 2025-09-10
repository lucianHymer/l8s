# Claude Code @ Syntax in Slash Commands

**Date**: 2025-09-10

## Issue

The @ syntax in Claude Code slash commands cannot be used with runtime variables for dynamic file references.

## Root Cause

The @ syntax in Claude Code slash commands directly includes file contents at command definition time, not runtime. For example, @path/to/file.txt will read and include that file's contents when the command is parsed. This means we cannot use @$1 to dynamically reference a file path from an argument - the @ syntax expects a literal file path at command definition time, not runtime variable substitution.

## Solution

To read a file dynamically based on an argument, use bash commands with the ! prefix instead:
- **Wrong**: `@$1` (attempts to use @ with variable)
- **Right**: `!cat "$1"` (uses bash to read file dynamically)

## Impact

This affects any slash command that needs to read files based on user-provided arguments, such as the /req command for reading requirements files.

## Related Files
- `pkg/embed/dotfiles/.claude/commands/req.md` - Requirements command using bash instead of @ syntax