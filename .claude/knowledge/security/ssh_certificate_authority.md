# SSH Certificate Authority Implementation

L8s implements a complete SSH Certificate Authority system to provide cryptographic verification of container identities and protect against MITM attacks.

## Background

Previously, L8s disabled SSH host key verification completely by setting:
- `StrictHostKeyChecking no`
- `UserKnownHostsFile /dev/null`

This left container connections vulnerable to man-in-the-middle attacks.

## Implementation Overview

The SSH CA system provides:
- **Cryptographic verification** of container identities
- **Protection against MITM attacks**
- **Seamless user experience** (no SSH warnings)
- **10-year certificate validity** for dev environments
- **Automatic trust** for all L8s containers

## Architecture Components

### CA Package
**Location**: `pkg/ssh/ca.go`

Manages:
- CA keypair generation
- Host key signing
- Known_hosts entries creation

### Configuration
Added CA paths to both:
- `config.Config` struct
- `container.Config` struct

Default paths:
- CA private key: `~/.config/l8s/ca/ca_key`
- CA public key: `~/.config/l8s/ca/ca_key.pub`
- Known hosts: `~/.config/l8s/known_hosts`

### Init Command Integration
During `l8s init`:
1. Generates CA keypair
2. Creates known_hosts file with CA trust entry
3. Stores CA keys in config directory

### Container Creation Flow
During container creation:
1. Generates container-specific SSH host keypair
2. Signs host key with CA (10-year validity)
3. Copies keys and certificates to container BEFORE startup
4. Configures sshd with certificate support

### SSH Config Generation
SSH configs now use:
- `StrictHostKeyChecking yes` (secure by default)
- `UserKnownHostsFile ~/.config/l8s/known_hosts` (CA-trusted hosts)

## Timing Optimization

Certificate setup happens before container startup:
- Uses `podman cp` to copy files to stopped containers
- Eliminates runtime dependencies
- No need for sshd restart/reload
- Ensures proper configuration from first boot

## Testing

Comprehensive test coverage in `pkg/ssh/ca_test.go`:
- CA generation
- Certificate signing
- Known_hosts entry creation
- Integration with existing tests

## Security Benefits

1. **Identity Verification**: Every container's SSH host key is cryptographically signed
2. **MITM Protection**: Strict host key checking prevents connection hijacking
3. **Trust Chain**: Single CA manages trust for all L8s containers
4. **No User Friction**: Automatic trust establishment, no manual fingerprint verification

## Related Files
- `pkg/ssh/ca.go` - CA implementation
- `pkg/ssh/ca_test.go` - CA tests
- `pkg/config/config.go` - Config with CA paths
- `pkg/container/types.go` - Container config with CA paths
- `pkg/container/manager.go` - Container creation with certificates
- `pkg/cli/handlers.go` - Init command CA generation
- `pkg/ssh/keys.go` - SSH config with strict checking