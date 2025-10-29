# Team Command Implementation Plan

## Overview

This document outlines the implementation plan for promoting the container-side `team` command (dtach session management) to first-class l8s CLI commands. This will allow users to manage persistent terminal sessions from their laptop rather than only from within containers.

Additionally, this document includes SSH connection stability improvements that complement the team session feature by making connections more resilient to network changes and laptop sleep.

## Background

### Current Implementation

L8s containers already have a sophisticated `team` command built into `.zshrc` (lines 68-146 in `pkg/embed/dotfiles/.zshrc`):

```bash
# Inside containers
team <name>              # Create or attach to a session
team ls|list             # List active sessions
team attach <name>       # Attach to existing session (read-only)
team create <name>       # Create new session
```

**How it works:**
- Uses `dtach` for session persistence (survives SSH disconnections)
- Sessions stored as `/tmp/dtach-<base64-name>.sock`
- Session names are base64-encoded to handle special characters
- Active session name exported as `DTACH_SESSION` environment variable
- Claude Code statusline shows active team with ⚒ icon (see `.claude/statusline.sh:78-81`)

**Key benefit:** When SSH connections die (laptop sleep, network change), the team session persists. Reconnect and reattach to continue exactly where you left off.

### Motivation for L8s Integration

Currently, users must:
1. SSH into container
2. Run `team <name>` manually
3. Remember which team sessions exist in which containers

With l8s integration, users can:
1. List all team sessions across containers from their laptop
2. Jump directly into a team session with one command
3. Discover which containers have active collaboration sessions

## Command Specifications

### Command Structure

Add a new command group "Team Management" with these commands:

```bash
# Primary command: SSH into container and join/create team session
l8s team <container> <session-name>

# List team sessions in a specific container
l8s team list <container>

# List team sessions across all containers (optional enhancement)
l8s team list --all
```

### Command Details

#### `l8s team <container> <session-name>`

**Purpose:** SSH into container and immediately join/create the specified team session.

**Arguments:**
- `<container>`: Container name (without dev- prefix, like other l8s commands)
- `<session-name>`: Name of the team session to join/create

**Behavior:**
- Validates container exists and is running
- Executes: `ssh -t <container> "team <session-name>"`
- User lands directly in the team session
- If session doesn't exist, it's created automatically (dtach -A behavior)

**Example:**
```bash
l8s team myproject backend
# SSH into dev-myproject and join/create "backend" team session
```

#### `l8s team list <container>`

**Purpose:** Show all active team sessions in a specific container.

**Arguments:**
- `<container>`: Container name (without dev- prefix)

**Behavior:**
- Validates container exists and is running
- Executes: `ssh <container> "team ls"` via ExecContainer
- Parses output and displays in user-friendly format
- Returns exit code 0 if container found, non-zero otherwise

**Output format:**
```
Active team sessions in dev-myproject:
  backend
  frontend
  debugging
```

**Example:**
```bash
l8s team list myproject
```

#### `l8s team list --all` (Optional Enhancement)

**Purpose:** Show all team sessions across all running containers.

**Behavior:**
- Iterates through all running containers
- Executes `team ls` in each
- Displays results grouped by container

**Output format:**
```
Team sessions across all containers:

dev-myproject:
  backend
  frontend

dev-another:
  work
  test
```

## Technical Implementation

### 1. Add Team Commands to CLI Factory

**File:** `pkg/cli/factory_lazy.go`

Add new command factory method:

```go
func (f *LazyCommandFactory) TeamCmd() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "team",
        Short: "Manage persistent team sessions in containers",
        Long: `Manage dtach-based persistent terminal sessions that survive SSH disconnections.

