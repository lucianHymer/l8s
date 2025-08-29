# SSH Certificate Authority Implementation Plan for L8s

## Executive Summary

Replace the current insecure SSH host key handling (disabled strict checking) with a proper SSH Certificate Authority (CA) system. This will provide cryptographic verification of container identities while maintaining the same user experience.

## Current State

L8s currently disables SSH host key verification entirely:
```bash
StrictHostKeyChecking no
UserKnownHostsFile /dev/null
```

This leaves connections vulnerable to MITM attacks. The system provides no verification that you're connecting to the legitimate container.

## Proposed Solution

Implement an SSH Certificate Authority that:
1. Generates a CA keypair during `l8s init`
2. Signs each container's SSH host key during creation
3. Configures SSH clients to trust the CA for all L8s containers
4. Provides automatic trust for all containers without security warnings

## Design Specifications

### CA Key Management

**Storage Location**: `~/.config/l8s/ca/`
- `ca_key` - CA private key (permissions: 0600)
- `ca_key.pub` - CA public key (permissions: 0644)

**Generation**: During `l8s init`, using ssh-keygen:
```bash
ssh-keygen -t ed25519 -f ~/.config/l8s/ca/ca_key -C "l8s-ca@$(hostname)" -N ""
```

### Certificate Configuration

**Validity Period**: 10 years (3650 days)
- Containers are ephemeral development environments
- Long validity avoids renewal complexity
- Can be regenerated anytime via container rebuild

**Certificate Signing**: During container creation:
```bash
# Generate host key for container (if not exists)
ssh-keygen -t ed25519 -f /tmp/ssh_host_ed25519_key -N ""

# Sign the host key with CA
ssh-keygen -s ~/.config/l8s/ca/ca_key \
  -I "dev-${container_name}" \
  -h \
  -V +3650d \
  -n "dev-${container_name},${remote_host}" \
  /tmp/ssh_host_ed25519_key.pub
```

### SSH Client Configuration

**Known Hosts File**: `~/.config/l8s/known_hosts`
```
@cert-authority *.l8s.local,${remote_host}:* ssh-ed25519 AAAAC3... l8s-ca@hostname
```

**SSH Config Entry** (updated):
```
Host dev-myproject
    HostName server.example.com
    Port 2200
    User dev
    StrictHostKeyChecking yes
    UserKnownHostsFile ~/.config/l8s/known_hosts
    ControlMaster auto
    ControlPath ~/.ssh/control-%r@%h:%p
    ControlPersist 10m
```

### Container SSHD Configuration

Each container's `/etc/ssh/sshd_config` must include:
```
HostKey /etc/ssh/ssh_host_ed25519_key
HostCertificate /etc/ssh/ssh_host_ed25519_key-cert.pub
```

## Implementation Steps

### Phase 1: CA Infrastructure (pkg/ssh/ca.go)

Create new CA package with functions:
```go
type CA struct {
    PrivateKeyPath string
    PublicKeyPath  string
}

func NewCA(configDir string) (*CA, error)
func (ca *CA) Generate() error
func (ca *CA) SignHostKey(hostKeyPath, containerName, remoteHost string) error
func (ca *CA) WriteKnownHostsEntry(knownHostsPath, remoteHost string) error
func (ca *CA) Exists() bool
```

### Phase 2: Init Command Updates (pkg/cli/handlers.go)

Modify `runInit` to:
1. Create CA directory with proper permissions
2. Generate CA keypair
3. Create l8s known_hosts file with CA entry
4. Store CA paths in config

### Phase 3: Container Creation Updates (pkg/container/manager.go)

Modify container creation to:
1. Generate container SSH host key
2. Sign with CA certificate
3. Copy both key and certificate to container
4. Configure container sshd to use certificate

### Phase 4: SSH Config Updates (pkg/ssh/keys.go)

Update `AddSSHConfig` to:
1. Set `StrictHostKeyChecking yes`
2. Point to `~/.config/l8s/known_hosts`
3. Remove `/dev/null` UserKnownHostsFile

