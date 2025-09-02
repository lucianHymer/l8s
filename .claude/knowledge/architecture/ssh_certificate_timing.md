# SSH Certificate Setup Timing Architecture

The SSH certificate configuration happens before container startup to ensure proper initialization and avoid runtime dependencies.

## Timing Strategy

### Setup Before Start
The `setupSSHCertificatesBeforeStart()` method configures SSH certificates while the container is stopped:

1. **Pre-startup configuration**:
   - Uses `podman cp` to copy files to stopped containers
   - No runtime dependencies on container state
   - Ensures sshd starts with proper configuration

2. **Files copied**:
   - Host keys (`/etc/ssh/ssh_host_*_key`)
   - Certificates (`/etc/ssh/ssh_host_*_key-cert.pub`)
   - SSHD configuration (`/etc/ssh/sshd_config.d/99-certificates.conf`)

## Benefits

### No Restart Required
- SSHD reads certificate configuration on first start
- No need for `pkill -HUP sshd` or service restart
- Eliminates race conditions during startup

### Reliability
- Files guaranteed to be in place before sshd starts
- No timing issues with container initialization
- Consistent behavior across container restarts

### Performance
- Single `podman cp` operation for all files
- No exec operations into running containers
- Faster container startup time

## Implementation Details

The setup occurs in this sequence:
1. Container created (but not started)
2. SSH certificates generated and signed
3. Files copied via `podman cp`
4. Container started with certificates already in place
5. SSHD automatically uses certificates on startup

## Error Handling

If certificate setup fails:
- Container creation continues (graceful degradation)
- Error logged but not fatal
- Allows debugging of certificate issues

## Related Files
- `pkg/container/manager.go` - Contains setupSSHCertificatesBeforeStart method
- `pkg/ssh/ca.go` - Certificate generation and signing