Team sessions allow multiple users (or the same user across reconnections) to share
a persistent terminal session inside a container. Sessions survive laptop sleep,
network changes, and SSH disconnections.`,
        GroupID: "container-mgmt",
    }

    // Add subcommands
    cmd.AddCommand(f.TeamJoinCmd())
    cmd.AddCommand(f.TeamListCmd())

    return cmd
}

func (f *LazyCommandFactory) TeamJoinCmd() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "join <container> <session-name>",
        Short: "Join or create a team session in a container",
        Long: `SSH into the container and join the specified team session.
If the session doesn't exist, it will be created automatically.

The session persists across SSH disconnections, so you can reconnect later.`,
        Args:  cobra.ExactArgs(2),
        RunE: func(cmd *cobra.Command, args []string) error {
            if err := f.ensureInitialized(); err != nil {
                return err
            }
            return f.factory.runTeamJoin(cmd.Context(), args[0], args[1])
        },
    }
    return cmd
}

func (f *LazyCommandFactory) TeamListCmd() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "list <container>",
        Short: "List active team sessions in a container",
        Args:  cobra.MinimumNArgs(1),
        RunE: func(cmd *cobra.Command, args []string) error {
            if err := f.ensureInitialized(); err != nil {
                return err
            }
            return f.factory.runTeamList(cmd.Context(), args[0])
        },
    }

    // Optional: add --all flag
    cmd.Flags().BoolP("all", "a", false, "List sessions across all containers")

    return cmd
}
```

**Note:** We could make the primary command `l8s team <container> <session>` instead of `l8s team join <container> <session>`. The shorter form is more ergonomic:

```go
// Alternative: Make team command directly callable
func (f *LazyCommandFactory) TeamCmd() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "team <container> <session-name>",
        Short: "Join or create a team session in a container",
        Args:  cobra.ExactArgs(2),
        RunE: func(cmd *cobra.Command, args []string) error {
            if err := f.ensureInitialized(); err != nil {
                return err
            }
            return f.factory.runTeamJoin(cmd.Context(), args[0], args[1])
        },
    }

    // Add list as subcommand
    cmd.AddCommand(f.TeamListCmd())

    return cmd
}
```

### 2. Add Handler Implementations

**File:** `pkg/cli/handlers.go`

Add handler functions:

```go
// runTeamJoin joins or creates a team session in a container
func (f *CommandFactory) runTeamJoin(ctx context.Context, name, sessionName string) error {
    // Validate container exists and is running
    containerName := f.Config.ContainerPrefix + "-" + name
    container, err := f.ContainerMgr.GetContainerInfo(ctx, containerName)
    if err != nil {
        return fmt.Errorf("failed to get container info: %w", err)
    }

    if container.Status != "running" {
        return fmt.Errorf("container '%s' is not running (status: %s)", name, container.Status)
    }

    // Execute SSH with team command
    // Use -t to force TTY allocation for interactive session
    sshCmd := exec.Command("ssh", "-t",
        fmt.Sprintf("%s-%s", f.Config.ContainerPrefix, name),
        fmt.Sprintf("team %s", sessionName))

    sshCmd.Stdin = os.Stdin
    sshCmd.Stdout = os.Stdout
    sshCmd.Stderr = os.Stderr

    return sshCmd.Run()
}

// runTeamList lists active team sessions in a container
func (f *CommandFactory) runTeamList(ctx context.Context, name string) error {
    // Check for --all flag (if implemented)
    // For now, assume single container

    // Validate container exists and is running
    containerName := f.Config.ContainerPrefix + "-" + name
    container, err := f.ContainerMgr.GetContainerInfo(ctx, containerName)
    if err != nil {
        return fmt.Errorf("failed to get container info: %w", err)
    }

    if container.Status != "running" {
        return fmt.Errorf("container '%s' is not running (status: %s)", name, container.Status)
    }

    // Execute team ls command in container
    cmd := []string{"su", "-", f.Config.ContainerUser, "-c", "team ls"}

    // Capture output
    var stdout bytes.Buffer
    // Note: We need to add ExecContainerWithOutput to the interface
    // For now, use a temporary approach with exec.Command

    sshCmd := exec.Command("ssh",
        fmt.Sprintf("%s-%s", f.Config.ContainerPrefix, name),
        "team ls")
    sshCmd.Stdout = &stdout
    sshCmd.Stderr = os.Stderr

    if err := sshCmd.Run(); err != nil {
        return fmt.Errorf("failed to list team sessions: %w", err)
    }

    // Display output
    fmt.Printf("Team sessions in %s:\n%s", containerName, stdout.String())

    return nil
}
```

