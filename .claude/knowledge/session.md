# Knowledge Capture Session - 2025-08-28

### [15:47] [architecture] Command Factory Pattern Structure
**Details**: L8s uses a dual command factory pattern:

1. **LazyCommandFactory** (pkg/cli/factory_lazy.go): Creates commands with lazy dependency initialization
   - Dependencies are only initialized when first command is executed via ensureInitialized()
   - Used in main.go for all commands except init
   - Wraps the original CommandFactory for actual execution

2. **CommandFactory** (pkg/cli/factory.go): Contains the actual command implementation
   - Uses dependency injection with interfaces: ContainerManager, GitClient, SSHClient
   - Commands delegate to handler functions (runCreate, runSSH, etc.)
   - Can be instantiated with real or mock dependencies for testing

Key advantages:
- Commands can be registered without requiring config file existence
- Fast CLI startup time as dependencies only load when needed
- Clean separation between command definition and execution
- Testable through dependency injection
**Files**: pkg/cli/factory_lazy.go, pkg/cli/factory.go, cmd/l8s/main.go
---

### [15:47] [pattern] Cobra Command Structure
**Details**: Commands follow a consistent structure in l8s:

1. **Command Definition**: Created in factory methods with cobra.Command
   - Use/Short/Long descriptions
   - Args validation (cobra.ExactArgs(1), cobra.MinimumNArgs(2), etc.)
   - RunE function for execution with error handling
   - Flags defined using cmd.Flags().StringVar(), cmd.Flags().BoolVar()

2. **Handler Pattern**: Execution delegated to handler methods
   - LazyCommandFactory.XXXCmd() creates command and ensures initialization
   - CommandFactory.runXXX() methods contain actual logic
   - Error handling in RunE functions, not in handlers

3. **Flag Handling**: Flags retrieved in handler using cmd.Flags().GetString/GetBool
   - Validation happens in handlers, not command definition
   - Mutually exclusive flags validated explicitly

4. **Context Usage**: All handlers accept context.Context for operations
   - Passed to container manager and other dependencies
   - Enables timeout and cancellation support
**Files**: pkg/cli/factory_lazy.go, pkg/cli/handlers.go
---

### [15:47] [architecture] SSH and Container Operations
**Details**: L8s performs all operations via SSH to remote containers:

1. **SSH Connection**: Uses SSH host configs for container access
   - Format: ssh containerPrefix-name (e.g., "dev-myproject")
   - SSH configs added automatically during container creation
   - Manager.SSHIntoContainer() executes 'ssh <container-name>' with stdio forwarding

2. **Remote Execution**: Two patterns for executing commands in containers
   - Direct SSH: For interactive commands (SSHIntoContainer)
   - Podman exec: For automated commands (ExecContainer, ExecContainerWithInput)

3. **File Transfer**: Currently uses exec commands for file operations
   - ExecContainerWithInput for writing files with stdin
   - mkdir, chmod, chown commands for permission management
   - No direct scp/rsync usage observed in codebase

4. **Container Communication**: All via SSH tunnel to remote Podman socket
   - PodmanClient connects to remote socket over SSH
   - Container operations happen on remote server, not locally
**Files**: pkg/container/manager.go, pkg/container/podman_client.go, pkg/cli/handlers.go
---

### [15:48] [architecture] L8s dual factory pattern for commands
**Details**: L8s uses a dual factory pattern for CLI commands:
1. LazyCommandFactory: Enables fast CLI startup by delaying dependency initialization until command execution via ensureInitialized()
2. CommandFactory: Contains actual command implementations with dependency injection

Commands are registered in cmd/l8s/main.go and follow cobra command patterns. Each command:
- Defined in LazyCommandFactory with cobra setup
- Delegates to CommandFactory.runXxx() for actual implementation
- Uses consistent error handling with fmt.Errorf() and context
- Outputs with color package for formatting

SSH operations use ContainerManager interface methods like ExecContainer and ExecContainerWithInput for remote execution and file transfer.
**Files**: pkg/cli/factory_lazy.go, pkg/cli/factory.go, cmd/l8s/main.go
---

### [15:59] [feature] L8s paste command implementation
**Details**: Implemented a paste command that transfers clipboard content from macOS to remote containers.

