# ZSH Completion Test Framework

L8s includes a sophisticated custom ZSH test framework for validating completion behavior.

## Framework Overview

**Location**: `host-integration/oh-my-zsh/l8s/tests/`

The framework provides comprehensive testing for ZSH completions with mock commands and assertion utilities.

## Core Components

### test_framework.sh
The main test framework that provides:
- Mock `l8s` command that returns predictable test data
- Assertion functions for validating completions
- Color-coded output with pass/fail indicators
- Completion capture by overriding the `compadd` builtin

### Test Organization

91 total tests across 4 test files:

1. **test_basic_completion.sh** (27 tests)
   - Command completion
   - Alias completion
   - Basic functionality

2. **test_container_completion.sh** (15 tests)
   - Container name completion
   - Partial matching
   - Name filtering

3. **test_context_filtering.sh** (34 tests)
   - State-aware container filtering
   - Command-specific filtering logic
   - Edge cases and error handling

4. **test_subcommand_completion.sh** (15 tests)
   - Subcommand handling
   - Multi-level command completion

## Key Features

### Mock L8s Command
The framework includes a mock `l8s` that:
- Returns fake container data for predictable testing
- Simulates different container states
- Provides consistent test environment

### Completion Capture
The framework captures completions by:
- Overriding the ZSH `compadd` builtin
- Storing completion results for verification
- Allowing precise testing of completion output

### Assertion Functions
Built-in assertions for:
- Checking if specific completions are present
- Verifying completion counts
- Validating filtering behavior

## Test Execution

Tests can be run individually or as a suite:
- Each test file is self-contained
- Framework handles setup and teardown
- Results are aggregated and reported

## Benefits

1. **Predictability**: Mock data ensures consistent test results
2. **Coverage**: Tests all aspects of completion behavior
3. **Maintainability**: Clear test organization and naming
4. **Debugging**: Color-coded output makes failures easy to spot

## Related Files
- `host-integration/oh-my-zsh/l8s/tests/test_framework.sh` - Core framework
- `host-integration/oh-my-zsh/l8s/tests/test_*.sh` - Individual test files
- `host-integration/oh-my-zsh/l8s/_l8s` - Completion being tested