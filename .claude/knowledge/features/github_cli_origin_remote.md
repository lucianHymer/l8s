# GitHub CLI Origin Remote Integration

The GitHub CLI (gh) integration in L8s containers requires an 'origin' remote to automatically detect the GitHub repository for operations.

## Problem

The GitHub CLI requires an 'origin' remote to be configured in the git repository to automatically detect which GitHub repository to operate on. Without this, users must manually specify the repository with the `-R` flag for every gh command.

Initially, L8s containers started with an empty git repository (git init) without any remotes. When code was pushed from the host, only a local remote was added on the host side pointing to the container, but the container itself had no knowledge of the upstream GitHub repository.

## Solution

L8s now automatically replicates the host repository's 'origin' remote to containers during creation. This enables seamless GitHub CLI usage within containers.

### Implementation Details

After pushing code to the container during `l8s create`:
1. The system checks if the host repository has an 'origin' remote
2. If found, it's automatically added to the container's git repository
3. This enables gh commands like `gh pr create` to work without the `-R` flag

### Error Handling

The origin remote replication is non-fatal:
- If adding the origin remote fails, container creation still succeeds
- A warning is logged that gh CLI may require manual configuration
- Users can manually add the remote if automatic setup fails

## Benefits

- **Seamless GitHub integration**: gh commands work immediately after container creation
- **No manual configuration**: Users don't need to add remotes manually
- **Consistent with local development**: Container git setup mirrors the host repository
- **Non-blocking**: Failures don't prevent container creation

## Technical Components

The implementation leverages existing git utilities:
- `git.ListRemotes()` - Get remotes from the host repository
- `git.AddRemote()` - Add remotes to the container repository
- Integration point in `runCreate` handler after code push

## Related Files
- `pkg/cli/handlers.go` - Contains the origin remote replication logic in runCreate
- `pkg/git/remote.go` - Git remote management utilities
- `pkg/container/manager.go` - Container management operations