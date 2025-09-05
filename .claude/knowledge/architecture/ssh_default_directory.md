# SSH Default Directory Configuration

L8s configures SSH connections to automatically start in the project workspace directory for better developer experience.

## Architecture Overview

### Initial Behavior
SSH connections initially logged users into the default home directory (/home/dev). The SSH connection was established using `exec.Command("ssh", containerName)` which ran a standard SSH command with no working directory configuration.

### Application-Level Implementation

L8s handles automatic navigation to `/workspace/project` at the application level, avoiding SSH config complications.

**Implementation Details**:
- The `SSHIntoContainer` method in `pkg/container/manager.go` handles directory navigation
- For interactive SSH sessions: Explicitly runs 'cd /workspace/project' before starting the shell
- For non-interactive operations: No directory change occurs, preserving compatibility
- SSH config remains clean without RemoteCommand directive

### Why Not RemoteCommand?

Initially attempted to use SSH RemoteCommand in the config:
- Would have added: `RemoteCommand cd /workspace/project && exec $SHELL -l`
- **Problem discovered**: RemoteCommand breaks git push operations with error "Cannot execute command-line and remote command"
- Git uses SSH for transport and needs to execute its own commands
- RemoteCommand conflicts with git's command execution needs

## Solution Architecture

The current implementation splits behavior based on session type:

1. **Interactive SSH (`l8s ssh`)**:
   - `SSHIntoContainer` method runs: `ssh -t <container> "cd /workspace/project && exec $SHELL -l"`
   - Users land directly in their project workspace
   - Full terminal capabilities preserved

2. **Non-interactive operations**:
   - Git push/pull operations work normally
   - SCP/SFTP transfers function correctly
   - SSH commands like `ssh dev-container ls` execute as expected
   - No directory change interference

## Impact Analysis

This application-level approach ensures compatibility:

1. **l8s exec**: Uses `Manager.ExecContainer` which calls `client.ExecContainer` - uses Podman exec directly, not affected.

2. **scp/file transfers**: Work normally as no RemoteCommand exists to interfere.

3. **VS Code Remote SSH**: May open in home directory initially, but users can configure VS Code's remote.SSH.defaultExtensions setting if needed.

4. **git operations**: Git push/pull to container remotes work perfectly as SSH transport is unmodified.

5. **Non-interactive SSH commands**: Commands like `ssh dev-myproject ls` work correctly without any directory change.

## Benefits

- Users land directly in their project workspace (for interactive sessions)
- Improved developer experience
- No impact on automated operations or git functionality
- Clean, maintainable solution without SSH config complications

## Related Files
- `pkg/container/manager.go` - SSHIntoContainer implementation with directory navigation
- `pkg/ssh/keys.go` - SSH config generation (without RemoteCommand)
- `pkg/ssh/keys_test.go` - Tests for SSH config generation
- `pkg/cli/handlers.go` - CLI command handlers