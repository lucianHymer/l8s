# SSH Certificates Lost on Container Rebuild

**Date**: 2025-09-05

## Issue

When containers are rebuilt using `l8s rebuild`, the SSH certificates and sshd configuration are lost.

## Root Cause

1. RebuildContainer removes and recreates the container
2. It calls client.CreateContainer (the low-level Podman operation) which doesn't include certificate setup
3. The certificate setup (setupSSHCertificatesBeforeStart) is only called in Manager.CreateContainer, not in RebuildContainer

## Solution

Call setupSSHCertificatesBeforeStart in RebuildContainer after creating the new container but before starting it, just like CreateContainer does.

## Related Files
- `pkg/container/manager.go` - Contains both CreateContainer and RebuildContainer methods