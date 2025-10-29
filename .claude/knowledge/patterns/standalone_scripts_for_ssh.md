# Standalone Scripts vs Shell Functions for SSH Execution

When commands need to be called via SSH from the host, they must be standalone executable scripts rather than shell functions.

## The Problem

Shell functions defined in `.zshrc` only work in interactive shells:
- SSH non-interactive sessions don't source `.zshrc`
- Functions aren't available for remote execution
- Commands fail with "command not found"

## The Solution

Create standalone executable scripts in `~/.local/bin/`:

1. **Create script file**: `pkg/embed/dotfiles/.local/bin/scriptname`
2. **Make it executable**: Add to executableFiles map in `manager.go`
3. **Deploy via embedded dotfiles**: The system handles deployment automatically

## Example: Team Command

**Before** (doesn't work via SSH):
```zsh
# In .zshrc
team() {
  # function implementation
}
```

**After** (works via SSH):
```bash
# In .local/bin/team (executable script)
#!/usr/bin/env zsh
# script implementation
```

## Deployment Process

The embedded dotfiles system:
1. Reads scripts from `pkg/embed/dotfiles/.local/bin/`
2. Deploys them to container's `~/.local/bin/`
3. Sets executable permissions based on executableFiles map
4. Scripts are available for both interactive and SSH execution

## Benefits

- **SSH compatible**: Works with `ssh host "~/.local/bin/script"`
- **Consistent**: Same behavior interactive vs non-interactive
- **Simple**: No need for login shell wrappers
- **Testable**: Scripts can be tested independently

## Related Files
- `pkg/embed/dotfiles/.local/bin/` - Script location
- `pkg/container/manager.go` - executableFiles map for permissions
- `pkg/embed/dotfiles/.zshrc` - Can still source scripts if needed