### 3. Register Commands

**File:** `cmd/l8s/main.go`

Add team command to the appropriate command group:

```go
// In the command registration section, add:
rootCmd.AddCommand(factory.TeamCmd())
```

The command will automatically be part of the "Container Management" group since we set `GroupID: "container-mgmt"` in the command definition.

### 4. Update Container Manager Interface (Optional)

**File:** `pkg/cli/interfaces.go`

If we want to add a proper output-capturing method:

```go
type ContainerManager interface {
    // ... existing methods ...

    // ExecContainerWithOutput executes a command and returns stdout
    ExecContainerWithOutput(ctx context.Context, name string, cmd []string) (string, error)
}
```

Then implement in `pkg/container/manager.go` and `pkg/container/podman_client.go`.

However, this is **optional** - we can use SSH directly for simplicity since we already have SSH configured.

### 5. Add Tests

**File:** `pkg/cli/handlers_test.go`

Add test cases for the new handlers:

```go
func TestRunTeamJoin(t *testing.T) {
    tests := []struct {
        name          string
        containerName string
        sessionName   string
        mockSetup     func(*MockContainerManager)
        wantErr       bool
    }{
        {
            name:          "successful join",
            containerName: "myproject",
            sessionName:   "backend",
            mockSetup: func(m *MockContainerManager) {
                m.EXPECT().
                    GetContainerInfo(gomock.Any(), "dev-myproject").
                    Return(&container.Container{
                        Name:   "dev-myproject",
                        Status: "running",
                    }, nil)
            },
            wantErr: false,
        },
        {
            name:          "container not running",
            containerName: "myproject",
            sessionName:   "backend",
            mockSetup: func(m *MockContainerManager) {
                m.EXPECT().
                    GetContainerInfo(gomock.Any(), "dev-myproject").
                    Return(&container.Container{
                        Name:   "dev-myproject",
                        Status: "stopped",
                    }, nil)
            },
            wantErr: true,
        },
        {
            name:          "container not found",
            containerName: "nonexistent",
            sessionName:   "backend",
            mockSetup: func(m *MockContainerManager) {
                m.EXPECT().
                    GetContainerInfo(gomock.Any(), "dev-nonexistent").
                    Return(nil, fmt.Errorf("container not found"))
            },
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test implementation
        })
    }
}
```

### 6. Update ZSH Completion

**File:** `host-integration/oh-my-zsh/l8s/_l8s`

Add completion for team command:

```zsh
# Add to the main case statement
team)
    local -a subcmds
    subcmds=('join:Join or create a team session' 'list:List active team sessions')
    _describe 'team subcommand' subcmds
    ;;

# Or if using direct team command (not subcommand):
team)
    # Complete container names for first argument
    if (( CURRENT == 3 )); then
        _l8s_complete_containers all
    fi
    # No completion for session name (user types it)
    ;;
```

## SSH Connection Stability Improvements

### Current Problems

Users experience frequent SSH disconnections when:
- Laptop locks or sleeps
- WiFi network changes
- VPN connects/disconnects
- Brief network interruptions

**Current Configuration:**

Client-side (`pkg/ssh/keys.go:85-87`):
```
ControlMaster auto
ControlPath ~/.ssh/control-%%r@%%h:%%p
ControlPersist 10m
```

Server-side (`pkg/container/manager.go:414-415`):
```
ClientAliveInterval 60
ClientAliveCountMax 3
```