### Phase 5: Build Process Updates (containers/Dockerfile)

Update container image to:
1. Configure sshd for certificate support
2. Set proper permissions on host key locations
3. Ensure sshd reloads config on container start

## File Changes Required

### New Files
- `pkg/ssh/ca.go` - CA management functionality
- `pkg/ssh/ca_test.go` - CA unit tests
- `~/.config/l8s/ca/` - CA key storage (runtime)
- `~/.config/l8s/known_hosts` - CA trust configuration (runtime)

### Modified Files
- `pkg/cli/handlers.go` - Update runInit
- `pkg/container/manager.go` - Add certificate signing to creation
- `pkg/ssh/keys.go` - Update SSH config generation
- `pkg/config/config.go` - Add CA path fields
- `containers/Dockerfile` - Update sshd configuration
- `pkg/embed/dotfiles/` - Include sshd config template

## Testing Requirements

### Unit Tests
- CA generation and key signing
- Known hosts entry generation
- Config file updates
- Error handling for missing/corrupt CA

### Integration Tests
- Full flow: init → create → connect
- Certificate validation
- Multiple containers with same CA
- CA regeneration scenarios

### Manual Testing Checklist
- [ ] Run `l8s init` - verify CA generation
- [ ] Create new container - verify certificate signing
- [ ] SSH to container - verify no warnings
- [ ] Verify strict checking is enabled
- [ ] Test with multiple containers
- [ ] Test CA deletion and regeneration

## Security Considerations

1. **CA Private Key**: Must be protected with 0600 permissions
2. **No Backup**: CA private key should not be backed up to cloud/git
3. **Rotation**: Document CA rotation procedure for compromised keys
4. **Scope**: CA only trusts L8s containers, not other SSH hosts

## Migration Guide for Existing Users

Since backwards compatibility is not required:

1. **Clean Removal of Old Containers**:
   ```bash
   l8s list  # Note all containers
   l8s remove <each-container>
   ```

2. **Update L8s**:
   ```bash
   git pull
   make clean
   make build
   sudo make install
   ```

3. **Reinitialize with CA**:
   ```bash
   rm -rf ~/.config/l8s
   l8s init  # Will generate new CA
   ```

4. **Rebuild Base Image**:
   ```bash
   l8s build
   ```

5. **Recreate Containers**:
   ```bash
   l8s create <name> <git-url> <branch>
   ```

6. **Clean SSH Config** (optional):
   Remove old entries from `~/.ssh/config` for dev-* hosts

## Success Criteria

- [ ] No SSH warnings when connecting to containers
- [ ] StrictHostKeyChecking enabled for all L8s containers
- [ ] CA keys properly secured with correct permissions
- [ ] Seamless user experience (no additional steps)
- [ ] All tests passing
- [ ] Documentation updated

## Error Handling

### CA Key Missing
- Clear error message: "CA key not found. Run 'l8s init' to generate."
- Do not auto-generate (security risk)

### Certificate Signing Failure
- Log detailed error
- Suggest CA regeneration if corrupt
- Fail container creation (don't create insecure container)

### Known Hosts Issues
- Auto-create if missing
- Append CA entry if not present
- Handle permission errors gracefully

## Performance Impact

Minimal - certificate signing adds <100ms to container creation. No impact on SSH connection time as OpenSSH handles certificate validation natively.

## Future Enhancements (Out of Scope)

- User certificate authentication (replace authorized_keys)
- Certificate constraints/principals
- Automatic certificate renewal
- HSM/hardware key storage for CA
- Multiple CA support for different environments

## References

- [OpenSSH Certificates](https://www.openssh.com/txt/release-5.4)
- [SSH Certificate Tutorial](https://smallstep.com/blog/use-ssh-certificates/)
- [Facebook's SSH CA Implementation](https://engineering.fb.com/2016/09/12/security/scalable-and-secure-access-with-ssh/)