# ContainerManager Interface Pattern

When adding new container operations to L8s, multiple locations must be updated to maintain interface consistency.

## Required Updates for New Methods

1. **Interface Definition**
   - File: `pkg/cli/interfaces.go`
   - Add method to ContainerManager interface

2. **Manager Implementation**
   - File: `pkg/container/manager.go`
   - Implement method on Manager struct
   - Usually delegates to client with container prefix handling

3. **Podman Client**
   - File: `pkg/container/podman_client.go`
   - Implement actual Podman operation on RealPodmanClient

4. **Mock Client**
   - File: `pkg/container/mock_client.go`
   - Add mock implementation for MockPodmanClient

5. **Test Mocks**
   - File: `pkg/cli/handlers_test.go`
   - Add to MockContainerManagerWithGit for handler tests
   - File: `pkg/cli/factory_lazy_test.go`
   - Add to MockContainerManager for factory tests

## Example: ExecContainerWithInput

### Interface Signature
```go
ExecContainerWithInput(ctx context.Context, name string, cmd []string, input []byte) error
```

### Implementation Pattern
- Manager delegates to client with container prefix handling
- Client implementation converts []byte to string for actual Podman exec
- All mock implementations must match interface signature exactly

## Why This Pattern?

This pattern ensures:
- Proper separation between CLI layer and container implementation
- Complete testability through dependency injection
- Consistent behavior across real and mock implementations
- Clear contract definition through interfaces

## Related Files
- `pkg/cli/interfaces.go` - Interface definitions
- `pkg/container/manager.go` - Manager implementation
- `pkg/cli/handlers_test.go` - Handler test mocks
- `pkg/cli/factory_lazy_test.go` - Factory test mocks