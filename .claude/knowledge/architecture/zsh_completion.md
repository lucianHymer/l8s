# ZSH Completion System Architecture

The L8s ZSH completion system provides sophisticated tab completion for all L8s commands and container names.

## Location and Structure

**Main file**: `host-integration/oh-my-zsh/l8s/_l8s`

The completion system uses ZSH's built-in completion framework with custom functions for L8s-specific behavior.

## Key Architecture Components

### Container Filtering
Container filtering happens locally in the completion function by parsing `l8s list` output, not through CLI flags:

1. The `_l8s_get_containers()` function filters container states:
   - **running**: `grep -i running`
   - **stopped**: `grep -iE '(stopped|created|exited)'`
   - **all**: no filtering applied

2. Container names have the 'dev-' prefix stripped for better UX during completion

### Flag Detection
Flag completion uses pattern detection:
- `[[ "$PREFIX" == -* ]]` to detect when user is typing a flag
- Each command has its own flag set defined in a case statement
- Flags are completed contextually based on the current command

### Context-Aware Filtering
The completion provides intelligent filtering based on command context:
- **start**: only stopped containers (makes sense to start only stopped ones)
- **stop/exec/paste**: only running containers (can only act on running containers)
- **remove/rm/rebuild/info/ssh**: all containers (can operate on any state)

## Implementation Details

The completion function:
1. Detects the current command being completed
2. Determines if completing a flag or container name
3. Applies appropriate filtering based on command context
4. Returns filtered results to ZSH completion system

## Integration

The completion integrates with Oh-My-Zsh plugin system:
- Located in standard Oh-My-Zsh completion directory
- Auto-loaded when l8s plugin is enabled
- Works with standard ZSH completion keybindings

## Related Files
- `host-integration/oh-my-zsh/l8s/_l8s` - Main completion file
- `host-integration/oh-my-zsh/l8s/tests/` - Test suite for completions