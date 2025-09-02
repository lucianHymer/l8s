# SSH Certificate Authority Feature

The SSH Certificate Authority feature provides secure, cryptographically-verified connections to L8s containers.

## User Experience

From a user perspective, the SSH CA is completely transparent:
- No SSH warnings or prompts about unknown hosts
- Automatic trust for all L8s containers
- Seamless SSH connections with full security
- No manual fingerprint verification needed

## How It Works

1. **First-time setup** (`l8s init`):
   - Generates a Certificate Authority keypair
   - Creates trusted known_hosts file
   - Stores CA in `~/.config/l8s/ca/`

2. **Container creation**:
   - Each container gets a unique SSH host key
   - Host key is signed by the CA
   - Certificate valid for 10 years
   - Certificates copied before container starts

3. **SSH connections**:
   - SSH verifies container identity via CA signature
   - Strict host key checking enabled
   - Protection against MITM attacks
   - No user interaction required

## Security Benefits

- **Cryptographic identity verification** for every container
- **MITM attack prevention** through strict checking
- **Automated trust management** via CA chain
- **No security warnings** that users might ignore

## Configuration

CA files stored in:
- `~/.config/l8s/ca/ca_key` - Private CA key
- `~/.config/l8s/ca/ca_key.pub` - Public CA key
- `~/.config/l8s/known_hosts` - Trusted hosts file

## Related Files
- `pkg/ssh/ca.go` - Core CA implementation
- `pkg/container/manager.go` - Integration with container creation
- `pkg/cli/handlers.go` - Init command CA setup