Key implementation details:
1. Command: `l8s paste <container> [name]` - transfers clipboard to /tmp/claude-clipboard/
2. Platform: macOS-only initially (uses osascript for images, pbpaste for text)
3. File naming: Default files (clipboard.png/txt) are replaced, custom names (clipboard-<name>.png/txt) are preserved
4. Architecture: Follows L8s dual factory pattern with LazyCommandFactory and CommandFactory
5. Transfer method: Uses ExecContainerWithInput to pipe content via tee command
6. Claude integration: Created /paste slash command in .claude/commands/paste.md that detects and prompts analysis of pasted files

Technical components:
- pkg/cli/clipboard.go: Clipboard detection and extraction utilities
- pkg/cli/handlers.go: runPaste handler implementation
- pkg/cli/factory_lazy.go: PasteCmd command definition
- pkg/embed/dotfiles/.claude/commands/paste.md: Claude slash command
- Added ExecContainerWithInput to ContainerManager interface and implementation
**Files**: pkg/cli/clipboard.go, pkg/cli/handlers.go, pkg/cli/factory_lazy.go, pkg/embed/dotfiles/.claude/commands/paste.md, pkg/container/manager.go, pkg/cli/interfaces.go
---

### [16:01] [gotcha] Testing unexported methods in L8s
**Details**: When adding new command handlers in L8s, the handler methods (like runPaste) are unexported (lowercase) and cannot be directly tested from test files.

Issue encountered:
- Handler methods like runPaste() are defined as unexported methods on CommandFactory
- Test files cannot directly call these methods: "cannot directly test unexported runPaste method"
- This affects both unit tests and integration tests

Solution:
- Unit tests should focus on testing the command structure and exported methods
- Integration tests need to use the actual CLI binary or exported command interfaces
- For handlers that need testing, consider either:
  1. Making them exported (RunPaste instead of runPaste) if they need direct testing
  2. Testing through the command's RunE function execution
  3. Using the actual l8s binary in integration tests

This is a deliberate design choice in L8s to keep the API surface clean - handlers are internal implementation details accessed only through the Cobra command structure.
**Files**: pkg/cli/handlers.go, pkg/cli/paste_test.go, test/integration/paste_test.go
---

### [16:01] [pattern] Adding methods to L8s ContainerManager interface
**Details**: When adding new container operations to L8s, you must update multiple locations to maintain interface consistency:

Required updates for new ContainerManager methods:
1. pkg/cli/interfaces.go - Add method to ContainerManager interface
2. pkg/container/manager.go - Implement method on Manager struct (usually delegates to client)
3. pkg/container/podman_client.go - Implement actual Podman operation on RealPodmanClient
4. pkg/container/mock_client.go - Add mock implementation for MockPodmanClient
5. pkg/cli/handlers_test.go - Add to MockContainerManagerWithGit for handler tests
6. pkg/cli/factory_lazy_test.go - Add to MockContainerManager for factory tests

Example for ExecContainerWithInput:
- Interface signature: ExecContainerWithInput(ctx context.Context, name string, cmd []string, input []byte) error
- Manager delegates to client with container prefix handling
- Client implementation converts []byte to string for actual Podman exec
- All mock implementations must match the interface signature exactly

This pattern ensures proper separation between the CLI layer and container implementation while maintaining testability.
**Files**: pkg/cli/interfaces.go, pkg/container/manager.go, pkg/cli/handlers_test.go, pkg/cli/factory_lazy_test.go
---

### [16:02] [testing] L8s make ci validation requirements
**Details**: The 'make ci' command runs comprehensive checks that must all pass:

1. Clean and dependency download
2. Build dependency checks  
3. Linting (skipped if golangci-lint not installed)
4. Go unit tests - ALL packages must compile and pass
5. ZSH plugin tests - completion and integration tests
6. Neovim plugin updates (updates lazy-lock.json)

Common issues that break CI:
- Missing interface methods in mocks (compilation failure)
- Unused imports (compilation failure)  
- Undefined methods/variables (compilation failure)
- Test failures

The CI uses build tags to exclude optional dependencies:
- exclude_graphdriver_btrfs
- exclude_graphdriver_devicemapper

When CI fails, fix compilation errors first, then test failures. The error messages clearly indicate which package and line has issues.
**Files**: Makefile
---

