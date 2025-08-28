# Gotcha: Nonexistent list command flags

**Date**: 2025-08-28

## Issue

The ZSH completion previously assumed `l8s list` supported `--running` and `--stopped` flags, but these were never implemented in the actual CLI.

### What Happened
- ZSH completion tried to use `l8s list --running` and `l8s list --stopped`
- The runList handler in `pkg/cli/handlers.go` doesn't process any flags
- The list command just lists all containers regardless of flags

### Solution

Container filtering must happen in the completion function itself by grepping the output, not by passing flags to the CLI command.

The completion now:
1. Always calls `l8s list` without flags
2. Pipes output through grep to filter by state
3. Uses patterns like `grep -i running` or `grep -iE '(stopped|created|exited)'`

## Lessons Learned

- Always verify CLI flags actually exist before using them in completions
- Local filtering in completion functions can be more reliable than depending on CLI flags
- Test completions against actual CLI behavior, not assumed behavior

## Related Files
- `host-integration/oh-my-zsh/l8s/_l8s` - Fixed completion implementation
- `pkg/cli/handlers.go` - runList handler that doesn't support flags