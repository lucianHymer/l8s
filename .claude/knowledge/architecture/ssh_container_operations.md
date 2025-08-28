# SSH and Container Operations

L8s performs all operations via SSH to remote containers, never locally.

## SSH Connection Architecture

SSH host configs are used for container access:
- **Format**: `ssh containerPrefix-name` (e.g., "dev-myproject")
- SSH configs added automatically during container creation
- `Manager.SSHIntoContainer()` executes 'ssh <container-name>' with stdio forwarding

## Remote Execution Patterns

Two patterns for executing commands in containers:

### Direct SSH
- Used for interactive commands
- Method: `SSHIntoContainer`
- Full terminal forwarding

### Podman Exec
- Used for automated commands
- Methods: `ExecContainer`, `ExecContainerWithInput`
- Programmatic execution with captured output

## File Transfer

Currently uses exec commands for file operations:
- `ExecContainerWithInput` for writing files with stdin
- `mkdir`, `chmod`, `chown` commands for permission management
- No direct scp/rsync usage in current codebase

## Container Communication

All communication via SSH tunnel to remote Podman socket:
- PodmanClient connects to remote socket over SSH
- Container operations happen on remote server, never locally
- Ensures true remote-only operation

## Related Files
- `pkg/container/manager.go` - Container management logic
- `pkg/container/podman_client.go` - Podman client implementation
- `pkg/cli/handlers.go` - Command handlers using these operations