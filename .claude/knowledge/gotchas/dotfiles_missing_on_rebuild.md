# Dotfiles Not Redeployed on Container Rebuild

**Date**: 2025-10-29

## Issue

Container rebuild doesn't automatically redeploy dotfiles, causing new files like the team script to be missing after rebuild. The RebuildContainer function only recreated the container without refreshing the dotfiles from the embedded filesystem.

## Root Cause

The RebuildContainer function was missing a call to `copyDotfiles()` after recreating and starting the container. This meant that any new or updated dotfiles added to `pkg/embed/dotfiles/` since the original container creation wouldn't be present after a rebuild.

## Symptoms

The issue manifests when:
1. New dotfiles are added to `pkg/embed/dotfiles/` (like `.local/bin/team`)
2. Container is rebuilt with `l8s rebuild`
3. New files are missing because rebuild didn't refresh dotfiles

## Solution

Added `m.copyDotfiles(ctx, containerName)` call in RebuildContainer after the container is started (Step 9). This ensures any new or updated dotfiles get deployed during rebuild operations, matching the behavior of initial container creation.

## Impact

Without this fix, users would need to manually recreate containers (rather than rebuild) to get updated dotfiles, which defeats the purpose of the rebuild command.

## Related Files
- `pkg/container/manager.go` - RebuildContainer implementation with copyDotfiles fix