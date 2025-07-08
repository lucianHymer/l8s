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
            case "$2" in
                "--running")
                    echo "NAME              STATUS      SSH_PORT    GIT_REMOTE                                     CREATED"
                    echo "dev-myproject     Running     2200        git@github.com:user/myproject.git            2024-01-15"
                    echo "dev-webapp        Running     2201        git@github.com:user/webapp.git               2024-01-14"
                    echo "dev-cli-tool      Running     2203        git@github.com:user/cli-tool.git            2024-01-12"
                    ;;
                "--stopped")
                    echo "NAME              STATUS      SSH_PORT    GIT_REMOTE                                     CREATED"
                    echo "dev-api           Stopped     2202        git@github.com:user/api.git                 2024-01-13"
                    ;;
                *)
                    # Default: all containers
                    echo "NAME              STATUS      SSH_PORT    GIT_REMOTE                                     CREATED"
                    echo "dev-myproject     Running     2200        git@github.com:user/myproject.git            2024-01-15"
                    echo "dev-webapp        Running     2201        git@github.com:user/webapp.git               2024-01-14"
                    echo "dev-api           Stopped     2202        git@github.com:user/api.git                 2024-01-13"
                    echo "dev-cli-tool      Running     2203        git@github.com:user/cli-tool.git            2024-01-12"
                    ;;
            esac
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

# Test 3: Exec command should only show running containers
completions=$(get_completions "l8s exec ")
assert_contains "$completions" "myproject" "Should show running container 'myproject' for exec"
assert_contains "$completions" "webapp" "Should show running container 'webapp' for exec"
assert_not_contains "$completions" "api" "Should NOT show stopped container 'api' for exec"

# Test 4: Remove, info, and ssh should show all containers
for cmd in "remove" "info" "ssh"; do
    completions=$(get_completions "l8s $cmd ")
    assert_contains "$completions" "myproject" "Should show all containers for '$cmd' command"
    assert_contains "$completions" "webapp" "Should show all containers for '$cmd' command"
    assert_contains "$completions" "api" "Should show all containers for '$cmd' command"
    assert_contains "$completions" "cli-tool" "Should show all containers for '$cmd' command"
done

# Test 5: Exec command should suggest commands after container name
completions=$(get_completions "l8s exec myproject ")
# We'll implement common command suggestions later in the actual plugin
# For now, we just test that it doesn't try to complete more containers
assert_not_contains "$completions" "webapp" "Should not suggest other containers after selecting one"

# Cleanup
cleanup_test_env

# Print results
print_summary