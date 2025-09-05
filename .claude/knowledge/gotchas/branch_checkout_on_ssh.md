# Branch Not Checked Out on Container SSH

**Date**: 2025-09-05

## Issue

When SSHing into a container, the default branch is checked out rather than the branch that was pushed during container creation.

## Root Cause

The container starts with an empty git repo on 'main' branch. When code is pushed during `l8s create`, it creates the branch but doesn't automatically check it out.

## Solution

After pushing the branch in runCreate (line 82 in handlers.go), execute a git checkout command in the container:

```go
checkoutCmd := []string{"su", "-", f.Config.ContainerUser, "-c",
    fmt.Sprintf("cd /workspace/project && git checkout %s", branch)}
f.ContainerMgr.ExecContainer(ctx, name, checkoutCmd)
```

This ensures the container is on the correct branch immediately after creation.

## Alternative Approaches Considered

1. Store branch in container labels (podman label)
2. Write branch info to a file in container (/workspace/.git/L8S_BRANCH)
3. Auto-checkout on SSH entry
4. Track which branch was pushed and ensure it's checked out

The direct checkout after push is the simplest and most reliable solution.

## Related Files
- `pkg/cli/handlers.go` - Contains runCreate function
- `pkg/container/manager.go` - Container management logic