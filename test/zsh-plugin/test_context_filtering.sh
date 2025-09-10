#!/usr/bin/env zsh

# Tests for context-aware filtering (running vs stopped containers)

# Source the test framework
source ./test_framework.sh

# Setup
setup_test_env

# Mock l8s with different container states
mock_l8s_filtered() {
    case "$1" in
        "list"|"ls")
            # Always return all containers - the completion function does the filtering
            echo "NAME              STATUS      SSH_PORT    GIT_REMOTE                                     CREATED"
            echo "dev-myproject     Running     2200        git@github.com:user/myproject.git            2024-01-15"
            echo "dev-webapp        Running     2201        git@github.com:user/webapp.git               2024-01-14"
            echo "dev-api           Stopped     2202        git@github.com:user/api.git                 2024-01-13"
            echo "dev-cli-tool      Running     2203        git@github.com:user/cli-tool.git            2024-01-12"
            ;;
    esac
}

# Override with our filtered mock
alias l8s=mock_l8s_filtered

run_test_suite "Context-Aware Container Filtering"

# Test 1: Stop command should only show running containers
completions=$(get_completions "l8s stop ")
assert_contains "$completions" "myproject" "Should show running container 'myproject' for stop"
assert_contains "$completions" "webapp" "Should show running container 'webapp' for stop"
assert_contains "$completions" "cli-tool" "Should show running container 'cli-tool' for stop"
assert_not_contains "$completions" "api" "Should NOT show stopped container 'api' for stop"

# Test 2: Start command should only show stopped containers
completions=$(get_completions "l8s start ")
assert_contains "$completions" "api" "Should show stopped container 'api' for start"
assert_not_contains "$completions" "myproject" "Should NOT show running container 'myproject' for start"
assert_not_contains "$completions" "webapp" "Should NOT show running container 'webapp' for start"

# Test 3: Exec command is git-native - doesn't take container names
completions=$(get_completions "l8s exec ")
assert_not_contains "$completions" "myproject" "Should NOT show containers for exec (git-native)"
assert_not_contains "$completions" "webapp" "Should NOT show containers for exec (git-native)"
assert_not_contains "$completions" "api" "Should NOT show containers for exec (git-native)"

# Test 3b: Paste command is now git-native (doesn't take container arg)
completions=$(get_completions "l8s paste ")
assert_not_contains "$completions" "myproject" "Should NOT show containers for paste (git-native)"
assert_not_contains "$completions" "webapp" "Should NOT show containers for paste (git-native)"
assert_not_contains "$completions" "api" "Should NOT show containers for paste (git-native)"

# Test 4: Only info shows all containers (remove, rm, rebuild, ssh are git-native)
completions=$(get_completions "l8s info ")
assert_contains "$completions" "myproject" "Should show all containers for 'info' command"
assert_contains "$completions" "webapp" "Should show all containers for 'info' command"
assert_contains "$completions" "api" "Should show all containers for 'info' command"
assert_contains "$completions" "cli-tool" "Should show all containers for 'info' command"

# Test 4b: Git-native commands don't take container names
for cmd in "remove" "rm" "rebuild" "ssh"; do
    completions=$(get_completions "l8s $cmd ")
    assert_not_contains "$completions" "myproject" "Should NOT show containers for '$cmd' (git-native)"
    assert_not_contains "$completions" "webapp" "Should NOT show containers for '$cmd' (git-native)"
    assert_not_contains "$completions" "api" "Should NOT show containers for '$cmd' (git-native)"
    assert_not_contains "$completions" "cli-tool" "Should NOT show containers for '$cmd' (git-native)"
done

# Test 5: Exec command doesn't take container names anymore (git-native)
# This test is no longer applicable
assert_not_contains "$(get_completions 'l8s exec myproject ')" "webapp" "Should not suggest other containers after selecting one"

# Cleanup
cleanup_test_env

# Print results
print_summary