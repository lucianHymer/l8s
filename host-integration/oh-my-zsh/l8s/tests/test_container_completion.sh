#!/usr/bin/env zsh

# Tests for dynamic container name completion

# Source the test framework
source ./test_framework.sh

# Setup
setup_test_env

# Override the l8s command with our mock
alias l8s=mock_l8s

run_test_suite "Container Name Completion"

# Test 1: Complete container names for ssh command
completions=$(get_completions "l8s ssh ")
assert_contains "$completions" "myproject" "Should complete 'myproject' container"
assert_contains "$completions" "webapp" "Should complete 'webapp' container"
assert_contains "$completions" "api" "Should complete 'api' container"
assert_contains "$completions" "cli-tool" "Should complete 'cli-tool' container"

# Test 2: Partial container name completion
completions=$(get_completions "l8s ssh my")
assert_contains "$completions" "myproject" "Should complete 'my' to 'myproject'"
# Note: In test environment, we get all completions. Real zsh filters by prefix.

completions=$(get_completions "l8s ssh cli")
assert_contains "$completions" "cli-tool" "Should complete 'cli' to 'cli-tool'"

# Test 3: Container completion for different commands
# Note: 'start' only shows stopped containers, others show running or all
completions=$(get_completions "l8s start ")
assert_contains "$completions" "api" "Should complete stopped containers for 'start' command"

for cmd in "stop" "remove" "info" "exec"; do
    completions=$(get_completions "l8s $cmd ")
    assert_contains "$completions" "myproject" "Should complete containers for '$cmd' command"
done

# Test 4: No container completion for commands that don't need it
completions=$(get_completions "l8s init ")
assert_not_contains "$completions" "myproject" "Should not complete containers for 'init' command"

completions=$(get_completions "l8s build ")
assert_not_contains "$completions" "myproject" "Should not complete containers for 'build' command"

completions=$(get_completions "l8s list ")
assert_not_contains "$completions" "myproject" "Should not complete containers for 'list' command"

# Test 5: Complete with dev- prefix stripped
completions=$(get_completions "l8s ssh dev-")
assert_contains "$completions" "myproject" "Should still complete when user types 'dev-' prefix"

# Cleanup
cleanup_test_env

# Print results
print_summary