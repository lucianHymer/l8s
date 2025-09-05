# Git-Native Test Updates

After implementing the git-native architecture, tests require updates to match the new behavior.

## Changes Required

### Container Naming
- Old format: User-specified names like "test-container"
- New format: Deterministic `dev-<repo>-<hash>` format
- Example: `dev-project-e3af8a`

### Command Arguments
- Commands no longer take container name arguments
- Container names are derived from the current worktree
- Tests must be updated to not pass explicit container names

### Mock Updates Needed

1. **Test mocks must expect new naming format**
   - Update MockContainerManager expectations
   - Update MockContainerManagerWithGit expectations

2. **Remove container name arguments from test commands**
   - Commands now derive container from git context
   - Only global commands (init, list, build) work without git context

3. **Add git repository context to tests**
   - Tests for git-aware commands must simulate being in a git repo
   - Use mock git client to provide repository information

## Affected Test Files
- `pkg/cli/handlers_test.go` - Handler unit tests
- `pkg/cli/factory_lazy_test.go` - Factory tests
- Integration tests that create/manage containers

## Testing Strategy

1. Update mocks to expect deterministic container names
2. Provide git context in test setup
3. Remove explicit container name arguments from command calls
4. Verify commands fail appropriately when not in git repository

## Related Files
- `pkg/cli/handlers_test.go` - Unit tests for handlers
- `pkg/cli/factory_lazy_test.go` - Factory pattern tests