### [12:40] [gotcha] SSH certificates lost on container rebuild
**Details**: When containers are rebuilt using `l8s rebuild`, the SSH certificates and sshd configuration are lost because:

1. RebuildContainer removes and recreates the container
2. It calls client.CreateContainer (the low-level Podman operation) which doesn't include certificate setup
3. The certificate setup (setupSSHCertificatesBeforeStart) is only called in Manager.CreateContainer, not in RebuildContainer

The fix is to call setupSSHCertificatesBeforeStart in RebuildContainer after creating the new container but before starting it, just like CreateContainer does.
**Files**: pkg/container/manager.go
---

