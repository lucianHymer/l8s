#!/usr/bin/env zsh

# Tests for subcommand completion (e.g., l8s remote add/remove)

# Source the test framework
source ./test_framework.sh

# Setup
setup_test_env

# Override the l8s command with our mock
alias l8s=mock_l8s

run_test_suite "Subcommand Completion"

# Test 1: Complete remote subcommands
completions=$(get_completions "l8s remote ")
assert_contains "$completions" "add" "Should complete 'add' subcommand for remote"
assert_contains "$completions" "remove" "Should complete 'remove' subcommand for remote"

# Test 2: Partial subcommand completion
completions=$(get_completions "l8s remote a")
assert_contains "$completions" "add" "Should complete 'a' to 'add'"
# Note: In test environment, we get all completions. Real zsh filters by prefix.

completions=$(get_completions "l8s remote r")
assert_contains "$completions" "remove" "Should complete 'r' to 'remove'"
# Note: In test environment, we get all completions. Real zsh filters by prefix.

# Test 3: Container completion after remote subcommands
completions=$(get_completions "l8s remote add ")
assert_contains "$completions" "myproject" "Should complete containers after 'remote add'"
assert_contains "$completions" "webapp" "Should complete containers after 'remote add'"

completions=$(get_completions "l8s remote remove ")
assert_contains "$completions" "myproject" "Should complete containers after 'remote remove'"
assert_contains "$completions" "webapp" "Should complete containers after 'remote remove'"

# Test 4: No further completion after container name
completions=$(get_completions "l8s remote add myproject ")
assert_equals "$completions" "" "Should not complete after container name"

# Test 5: Create command special arguments
completions=$(get_completions "l8s create myproject ")
assert_equals "$completions" "" "Should not auto-complete git URLs (user must type)"

# After git URL, could suggest common branches (but this is optional)
# We'll leave this unimplemented for now as it's complex

# Cleanup
cleanup_test_env

# Print results
print_summary