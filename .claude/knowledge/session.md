### [14:35] [architecture] Container Creation Flow and Port Forwarding Architecture
**Details**: ## Container Creation Entry Points

**CLI Handler**: `pkg/cli/handlers.go` - `runCreate()` at line 23
- Validates git repository context
- Gets container name from worktree
- Finds SSH public key
- Calls Manager.CreateContainer()

**Manager Entry**: `pkg/container/manager.go` - `CreateContainer()` at line 39
- Validates container name
- Finds available SSH port (starting from SSHPortStart config)
- Finds available web port with CONSISTENT OFFSET from SSH port:
  - webPortOffset = sshPort - SSHPortStart
  - webPort = WebPortStart + webPortOffset
- Creates ContainerConfig with both port numbers
- Calls client.CreateContainer()
- Sets up SSH certificates BEFORE starting (line 115)
- Starts container (line 123)
- Fixes volume ownership
- Sets up SSH
- Copies dotfiles
- Initializes git repository
- Adds SSH config entry (line 157)
- Adds git remote

## PodmanClient Port Configuration

**Client Implementation**: `pkg/container/podman_client.go` - `CreateContainer()` at line 186
- Creates SpecGen for container
- Sets up port mappings (lines 194-205):
  - SSH: Container port 22 → HostPort (from config.SSHPort)
  - Web: Container port 3000 → HostPort (from config.WebPort)
- Creates named volumes for home and workspace with "U" option (for user namespace)
- Sets environment variables (USER, HOME)
- Command is `/usr/sbin/sshd -D`

## ContainerConfig Type

**Location**: `pkg/container/types.go` - `ContainerConfig` struct at line 18
Fields:
- Name: Full container name with prefix
- SSHPort: Host port mapped to container port 22
- WebPort: Host port mapped to container port 3000
- SSHPublicKey: Public key for authorized_keys
- BaseImage: Image to create from
- ContainerUser: Container user account
- Labels: Map including LabelSSHPort and LabelWebPort

## Configuration Defaults

**Location**: `pkg/config/config.go`
- SSHPortStart: 2200
- WebPortStart: 3000
- Container type `Config` stores these in lines 45-47

## Web Port Forwarding Implementation Details

1. **Port Allocation** (manager.go:74-76):
   - Consistent offset system maintains same relative port spacing
   - If SSH is 2201, web is 3001
   - If SSH is 2205, web is 3005

2. **Podman Port Mapping** (podman_client.go:194-205):
   - Creates PortMapping for port 3000 in container
   - Maps to calculated WebPort on host

3. **Container Info Retrieval**:
   - ListContainers (line 321-345): Checks port mappings for port 3000, falls back to labels
   - GetContainerInfo (line 379-409): Same logic for single container
   - Stores in Container.WebPort field (types.go:13)

4. **Display in CLI** (handlers.go):
   - List command (lines 229-231): Shows WebPort with "-" if 0
   - Info command (lines 363-365): Displays as "Web: http://localhost:PORT"

## Container Rebuild and Port Preservation

**RebuildContainer** (manager.go:799-879):
- Step 1: Gets current container info including WebPort (line 813)
- Step 2: Stops container
- Step 3: Removes container (preserves named volumes)
- Step 4: Creates new container with same SSH and Web ports (lines 851-852)
- Step 5: Sets up SSH certificates again
- Step 6: Starts new container

## Key Design Pattern for Audio Support

The web port forwarding pattern is perfect for audio support:
1. Add AudioPort field to ContainerConfig and Container types (similar to WebPort)
2. Allocate port with same offset system in CreateContainer
3. Add port mapping in podman_client.go CreateContainer (similar to port 3000)
4. Handle port in all info/list retrieval methods
5. Update RebuildContainer to preserve the audio port

No changes needed to config structure - just reuse WebPortStart pattern.
**Files**: pkg/cli/handlers.go, pkg/container/manager.go, pkg/container/podman_client.go, pkg/container/types.go, pkg/config/config.go
---

### [14:35] [architecture] SSH Port Forwarding and Audio Tunnel Architecture
**Details**: ## SSH Configuration and Port Forwarding

L8s uses SSH Config entries generated in `GenerateSSHConfigEntry()` in `pkg/ssh/keys.go:70-110`. The function creates SSH config blocks with no LocalForward or RemoteForward directives currently.

### SSH Config Structure (lines 70-92 in pkg/ssh/keys.go)
Each container gets an SSH config entry with:
- Host alias: Container name (e.g., `dev-myproject`)
- HostName: Remote server address
- Port: Unique SSH port (e.g., 2200, 2201, etc.)
- User: Container user (e.g., `dev`)
- StrictHostKeyChecking: yes (with CA) or no (fallback)
- ControlMaster: auto (connection multiplexing enabled)
- ControlPath: ~/.ssh/control-%r@%h:%p
- ControlPersist: 1h (connections stay alive 1 hour)
- ServerAliveInterval: 30 seconds
- ServerAliveCountMax: 6 (3 minute tolerance)
- ConnectTimeout: 10 seconds
- TCPKeepAlive: yes

**Key insight**: No LocalForward or RemoteForward directives exist. This is where audio tunnel (or any port forwarding) would be added.

## Port Allocation System

### Configuration (pkg/config/config.go:38-39)
- SSHPortStart: Default 2200 (configurable in config.yaml)
- WebPortStart: Default 3000 (configurable in config.yaml)
- Both stored in Config struct with validation (lines 94-101)

