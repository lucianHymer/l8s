# Gotcha: Testing Unexported Methods in L8s

**Date**: 2025-08-28

## Issue

When adding new command handlers in L8s, the handler methods (like `runPaste`) are unexported (lowercase) and cannot be directly tested from test files.

### Problems Encountered
- Handler methods like `runPaste()` are defined as unexported methods on CommandFactory
- Test files cannot directly call these methods: "cannot directly test unexported runPaste method"
- This affects both unit tests and integration tests

## Solution Approaches

### Unit Tests
- Focus on testing command structure and exported methods
- Test through the command's RunE function execution
- Use the factory pattern to test with mock dependencies

### Integration Tests
- Use the actual CLI binary for end-to-end testing
- Test through exported command interfaces
- Leverage the mock container manager for isolated testing

### Design Rationale

This is a **deliberate design choice** in L8s:
- Keeps the API surface clean
- Handlers are internal implementation details
- Access only through Cobra command structure
- Enforces proper testing through public interfaces

## Options for Testing New Handlers

1. **Make handler exported** (RunPaste instead of runPaste)
   - Only if direct testing is absolutely necessary
   - Generally discouraged

2. **Test through RunE function**
   - Preferred approach
   - Tests the actual command execution path

3. **Use integration tests**
   - Test with actual l8s binary
   - Most realistic testing scenario

## Related Files
- `pkg/cli/handlers.go` - Handler implementations
- `pkg/cli/paste_test.go` - Unit test attempts
- `test/integration/paste_test.go` - Integration test approaches