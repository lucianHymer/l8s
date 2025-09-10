# Command Grouping Feature

L8s implements command grouping in the help output to organize commands by their usage context and requirements.

## Command Categories

The commands are organized into four groups for better discoverability:

### 1. Container Operations (Git Repository Required)
Commands that require being in a git repository:
- **create**: Create container from current repository
- **ssh**: SSH into container for current repository
- **rebuild**: Rebuild container for current repository
- **rm/remove**: Remove container for current repository
- **exec**: Execute command in container for current repository
- **push**: Push changes to container (planned)
- **pull**: Pull changes from container (planned)
- **status**: Show status for current repository (planned)

### 2. Container Management
Commands that work with specific containers by name:
- **start**: Start a stopped container
- **stop**: Stop a running container
- **info**: Show container information
- **paste**: Paste clipboard to container
- **remote add/remove**: Manage git remotes for containers

### 3. System Setup
Global setup and configuration commands:
- **init**: Initialize L8s configuration
- **install-zsh-plugin**: Install ZSH completion plugin
- **connection**: Manage SSH connection settings

### 4. Maintenance & Development
Utility commands for maintenance:
- **list**: List all containers
- **rebuild-all**: Rebuild all containers
- **build**: Build L8s binary

## Implementation Details

### Cobra Command Grouping
Cobra provides built-in support for command grouping:

1. **Define groups**: Use `rootCmd.AddGroup()` with ID and Title
2. **Assign commands**: Set `GroupID` field on each command
3. **Order matters**: Groups appear in the order they're defined
4. **Built-in commands**: Use `SetHelpCommandGroupId()` and `SetCompletionCommandGroupId()`

### Key Requirements
- All groups must be defined before assigning commands to them
- Every command should belong to a group for consistent help output
- Group IDs should be descriptive (e.g., "repo-ops", "container-mgmt")

## Benefits

1. **Clear context requirements**: Users immediately see which commands need git context
2. **Better discoverability**: Related commands are grouped together
3. **Improved UX**: Reduces cognitive load when searching for commands
4. **Explicit design**: Makes the git-native architecture visible in the interface

## Related Files
- `cmd/l8s/main.go` - Command registration and group setup
- `pkg/cli/factory_lazy.go` - Command implementations