### Port Finding (pkg/container/podman_client.go:423-452)
FindAvailablePort() scans for available ports:
1. Lists all containers with l8s.managed label
2. Builds map of ports in use (both SSH and Web) for running containers only
3. Finds first available port in range: startPort to startPort+100
4. Searches locally (doesn't verify remote availability)

### Container Port Allocation (pkg/container/manager.go:68-79)
During CreateContainer:
1. Find available SSH port starting from SSHPortStart
2. Calculate WebPortOffset = sshPort - SSHPortStart
3. Find web port at: WebPortStart + webPortOffset (maintains consistent offset)

Example: SSHPortStart=2200, WebPortStart=3000
- First container: SSH 2200, Web 3000 (offset 0)
- Second container: SSH 2201, Web 3001 (offset 1)

### Port Mappings (pkg/container/podman_client.go:194-205)
In CreateContainer(), PortMappings array creates Podman bindings:
- Port 22 (SSH) → hostPort=SSHPort
- Port 3000 (web) → hostPort=WebPort

### Port Storage (pkg/container/types.go:62-63)
Two label constants for persistence:
- LabelSSHPort = "l8s.ssh.port"
- LabelWebPort = "l8s.web.port"
Stored in container labels during creation.

### Port Retrieval (pkg/container/podman_client.go:321-348)
When listing containers, ports retrieved from:
1. First: Actual port mappings from inspect (NetworkSettings.Ports)
2. Fallback: Container labels if mappings unavailable
Both SSH and WebPort retrieved this way, supporting port recovery.

## Initialization and Setup

### Init Command (pkg/cli/handlers.go:606-834)
The runInit handler performs:
1. Connection configuration (server address, username)
2. SSH connectivity test (lines 656-672)
3. Container configuration (prefix, image, user, ports)
4. SSH CA generation (lines 745-772)
5. Known hosts setup for CA trust
6. GitHub token configuration (optional, lines 774-812)

**No audio configuration here yet** - this is where AudioPortStart and AudioPort configuration would be added.

### Container Creation Flow (pkg/container/manager.go:38-160)
1. Validate container name
2. Check container doesn't exist
3. Find available SSH port
4. Find available web port (offset-based)
5. Create container config with port labels
6. Create container
7. Setup SSH certificates (before start)
8. Start container
9. Fix volume ownership
10. Setup SSH auth
11. Copy dotfiles
12. Initialize git repository
13. Push code to container
14. Replicate origin remote

**Where audio setup would fit**: Could be inserted after step 12, during initial container setup phase.

## SSH Config Entry Generation

Function signature (pkg/ssh/keys.go:70):
```go
func GenerateSSHConfigEntry(
    containerName string, 
    sshPort int, 
    containerUser, 
    prefix, 
    remoteHost string, 
    knownHostsPath string
) string
```

Current output format (no forwarding):
```
Host dev-container-name
    HostName remote.server
    Port 2200
    User dev
    [CA or insecure settings]
    [Keepalive settings]
```

**To add audio forwarding** would require:
- Adding new parameter (e.g., audioPort int)
- Adding LocalForward line if audioPort > 0
- Example: LocalForward 127.0.0.1:59999 127.0.0.1:59999

## AddSSHConfig Wrapper (pkg/ssh/keys.go:304-326)

Wrapper function that:
1. Loads configuration
2. Gets active connection address
3. Calls GenerateSSHConfigEntry() with config values
4. Calls AddSSHConfigEntry() to write to ~/.ssh/config

This is where audio port would be passed through from config to SSH entry generation.
**Files**: pkg/ssh/keys.go, pkg/config/config.go, pkg/container/types.go, pkg/container/manager.go, pkg/container/podman_client.go, pkg/cli/handlers.go
---

### [14:37] [architecture] Volume Mount System and Audio Socket Support
**Details**: L8s uses Podman's specgen package for container configuration. The volume mounting system has two approaches:

1. **Named Volumes** (Current Implementation):
   - Used for persistent home and workspace directories
   - Type: `specgen.NamedVolume` with fields: Name, Dest, Options, SubPath
   - Options: "U" flag enables user namespace remapping for ownership
   - Location in code: lines 211-222 of podman_client.go

2. **Bind Mounts** (Available for Audio Support):
   - Type: `spec.Mount` from OCI spec (github.com/opencontainers/runtime-spec/specs-go)
   - Structure has: Destination, Source, Type ("bind"), Options []string
   - Mounted via: `s.Mounts` field on SpecGenerator (ContainerStorageConfig.Mounts)
   - Options: "ro", "rw", "nosuid", "nodev", "noexec", "Z", "z", etc.
   - No user namespace remapping like named volumes

3. **Environment Variables**:
   - Set via: `s.Env = map[string]string{}`
   - Location: lines 225-228 of podman_client.go
   - Can add PULSE_SERVER, DBUS_SESSION_BUS_ADDRESS, etc. here

Audio socket mounting (e.g., /run/user/1000/pulse) would use bind mounts with spec.Mount, not named volumes. User namespace ("U" option) applies to named volumes but bind mounts require explicit permission handling through options like "Z" or "z" depending on SELinux context needs.

The "U" option on named volumes handles user remapping for ownership conflicts since containers run as non-root user.
**Files**: pkg/container/podman_client.go (lines 207-228), pkg/container/types.go (ContainerStorageConfig definition)
---