**Issues:**
1. No client-side keepalives - client doesn't detect dead connections
2. ControlPersist is only 10 minutes
3. No explicit TCP keepalive setting

### Recommended SSH Config Changes

**File:** `pkg/ssh/keys.go` - Update `GenerateSSHConfigEntry` function

Add these settings to both the CA-enabled and fallback SSH config blocks (lines 78-101):

```go
func GenerateSSHConfigEntry(containerName string, sshPort int, containerUser, prefix, remoteHost string, knownHostsPath string) string {
    // ... existing code ...

    // If knownHostsPath is provided, use strict checking with CA
    if knownHostsPath != "" {
        return fmt.Sprintf(`Host %s
    HostName %s
    Port %d
    User %s
    StrictHostKeyChecking yes
    UserKnownHostsFile %s
    ControlMaster auto
    ControlPath ~/.ssh/control-%%r@%%h:%%p
    ControlPersist 1h
    ServerAliveInterval 30
    ServerAliveCountMax 6
    ConnectTimeout 10
    TCPKeepAlive yes
`, hostAlias, remoteHost, sshPort, containerUser, knownHostsPath)
    }

    // Fallback to insecure mode if no CA configured
    return fmt.Sprintf(`Host %s
    HostName %s
    Port %d
    User %s
    StrictHostKeyChecking no
    UserKnownHostsFile /dev/null
    ControlMaster auto
    ControlPath ~/.ssh/control-%%r@%%h:%%p
    ControlPersist 1h
    ServerAliveInterval 30
    ServerAliveCountMax 6
    ConnectTimeout 10
    TCPKeepAlive yes
`, hostAlias, remoteHost, sshPort, containerUser)
}
```

### SSH Setting Explanations

| Setting | Old Value | New Value | Purpose |
|---------|-----------|-----------|---------|
| `ControlPersist` | 10m | 1h | Keep multiplexed connection alive for 1 hour of idle time |
| `ServerAliveInterval` | (none) | 30 | Client sends keepalive every 30 seconds |
| `ServerAliveCountMax` | (none) | 6 | Tolerate 6 failed keepalives (3 minutes total) |
| `ConnectTimeout` | (none) | 10 | Don't hang forever on initial connection |
| `TCPKeepAlive` | (default) | yes | Explicitly enable TCP-level keepalives |

**Combined effect:**
- Client actively detects dead connections in ~3 minutes (30s × 6)
- Multiplexed connection persists for 1 hour of idle time
- Better survival through brief network hiccups
- Faster detection when connection is truly dead

### What Won't Be Fixed

**True laptop sleep/hibernate** will still kill connections because the network stack is suspended. This is unavoidable with SSH. However, these settings will:
- **Detect** dead connections faster
- **Prevent** disconnections during brief network hiccups
- **Survive** short periods where network stays partially active

For surviving true sleep, users need to:
- Use team sessions (dtach) so work persists across reconnections
- Consider `mosh` for roaming scenarios (future enhancement)
- Use tmux inside containers as additional protection

### Testing SSH Improvements

Manual test cases:
1. **Normal operation**: SSH should work as before
2. **Idle connection**: Leave SSH session idle for 10+ minutes, verify it stays alive
3. **Network change**: Switch WiFi networks while connected (connection will die but reconnect quickly)
4. **Laptop lock**: Lock laptop for 1-2 minutes, unlock and try typing (should detect dead connection)
5. **Brief network loss**: Unplug ethernet for 10 seconds, plug back in (should survive)

### Update Tests

**File:** `pkg/ssh/keys_test.go`

Update the expected output in test cases to include new settings (lines 171-190):

```go
// Both test cases need these lines added:
expected := `Host test-container
    HostName example.com
    Port 2222
    User dev
    StrictHostKeyChecking yes
    UserKnownHostsFile /path/to/known_hosts
    ControlMaster auto
    ControlPath ~/.ssh/control-%r@%h:%p
    ControlPersist 1h
    ServerAliveInterval 30
    ServerAliveCountMax 6
    ConnectTimeout 10
    TCPKeepAlive yes
