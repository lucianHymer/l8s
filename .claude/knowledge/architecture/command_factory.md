# Command Factory Architecture

L8s uses a sophisticated dual factory pattern for CLI command management that enables fast startup and clean dependency injection.

## Dual Factory Pattern

### LazyCommandFactory
**Location**: `pkg/cli/factory_lazy.go`

Creates commands with lazy dependency initialization:
- Dependencies only initialized when first command executes via `ensureInitialized()`
- Used in `main.go` for all commands except init
- Wraps the original CommandFactory for actual execution
- Enables fast CLI startup as dependencies only load when needed

### CommandFactory
**Location**: `pkg/cli/factory.go`

Contains actual command implementations:
- Uses dependency injection with interfaces: ContainerManager, GitClient, SSHClient
- Commands delegate to handler functions (`runCreate`, `runSSH`, etc.)
- Can be instantiated with real or mock dependencies for testing
- Clean separation between command definition and execution

## Key Advantages

1. **Performance**: Commands can be registered without requiring config file existence
2. **Fast Startup**: Dependencies only load when actually needed
3. **Testability**: Clean dependency injection allows easy mocking
4. **Separation of Concerns**: Clear boundary between command definition and execution logic

## Related Files
- `pkg/cli/factory_lazy.go` - Lazy initialization wrapper
- `pkg/cli/factory.go` - Core command factory
- `cmd/l8s/main.go` - Command registration
- `pkg/cli/handlers.go` - Handler implementations