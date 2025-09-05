# L8s Test Server Setup

This document describes setting up a test server for L8s development from within L8s containers.

## Architecture Overview

Dev containers can test L8s against a dedicated test server:
- Dev container runs `l8s init` pointing to test server
- Test server accepts certificate-based auth from dev containers
- Single shared test server (use one at a time by convention)

## One-Time Test Server Setup (Admin)

### 1. Create Test LXC
```bash
# On hypervisor
lxc launch ubuntu:24.04 l8s-test-target
lxc config set l8s-test-target security.nesting true
```

### 2. Install Dependencies
```bash
lxc exec l8s-test-target -- bash
apt update && apt install -y podman openssh-server sudo
```

### 3. Create Test User
```bash
useradd -m -s /bin/bash dev
echo "dev ALL=(ALL) NOPASSWD: /usr/bin/podman" >> /etc/sudoers.d/l8s
```

### 4. Configure SSH for Certificate Auth
```bash
# Will add CA key in step 5
echo "TrustedUserCAKeys /etc/ssh/l8s_ca.pub" >> /etc/ssh/sshd_config
```

### 5. Extract and Install L8s CA Public Key
From any existing L8s installation:
```bash
# Extract CA public key
cat ~/.config/l8s/ca/ca_key.pub

# Copy output to test server
lxc exec l8s-test-target -- bash -c 'cat > /etc/ssh/l8s_ca.pub'
# [paste the key]

# Restart SSH
lxc exec l8s-test-target -- systemctl restart sshd
```

### 6. Fix Podman Permissions (Survives Reboot)
```bash
lxc exec l8s-test-target -- bash -c 'cat > /etc/tmpfiles.d/podman.conf' <<EOF
d /run/podman 0750 root podman -
EOF
```

## Using Test Server from Dev Container

### 1. Initialize L8s in Container
```bash
# Inside dev container
l8s init
# Enter test server IP when prompted
# Use same settings as production but different remote_host
```

### 2. Generate and Sign User Certificate
```bash
# Generate key in container
ssh-keygen -t ed25519 -f ~/.ssh/id_ed25519 -N ""

# Admin signs it (from host with CA access)
ssh-keygen -s ~/.config/l8s/ca/ca_key \
  -I "dev-container-$(date +%s)" \
  -n "dev" \
  -V +30d \
  ~/.ssh/id_ed25519.pub

# Copy cert back to container
# Now container can SSH to test server!
```

### 3. Test L8s Operations
```bash
# From dev container
l8s create test-container
l8s list
l8s ssh test-container
l8s remove test-container
```

## Recovery Instructions

If test server needs rebuilding, follow "One-Time Test Server Setup" again.

Key files to preserve:
- `/etc/ssh/l8s_ca.pub` - CA public key for certificate trust
- Test server IP address

## Security Notes

- Test server is isolated from production
- Certificate auth prevents unauthorized access
- Single shared server (coordinate usage)
- No sensitive data on test server