# Remote Server Setup for l8s

This guide provides detailed instructions for setting up your remote server to work with l8s. l8s uses SSH to connect to a remote Podman instance for secure container management.

## Prerequisites

- A Linux server (Fedora, RHEL, CentOS, Ubuntu, or Debian)
- SSH access to the server
- A non-root user with sudo privileges

## Step 1: Install Podman

### Fedora/RHEL/CentOS
```bash
sudo dnf install -y podman
```

### Ubuntu/Debian
```bash
sudo apt-get update
sudo apt-get install -y podman
```

## Step 2: Create a Dedicated User (Optional but Recommended)

If you want to use a dedicated user for l8s operations:

```bash
# Create a new user
sudo useradd -m -s /bin/bash poduser

# Add the user to the wheel/sudo group for sudo access
sudo usermod -a -G wheel poduser  # For Fedora/RHEL/CentOS
# OR
sudo usermod -a -G sudo poduser   # For Ubuntu/Debian

# Set a password (or set up SSH key authentication)
sudo passwd poduser
```

## Step 3: Configure Podman for Remote Access

l8s requires access to the system (root) Podman socket for container management. We'll set this up securely using a dedicated group.

### 3.1 Create the Podman Group

```bash
# Create a podman group
sudo groupadd -f podman

# Add your user to the podman group
sudo usermod -a -G podman poduser  # Replace 'poduser' with your username
```

### 3.2 Configure Podman Socket Permissions

Create a systemd override to ensure the Podman socket has the correct permissions:

```bash
# Create the override directory
sudo mkdir -p /etc/systemd/system/podman.socket.d

# Create the override configuration
sudo tee /etc/systemd/system/podman.socket.d/override.conf << 'EOF'
[Socket]
SocketMode=0660
SocketGroup=podman
EOF

# Reload systemd and restart the podman socket
sudo systemctl daemon-reload
sudo systemctl enable --now podman.socket
```

### 3.3 Configure Directory Permissions

The Podman runtime directory needs proper permissions for security:

```bash
# Set secure permissions on the podman directory
sudo chgrp podman /run/podman
sudo chmod 750 /run/podman

# Make these permissions persistent across reboots
# Since /run is a tmpfs (wiped on reboot), we need systemd-tmpfiles to recreate permissions
sudo tee /etc/tmpfiles.d/podman.conf << 'EOF'
# Ensure /run/podman has correct permissions for security
# Only root and podman group members can access the directory
# Format: type path mode user group age
d /run/podman 0750 root podman -
EOF

# Verify the permissions are correct
ls -la /run/podman/podman.sock
# Should show: srw-rw---- 1 root podman ... /run/podman/podman.sock
```

## Step 4: Configure SSH Access

### 4.1 Set Up SSH Key Authentication

On your local machine:

```bash
# Generate an SSH key if you don't have one
ssh-keygen -t rsa -b 4096 -f ~/.ssh/id_rsa

# Copy your public key to the server
ssh-copy-id poduser@your-server-ip
```

### 4.2 Configure SSH Agent

l8s requires ssh-agent for secure connections:

```bash
# Start ssh-agent
eval $(ssh-agent)

# Add your SSH key
ssh-add ~/.ssh/id_rsa
```

## Step 5: Configure Sudo for Passwordless Podman Access

For the user to run Podman commands with sudo without a password prompt:

```bash
# Edit sudoers file
sudo visudo

# Add this line (replace 'poduser' with your username):
poduser ALL=(ALL) NOPASSWD: /usr/sbin/podman, /usr/bin/podman
```

## Step 6: Verify the Setup

Test that everything is working correctly:

```bash
# Test SSH connection
ssh poduser@your-server-ip "echo 'SSH connection works'"

# Test podman access via the group
ssh poduser@your-server-ip "ls -la /run/podman/podman.sock"

# Test sudo podman access
ssh poduser@your-server-ip "sudo podman version"

# Test podman info
ssh poduser@your-server-ip "sudo podman info"
```

