# Web Port Forwarding for Containers

L8s provides automatic port forwarding for web applications running inside containers, exposing them to the host system.

## Overview

Port 3000 (configurable) inside containers is automatically forwarded to the host machine, enabling access to web applications (Node.js, React, etc.) running in development containers.

## Implementation Details

### Configuration
- **WebPortStart**: Default port 3000 (configurable in config)
- **Port Assignment**: Uses consistent offset from SSH ports
  - Example: SSH port 2201 → Web port 3001
- **Conflict Prevention**: FindAvailablePort checks both SSH and web ports

### Container Setup
- Port mappings configured during container creation
- Format: Container port 3000 → Host port (WebPortStart + offset)
- Preserved during container rebuilds

### User Interface
- **info command**: Displays web port as `http://localhost:PORT`
- **list command**: Shows web port information
- **status command**: Includes web port in output

## Architecture

The web port forwarding follows the same proven pattern as SSH port mapping:
- Stored in Container and ContainerConfig types
- Managed by Podman port mappings
- Automatic port allocation using the same offset system

## Access Pattern

Once a container is running with a web application on port 3000:
```bash
l8s info          # Shows: Web: http://localhost:3001
curl http://localhost:3001  # Access the web app
```

## Related Files
- `pkg/config/config.go` - WebPortStart configuration
- `pkg/container/types.go` - WebPort field definitions
- `pkg/container/manager.go` - Port allocation logic
- `pkg/container/podman_client.go` - Podman port mapping
- `pkg/cli/handlers.go` - Info/list command display
