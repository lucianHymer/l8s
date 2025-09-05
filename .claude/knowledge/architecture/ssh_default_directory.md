# SSH Default Directory Configuration

L8s configures SSH connections to automatically start in the project workspace directory for better developer experience.

## Architecture Overview

### Initial Behavior
SSH connections initially logged users into the default home directory (/home/dev). The SSH connection was established using `exec.Command("ssh", containerName)` which ran a standard SSH command with no working directory configuration.

### RemoteCommand Implementation

The SSH config now includes a RemoteCommand that automatically changes to `/workspace/project` when establishing interactive connections.

**Implementation Details**:
- Modified `GenerateSSHConfigEntry` in `pkg/ssh/keys.go`
- Added: `RemoteCommand cd /workspace/project && exec $SHELL -l`
- Only affects interactive SSH sessions (l8s ssh or direct ssh commands)
- Non-interactive commands (e.g., `ssh dev-container ls`) ignore RemoteCommand by SSH design
- SCP and SFTP operations remain unaffected
- RequestTTY remains at default (auto) which handles both interactive and non-interactive cases correctly

## Impact Analysis

RemoteCommand does NOT interfere with other L8s operations:

1. **l8s exec**: Uses `Manager.ExecContainer` which calls `client.ExecContainer` - uses Podman exec directly, NOT SSH. RemoteCommand doesn't affect it.

2. **scp/file transfers**: SCP operations like `scp file.txt dev-myproject:` still work because RemoteCommand only runs for interactive SSH sessions (when RequestTTY is yes). SCP doesn't request a TTY, so RemoteCommand is skipped.

3. **VS Code Remote SSH**: Benefits from RemoteCommand - VS Code opens directly in /workspace/project.

4. **git operations**: Git push/pull to container remotes use SSH for transport but don't execute commands interactively, so RemoteCommand doesn't interfere.

5. **Non-interactive SSH commands**: Commands like `ssh dev-myproject ls` work correctly as RemoteCommand is ignored for non-interactive sessions.

## Benefits

- Users land directly in their project workspace
- Improved developer experience
- No impact on automated operations
- Seamless integration with VS Code and other SSH tools

## Related Files
- `pkg/ssh/keys.go` - SSH config generation with RemoteCommand
- `pkg/ssh/keys_test.go` - Tests for SSH config generation
- `pkg/container/manager.go` - Container management and SSH operations
- `pkg/cli/handlers.go` - CLI command handlers