## Step 7: Configure l8s

On your local machine, initialize l8s with your server details:

```bash
l8s init
```

Enter the following when prompted:
- Remote host: `your-server-ip`
- Remote user: `poduser` (or your username)
- SSH key path: `/home/youruser/.ssh/id_rsa`
- Container user: `youruser` (the user inside containers)
- Base image: `fedora:latest` (or your preferred base)

## Troubleshooting

### Permission Denied on Socket

If you get "permission denied" errors when accessing the socket:

1. Ensure the user is in the podman group:
   ```bash
   ssh poduser@your-server-ip "groups"
   # Should show: poduser wheel podman
   ```

2. If the group was just added, you may need to log out and back in:
   ```bash
   ssh poduser@your-server-ip "sudo -u poduser groups"
   ```

3. Check socket permissions:
   ```bash
   ssh poduser@your-server-ip "sudo ls -la /run/podman/podman.sock"
   # Should show: srw-rw---- 1 root podman ...
   ```

4. Check directory permissions:
   ```bash
   ssh poduser@your-server-ip "ls -ld /run/podman"
   # Should show: drwxr-x--- ... root podman /run/podman
   ```
   
   **Important**: After a system reboot, `/run/podman` may revert to 700 permissions (root-only) since `/run` is a tmpfs. If this happens:
   ```bash
   ssh poduser@your-server-ip "sudo chgrp podman /run/podman && sudo chmod 750 /run/podman"
   ```
   
   To make this permanent, create a systemd tmpfiles configuration:
   ```bash
   sudo tee /etc/tmpfiles.d/podman.conf << 'EOF'
   # Ensure /run/podman has correct permissions for security
   # Only root and podman group members can access the directory
   # Format: type path mode user group age
   d /run/podman 0750 root podman -
   EOF
   ```
   
   This tells systemd to recreate the directory with proper permissions on each boot (since `/run` is a tmpfs that gets cleared).

### SSH Key Issues

If you get SSH key-related errors:

1. Ensure ssh-agent is running:
   ```bash
   echo $SSH_AUTH_SOCK
   # Should show a socket path
   ```

2. Check that your key is loaded:
   ```bash
   ssh-add -l
   # Should show your key fingerprint
   ```

### Podman Service Not Running

If the Podman socket is not active:

```bash
# Check status
ssh poduser@your-server-ip "sudo systemctl status podman.socket"

# Enable and start
ssh poduser@your-server-ip "sudo systemctl enable --now podman.socket"
```

### SELinux Issues (Fedora/RHEL/CentOS)

If you're on a system with SELinux enabled and encounter issues:

```bash
# Check for SELinux denials
sudo ausearch -m avc -ts recent

# If needed, create a policy module (example)
sudo ausearch -m avc -ts recent | audit2allow -M mypodman
sudo semodule -i mypodman.pp
```

## Security Considerations

1. **User Isolation**: The poduser (or your chosen user) has sudo access to Podman only, limiting potential security exposure.

2. **Group-based Access**: Using the podman group provides controlled access to the socket without requiring full root access.

3. **SSH Key Authentication**: Always use SSH keys instead of passwords for better security.

4. **Firewall**: Ensure your firewall only allows SSH access from trusted IPs:
   ```bash
   sudo firewall-cmd --permanent --add-rich-rule='rule family="ipv4" source address="YOUR.CLIENT.IP.ADDRESS" port protocol="tcp" port="22" accept'
   sudo firewall-cmd --reload
   ```

## Next Steps

Once your server is configured, you can:

1. Build the l8s base image:
   ```bash
   l8s build
   ```

2. Create your first development container:
   ```bash
   l8s create myproject https://github.com/yourusername/yourproject
   ```

3. Connect to your container:
   ```bash
   l8s ssh myproject
   ```

For more information, see the [main README](../README.md).