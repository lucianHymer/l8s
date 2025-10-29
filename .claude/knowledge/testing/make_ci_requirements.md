# Make CI Validation Requirements

The `make ci` command runs comprehensive checks that must all pass for code to be considered ready.

## CI Pipeline Steps

1. **Clean and dependency download**
   - Removes build artifacts
   - Downloads Go dependencies

2. **Build dependency checks**
   - Verifies all packages compile
   - Checks for missing dependencies

3. **Linting**
   - Uses golangci-lint if available
   - Skipped if not installed (with warning)

4. **Go unit tests**
   - ALL packages must compile and pass
   - Runs with coverage reporting

5. **ZSH plugin tests**
   - Completion tests
   - Integration tests

6. **Neovim plugin updates**
   - Updates lazy-lock.json
   - Ensures plugin compatibility

## Common CI Failures

### Compilation Failures
- **Missing interface methods in mocks**
  - Solution: Update all mocks when interface changes
- **Unused imports**
  - Solution: Remove or use the import
- **Undefined methods/variables**
  - Solution: Implement missing methods or define variables

### Test Failures
- Fix compilation errors first
- Then address test logic issues
- Error messages clearly indicate package and line

## Build Tags

CI uses build tags to exclude optional dependencies:
- `exclude_graphdriver_btrfs`
- `exclude_graphdriver_devicemapper`

These reduce binary size and compilation requirements.

### Running Tests Without System Dependencies

When system dependencies like gpgme or btrfs are missing, tests can still run using build tags. The Makefile already handles this:
- `make test` and `make test-go` automatically use `-tags exclude_graphdriver_btrfs,exclude_graphdriver_devicemapper`
- This allows testing even without full Podman dependencies installed locally
- No special configuration needed - the Makefile handles it automatically

## Troubleshooting

1. **Fix compilation errors first** - Can't run tests if code doesn't compile
2. **Check all mock implementations** - Interface changes require mock updates
3. **Run `make fmt`** - Format code before committing
4. **Run `make lint`** - Check for style issues
5. **Run `make test`** - Verify tests pass locally

## Related Files
- `Makefile` - CI configuration and commands