#!/usr/bin/env zsh

# Tests for dynamic container name completion

# Source the test framework
source ./test_framework.sh

# Setup
setup_test_env

# Override the l8s command with our mock
alias l8s=mock_l8s

run_test_suite "Container Name Completion"

# Test 1: Complete container names for info command (not ssh - that's git-native)
completions=$(get_completions "l8s info ")
assert_contains "$completions" "myproject" "Should complete 'myproject' container"
assert_contains "$completions" "webapp" "Should complete 'webapp' container"
assert_contains "$completions" "api" "Should complete 'api' container"
assert_contains "$completions" "cli-tool" "Should complete 'cli-tool' container"

# Test 2: Partial container name completion for info
completions=$(get_completions "l8s info my")
assert_contains "$completions" "myproject" "Should complete 'my' to 'myproject'"
# Note: In test environment, we get all completions. Real zsh filters by prefix.

completions=$(get_completions "l8s info cli")
assert_contains "$completions" "cli-tool" "Should complete 'cli' to 'cli-tool'"

# Test 3: Container completion for different commands
# Note: 'start' only shows stopped containers, others show running or all
completions=$(get_completions "l8s start ")
assert_contains "$completions" "api" "Should complete stopped containers for 'start' command"

# Only 'stop' and 'info' still take container names
for cmd in "stop" "info"; do
    completions=$(get_completions "l8s $cmd ")
    assert_contains "$completions" "myproject" "Should complete containers for '$cmd' command"
done

# Git-native commands don't take container names
for cmd in "remove" "exec"; do
    completions=$(get_completions "l8s $cmd ")
    assert_not_contains "$completions" "myproject" "Should NOT complete containers for '$cmd' command (git-native)"
done

# Test 4: No container completion for commands that don't need it
completions=$(get_completions "l8s init ")
assert_not_contains "$completions" "myproject" "Should not complete containers for 'init' command"

completions=$(get_completions "l8s build ")
assert_not_contains "$completions" "myproject" "Should not complete containers for 'build' command"

completions=$(get_completions "l8s list ")
assert_not_contains "$completions" "myproject" "Should not complete containers for 'list' command"

# Test 5: SSH is git-native, doesn't take container names
completions=$(get_completions "l8s ssh ")
assert_not_contains "$completions" "myproject" "Should NOT complete containers for 'ssh' (git-native)"

# Cleanup
cleanup_test_env

# Print results
print_summary