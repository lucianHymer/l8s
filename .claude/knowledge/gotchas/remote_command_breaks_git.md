# RemoteCommand Breaks Git Push Operations

**Date**: 2025-09-05

## Issue

When RemoteCommand is set in SSH config, git push operations fail with the error: "Cannot execute command-line and remote command". This prevents the initial code push during container creation.

## Root Cause

Git uses SSH for transport when pushing to remote repositories. When RemoteCommand is configured in the SSH config entry, it conflicts with git's need to execute its own commands over SSH. The SSH protocol doesn't allow both a RemoteCommand (from config) and a command-line command (from git) to be executed in the same session.

## Why It Matters

This issue was discovered when attempting to use RemoteCommand to automatically navigate users to `/workspace/project` on SSH login. While RemoteCommand would work for interactive SSH sessions, it completely breaks git operations that rely on SSH transport.

## Solution

Instead of using RemoteCommand in SSH config, L8s handles directory navigation at the application level:
- For interactive SSH sessions (`l8s ssh`): The `SSHIntoContainer` method explicitly runs 'cd /workspace/project' before starting the shell
- For non-interactive operations (git, scp, etc.): No directory change occurs, ensuring compatibility

This approach provides the desired user experience without breaking critical git functionality.

## Related Files
- `pkg/ssh/keys.go` - SSH config generation (RemoteCommand removed)
- `pkg/container/manager.go` - SSHIntoContainer implementation with directory change