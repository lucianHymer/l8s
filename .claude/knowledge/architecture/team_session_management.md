# Team Session Management with dtach

L8s provides persistent terminal sessions using dtach, enabling collaborative workflows and session survival across SSH disconnections.

## Overview

The team command creates persistent terminal sessions that:
- Survive SSH disconnections
- Can be shared across multiple connections
- Integrate with Claude Code statusline
- Use git-native container resolution

## Architecture

### Session Storage
- **Socket location**: `/tmp/dtach-<base64-name>.sock`
- **Session tracking**: `DTACH_SESSION` environment variable
- **Display**: ⚒ icon in Claude Code statusline

### Command Structure
```bash
l8s team <name>     # Join or create session
l8s team list       # Show active sessions
```

### Git-Native Integration
Both commands use git-native architecture:
- Container derived from current worktree
- No manual container name specification needed
- Consistent with L8s design philosophy

## Implementation Details

### Team Script
**Location**: `pkg/embed/dotfiles/.local/bin/team`

The script is:
- Deployed to `~/.local/bin/team` in containers
- Marked executable via executableFiles map
- Callable via SSH using fully qualified path

### L8s Integration
**Location**: `pkg/cli/handlers.go`

L8s commands:
1. Resolve container from git context
2. Execute team script via SSH with fully qualified path: `~/.local/bin/team`
3. Forward terminal I/O for interactive sessions

### Statusline Integration
**Location**: `pkg/embed/dotfiles/.claude/statusline.sh`

Displays active session:
- Reads `DTACH_SESSION` environment variable
- Shows ⚒ icon with session name
- Updates automatically when session changes

## Benefits

1. **Persistence**: Sessions survive network interruptions
2. **Collaboration**: Multiple users can attach to same session
3. **Visibility**: Active session shown in editor statusline
4. **Integration**: Seamless with L8s git-native workflow

## Usage Example

```bash
# In your project directory
l8s team backend      # Create/join "backend" session
# Session shown in Claude Code statusline: ⚒ backend

# Later, or from another terminal
l8s team backend      # Rejoin same session

# List sessions
l8s team list         # Shows all active dtach sessions
```

## Related Files
- `pkg/embed/dotfiles/.local/bin/team` - Team session script
- `pkg/cli/handlers.go` - L8s team command handlers
- `pkg/embed/dotfiles/.claude/statusline.sh` - Statusline integration
