# L8s Remote Server Setup Guide

This guide walks you through setting up a remote server for l8s container management.

## Why Remote-Only?

L8s is designed to be remote-only for security isolation. This ensures:
- No code execution on developer laptops
- Complete isolation for AI workloads
- Safe execution of untrusted code
- Consistent development environments across teams

## Recommended Setup: LXC Container

We recommend running Podman inside an LXC container for additional isolation:

```bash
# On your host server (e.g., Proxmox)
lxc-create -n l8s-podman -t download -- -d fedora -r 39 -a amd64

# Start and enter the container
lxc-start -n l8s-podman
lxc-attach -n l8s-podman
```

## Quick Setup

### Option 1: Automated Setup Script

Inside your LXC container or dedicated server:

```bash
# Download and run the setup script
curl -fsSL https://raw.githubusercontent.com/l8s/l8s/main/scripts/setup-server.sh | bash
```

### Option 2: Manual Setup

1. **Install Podman**:
   ```bash
   # Fedora/RHEL
   dnf install -y podman
   
   # Ubuntu/Debian
   apt-get update && apt-get install -y podman
   ```

2. **Enable Podman Socket**:
   ```bash
   # For root Podman (recommended in LXC)
   systemctl enable --now podman.socket
   
   # Verify socket is available
   ls -la /run/podman/podman.sock
   ```

3. **Configure SSH Access**:
   ```bash
   # Ensure SSH is running
   systemctl enable --now sshd
   
   # Add your SSH key from your laptop
   # On your laptop:
   ssh-copy-id root@your-server
   ```

## Client Configuration

On your development laptop:

1. **Install l8s**:
   ```bash
   git clone https://github.com/l8s/l8s.git
   cd l8s
   make build
   sudo make install
   ```

2. **Configure Remote Connection**:
   ```bash
   l8s init
   
   # You'll be prompted for:
   # Remote host: your-server.example.com
   # Remote user: root (if using LXC)
   # Remote socket: /run/podman/podman.sock
   ```

3. **Test Connection**:
   ```bash
   # Verify l8s can connect
   l8s list
   ```

## Network Configuration

### Firewall Rules

If using a firewall, ensure these are allowed:
- SSH (port 22) from your laptop
- Container SSH ports (default 2200-2299) from your laptop

```bash
# Example with firewalld
firewall-cmd --permanent --add-service=ssh
firewall-cmd --permanent --add-port=2200-2299/tcp
firewall-cmd --reload
```

### SSH Configuration

L8s uses SSH connection multiplexing for performance. Ensure your SSH client supports ControlMaster:

```bash
# Test SSH multiplexing
ssh -o ControlMaster=auto -o ControlPath=/tmp/test-%r@%h:%p your-server echo "OK"
```

## Security Best Practices

1. **Use LXC Containers**: Run Podman inside LXC for additional isolation
2. **Dedicated Server**: Use a dedicated server or VM for l8s containers
3. **SSH Keys Only**: Never enable password authentication
4. **Firewall**: Restrict access to known IP addresses
5. **Updates**: Keep the server and Podman updated

## Troubleshooting

### Connection Errors

```bash
# Test SSH connectivity
ssh root@your-server echo "Connected"

# Check Podman socket
ssh root@your-server systemctl status podman.socket

# Test Podman
ssh root@your-server podman version
```

### Socket Permission Issues

If you see permission errors:

```bash
# Check socket permissions
ls -la /run/podman/podman.sock

# For root Podman, socket should be owned by root
# For rootless, it should be in user's XDG_RUNTIME_DIR
```

### SSH Agent Issues

L8s requires ssh-agent:

```bash
# Start ssh-agent
eval $(ssh-agent)

# Add your key
ssh-add ~/.ssh/id_ed25519

# Verify
ssh-add -l
```

## Advanced Configuration

### Multiple Servers

You can edit `~/.config/l8s/config.yaml` to switch between servers:

```yaml
# Development server
remote_host: "dev.example.com"
remote_user: "root"

# Production server (commented out)
# remote_host: "prod.example.com"
# remote_user: "root"
```

### Custom Podman Socket

If your Podman socket is in a non-standard location:

```yaml
remote_socket: "/custom/path/to/podman.sock"
```

## Support

For issues or questions:
- GitHub Issues: https://github.com/l8s/l8s/issues
- Documentation: https://github.com/l8s/l8s/docs