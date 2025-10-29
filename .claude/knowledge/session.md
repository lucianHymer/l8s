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

### [13:10] [gotcha] containerfiles location
**Details**: The Containerfiles are located in pkg/embed/containers/, not pkg/containerfiles/. The main Containerfile is at pkg/embed/containers/Containerfile.
**Files**: pkg/embed/containers/Containerfile
---

### [18:35] [testing] Running tests with missing system dependencies
**Details**: When system dependencies like gpgme or btrfs are missing, tests can still run using build tags. The Makefile already includes this: `make test` and `make test-go` use `-tags exclude_graphdriver_btrfs,exclude_graphdriver_devicemapper` to exclude optional dependencies. This allows testing even without full Podman dependencies installed locally.
**Files**: Makefile, pkg/ssh/keys_test.go, pkg/cli/handlers_test.go
---

### [18:35] [gotcha] SSH commands and PATH dependency issues
**Details**: When executing commands via SSH non-interactively (ssh host "command"), the remote shell doesn't source .zshrc or set up PATH properly. This causes "command not found" errors even if the command exists in ~/.local/bin. Solution: Use fully qualified paths like "~/.local/bin/team" instead of relying on PATH. The ~ is expanded by SSH automatically. Avoid wrapping in "zsh -l -c" as it's unnecessary with fully qualified paths.
**Files**: pkg/cli/handlers.go
---

### [18:35] [pattern] Standalone scripts vs shell functions for SSH execution
**Details**: When a command needs to be called via SSH from the host, it should be a standalone executable script rather than a shell function. Shell functions in .zshrc only work in interactive shells. Solution: Create scripts in ~/.local/bin/ (or another bin directory) and mark them executable in the executableFiles map in manager.go. The embedded dotfiles system preserves this and deploys them properly. Example: team command moved from .zshrc function to .local/bin/team script.
**Files**: pkg/embed/dotfiles/.local/bin/team, pkg/container/manager.go, pkg/embed/dotfiles/.zshrc
---

### [18:35] [architecture] Team session management with dtach
**Details**: The team command provides persistent terminal sessions using dtach. Sessions are stored as /tmp/dtach-<base64-name>.sock and survive SSH disconnections. The DTACH_SESSION environment variable tracks the active session name and is displayed in the Claude Code statusline with a ⚒ icon. L8s integration: `l8s team <name>` joins/creates sessions, `l8s team list` shows active sessions. Both commands use git-native architecture to derive container from current worktree. The team script is deployed to ~/.local/bin/team with executable permissions.
**Files**: pkg/embed/dotfiles/.local/bin/team, pkg/cli/handlers.go, pkg/embed/dotfiles/.claude/statusline.sh
---

### [18:35] [config] SSH connection stability improvements
**Details**: Enhanced SSH connection stability with improved keepalive settings in GenerateSSHConfigEntry: ControlPersist changed from 10m to 1h (multiplexed connections last longer), added ServerAliveInterval 30 (client sends keepalive every 30s), added ServerAliveCountMax 6 (tolerates 6 failed keepalives = 3 min total before disconnect), added ConnectTimeout 10 (faster timeout on initial connection), added TCPKeepAlive yes (explicit TCP-level keepalives). These settings apply to both CA-enabled and fallback SSH configs. Benefit: Better survival through brief network hiccups, faster detection of dead connections (~3 min vs indefinite hang).
**Files**: pkg/ssh/keys.go, pkg/ssh/keys_test.go
---

