### [20:55] [features] Web port forwarding for containers
**Details**: Implemented port forwarding for port 3000 from containers to the host system. This exposes web applications running on port 3000 inside containers (common for Node.js, React, and other web frameworks) to the host machine. 

Implementation details:
- Added WebPortStart configuration field (default: 3000) to Config struct
- Added WebPort field to Container and ContainerConfig types  
- Web ports use consistent offset from SSH ports (e.g., SSH 2201 → Web 3001)
- Port mappings configured in Podman CreateContainer with port 3000 → host port
- FindAvailablePort checks both SSH and web ports to avoid conflicts
- RebuildContainer preserves web port mappings
- Info and list commands display web port information
- Web access shown as http://localhost:PORT in info command

The feature follows the exact same pattern as SSH port mapping which was already well-tested in the codebase.
**Files**: pkg/config/config.go, pkg/container/types.go, pkg/container/manager.go, pkg/container/podman_client.go, pkg/cli/handlers.go
---

