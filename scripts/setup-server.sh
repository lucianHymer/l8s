#!/bin/bash
# L8s Server Setup Script for Root Podman in LXC Container

set -e

echo "=== L8s Server Setup (Root Podman in LXC) ==="
echo
echo "This script sets up Podman running as root inside an LXC container."
echo "The LXC container provides isolation from the host system."
echo

# Note: This script is designed for Podman running as root inside an LXC container
# The LXC container provides isolation from the host system

# Detect OS
if [ -f /etc/os-release ]; then
    . /etc/os-release
    OS=$ID
else
    echo "Cannot detect OS. /etc/os-release not found."
    exit 1
fi

# Install Podman if not present
if ! command -v podman &> /dev/null; then
    echo "Installing Podman..."
    case $OS in
        fedora|rhel|centos)
            dnf install -y podman
            ;;
        ubuntu|debian)
            apt-get update
            apt-get install -y podman
            ;;
        *)
            echo "Unsupported OS: $OS"
            echo "Please install Podman manually."
            exit 1
            ;;
    esac
else
    echo "✓ Podman is already installed"
fi

# Enable Podman socket at system level
echo "Enabling Podman socket..."
systemctl enable --now podman.socket

# Verify socket is available
if [ ! -S /run/podman/podman.sock ]; then
    echo "ERROR: Podman socket not found at /run/podman/podman.sock"
    exit 1
fi

echo "✓ Podman socket is active"

# Test Podman
echo "Testing Podman..."
podman version

# Configure container policy to allow pulling images
echo "Configuring container policy..."
mkdir -p /etc/containers
cat > /etc/containers/policy.json <<EOF
{
    "default": [
        {
            "type": "insecureAcceptAnything"
        }
    ],
    "transports": {
        "docker": {
            "docker.io": [
                {
                    "type": "insecureAcceptAnything"
                }
            ]
        }
    }
}
EOF

echo "✓ Container policy configured"

# Configure registries
echo "Configuring container registries..."
cat > /etc/containers/registries.conf <<EOF
unqualified-search-registries = ["docker.io"]

[[registry]]
location = "docker.io"
insecure = false
EOF

echo "✓ Container registries configured"

# Set up SSH access
echo
echo "=== SSH Configuration ==="
echo

# Ensure SSH is installed and running
if ! command -v sshd &> /dev/null; then
    echo "Installing OpenSSH server..."
    case $OS in
        fedora|rhel|centos)
            dnf install -y openssh-server
            ;;
        ubuntu|debian)
            apt-get install -y openssh-server
            ;;
    esac
fi

# Enable and start SSH
systemctl enable --now sshd
echo "✓ SSH daemon is running"

# Get server hostname/IP
HOSTNAME=$(hostname -I | awk '{print $1}')
if [ -z "$HOSTNAME" ]; then
    HOSTNAME=$(hostname)
fi

echo
echo "=== Setup Complete ==="
echo
echo "Next steps:"
echo "1. Add your SSH key from your laptop:"
echo "   ssh-copy-id root@$HOSTNAME"
echo
echo "2. Configure l8s on your laptop:"
echo "   l8s init"
echo
echo "3. Use these settings when prompted:"
echo "   Remote host: $HOSTNAME"
echo "   Remote user: root"
echo "   Remote socket: /run/podman/podman.sock"
echo
echo "Note: Podman is running as root inside this LXC container,"
echo "      which provides isolation from the host system."
echo
echo "Security recommendations:"
echo "- This LXC container should be dedicated to l8s"
echo "- Restrict network access to trusted sources"
echo "- Regularly update the container OS and Podman"
echo