`
```

## Implementation Phases

### Phase 0: SSH Connection Improvements (Do First)
- [ ] Update `GenerateSSHConfigEntry` with new SSH settings
- [ ] Update `keys_test.go` with new expected output
- [ ] Test SSH config generation
- [ ] Manually verify improved connection stability

### Phase 1: Team Command Implementation (MVP)
- [ ] Add `l8s team <container> <session>` command
- [ ] Add `l8s team list <container>` command
- [ ] Basic error handling (container not found, not running)
- [ ] Register commands in main.go

### Phase 2: Polish
- [ ] Add ZSH completion
- [ ] Add proper test coverage
- [ ] Improve output formatting for `team list`
- [ ] Add command to kill/remove sessions remotely

### Phase 3: Enhancements (Optional)
- [ ] Add `l8s team list --all` for listing across all containers
- [ ] Add `l8s team kill <container> <session>` to terminate sessions
- [ ] Add `--team` flag to `l8s ssh` command for automatic session join
- [ ] Consider adding session metadata (creation time, active users)
- [ ] Investigate `mosh` integration for bulletproof roaming

## User Experience Examples

### Example 1: Quick Team Session Join
```bash
# User wants to join the backend team session
$ l8s team myproject backend

# Lands directly in persistent session
# Session survives if SSH disconnects
# Can detach with Ctrl+\ and reconnect later
```

### Example 2: Discovering Active Sessions
```bash
# See what sessions exist
$ l8s team list myproject
Team sessions in dev-myproject:
  backend
  frontend
  debugging

# Join an existing session
$ l8s team myproject frontend
```

### Example 3: Working with Multiple Containers
```bash
# Different containers, different teams
$ l8s team api-service backend
$ l8s team frontend-app development
$ l8s team infrastructure terraform
```

## Integration with Existing Features

### Claude Code Statusline
The Claude Code statusline (`.claude/statusline.sh`) already shows the active team session:

```bash
# When in a team session, statusline shows:
🤖 [Model] container⚒backend:path [style]
                      ^^^^^^^^^^^
                      Team indicator
```

No changes needed - it automatically detects `$DTACH_SESSION`.

### SSH Configuration
The team commands use the existing SSH configuration in `~/.ssh/config`:
- ControlMaster for connection multiplexing (now with 1h persistence)
- Client-side keepalives (new: ServerAliveInterval/CountMax)
- Existing host aliases (dev-container-name)
- Certificate-based authentication

**Note:** The SSH improvements (Phase 0) enhance the existing configuration but don't change the fundamental architecture.

### Container Dotfiles
The `team` command is embedded in `.zshrc` and will continue to work as-is. The l8s commands simply provide a convenient way to invoke it from the host.

## Security Considerations

### Session Isolation
- Sessions are isolated per container (sockets in `/tmp/`)
- Multiple users can attach to the same session (by design for collaboration)
- Sessions run as the container user (not root)

### Socket Permissions
The dtach sockets in `/tmp/` are created with default permissions. If stricter isolation is needed:
- Could use `$HOME/` instead of `/tmp/` for socket storage
- Could implement per-user socket directories
- Could add authentication/authorization for session access

Currently, anyone who can SSH into the container can attach to any team session. This matches the collaborative intent.

## Alternative Approaches Considered

### Approach 1: Podman Exec Instead of SSH
Could use `podman exec` to run team commands instead of SSH:

**Pros:**
- More direct (no SSH layer)
- Could potentially capture output more easily

**Cons:**
- Interactive sessions work better over SSH (proper TTY, signal handling)
- SSH already configured and working
- Loses connection multiplexing benefits

**Decision:** Stick with SSH for consistency with existing `l8s ssh` command.

### Approach 2: Integrate with `l8s ssh`
Add `--team` flag to existing ssh command:

```bash
l8s ssh myproject --team backend
```

**Pros:**
- Fewer top-level commands
- Natural extension of ssh

**Cons:**
- Less discoverable (hidden behind a flag)
- Can't easily list sessions without ssh'ing in
- Mixes concerns (ssh is generic, team is specific)

**Decision:** Use dedicated `l8s team` commands for better discoverability and functionality.

### Approach 3: Embed in Container Only
Keep team command container-only (current state):

**Pros:**
- Simpler implementation (already done)
- Users have direct control

**Cons:**
- Users must SSH first, then run team
- Can't discover sessions from laptop
- Extra steps for common workflow

**Decision:** Promote to l8s for better UX.

## Testing Strategy

### Manual Testing
1. Create team session from within container
2. List sessions using `l8s team list`
3. Join session using `l8s team`
4. Verify detach/reattach works
5. Test with multiple containers
6. Test error cases (stopped container, nonexistent container)

### Automated Testing
1. Unit tests for handlers (mock container manager)
2. Integration tests (requires running containers)
3. ZSH completion tests (extend existing completion test framework)

### Edge Cases to Test
- Container exists but is stopped
- Container doesn't exist
- Session name with special characters (spaces, quotes)
- Multiple sessions with similar names
- Session exists but is orphaned (no process)

## Documentation Updates

### Files to Update
1. `CLAUDE.md` - Add team command usage
2. `README.md` - Add to command examples
3. Man pages (if we generate them)

### Example Documentation

```markdown
## Team Sessions

L8s supports persistent terminal sessions using dtach. Sessions survive SSH disconnections,
laptop sleep, and network changes.

### Quick Start

```bash
# Join or create a team session
l8s team myproject backend

# List active sessions
l8s team list myproject

# Detach from session: Ctrl+\
# Reattach later: same join command
```

### Use Cases

- **Collaboration**: Multiple developers can attach to the same session
- **Persistence**: Sessions survive network interruptions
- **Long-running tasks**: Keep builds/tests running across disconnections
- **Pair programming**: Share a terminal session in real-time
```

## Dependencies

### Existing Dependencies
- `dtach` - Already installed in container images (in Containerfile)
- SSH - Already configured and working
- ZSH - Team command embedded in .zshrc

### No New Dependencies Required
The implementation uses only existing infrastructure.

## Success Metrics

How to know the implementation is successful:

### SSH Improvements
1. **Connection stability**: Fewer unexpected disconnections during normal use
2. **Detection speed**: Dead connections detected within 3 minutes
3. **Idle persistence**: Connections survive 1+ hour of idle time
4. **Backward compatibility**: No breaking changes to existing SSH behavior

### Team Commands
1. **Functionality**: Users can join/list team sessions from laptop
2. **Reliability**: Sessions survive SSH disconnections (enhanced by SSH improvements)
3. **Discoverability**: `l8s team --help` provides clear guidance
4. **Performance**: Commands execute quickly (<1s for list, instant for join)
5. **Compatibility**: Works with existing SSH configuration and container setup

## Questions for Discussion

1. **Command naming**: `l8s team <container> <session>` vs `l8s team join <container> <session>`?
   - Recommendation: Shorter form for better ergonomics

2. **List all containers**: Implement `--all` flag immediately or wait?
   - Recommendation: Start with single container, add --all later if needed

3. **Additional subcommands**: Should we add `kill`/`remove` for sessions?
   - Recommendation: Start simple, add if users request it

4. **Default session name**: Should `l8s team <container>` (no session name) default to "work"?
   - Recommendation: Require explicit session name for clarity

## References

- Existing team command: `pkg/embed/dotfiles/.zshrc` lines 68-146
- Statusline integration: `pkg/embed/dotfiles/.claude/statusline.sh` lines 78-81
- SSH implementation: `pkg/container/manager.go` SSHIntoContainer method
- dtach documentation: https://github.com/crigler/dtach
