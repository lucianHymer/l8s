#!/usr/bin/env zsh

# Test framework for zsh completion testing
# Sets up environment and provides assertion functions

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
NC='\033[0m' # No Color

# Test counters
TESTS_RUN=0
TESTS_PASSED=0
TESTS_FAILED=0

# Setup test environment
setup_test_env() {
    # Create temporary directory for test
    TEST_DIR=$(mktemp -d)
    export TEST_DIR
    export ZDOTDIR=$TEST_DIR
    
    # Initialize completion system
    autoload -U compinit
    compinit -d $TEST_DIR/.zcompdump
    
    # Load the completion function directly
    source ../../pkg/embed/host-integration/oh-my-zsh/l8s/_l8s
}

# Cleanup test environment
cleanup_test_env() {
    rm -rf $TEST_DIR
}

# Capture completions for a given command line
get_completions() {
    local cmd="$1"
    local cursor_pos=${2:-${#cmd}}
    
    # Reset completion variables
    unset compadd_output
    compadd_output=()
    
    # Override compadd to capture completions
    compadd() {
        local -a args
        args=()
        # Skip options and their arguments
        while [[ $# -gt 0 ]]; do
            case "$1" in
                -a)
                    # -a means use array variable
                    shift
                    local array_name="$1"
                    eval "args=(\"\${${array_name}[@]}\")"
                    shift
                    ;;
                -d)
                    # -d means descriptions, skip
                    shift 2
                    ;;
                --)
                    shift
                    args+=("$@")
                    break
                    ;;
                -*)
                    shift
                    ;;
                *)
                    args+=("$1")
                    shift
                    ;;
            esac
        done
        compadd_output+=("${args[@]}")
    }
    
    # Set up completion context variables needed by _arguments
    local curcontext="test:test:test"
    local -a words
    words=(${(z)cmd})
    local CURRENT=$#words
    if [[ $cmd == *" " ]]; then
        words+=("")
        CURRENT=$((CURRENT + 1))
    fi
    
    # Call the completion function
    _l8s
    
    # No need to restore compadd - it's local to this function
    
    # Return captured completions
    echo "${compadd_output[@]}"
}

# Test assertion functions
assert_contains() {
    local haystack="$1"
    local needle="$2"
    local test_name="$3"
    
    TESTS_RUN=$((TESTS_RUN + 1))
    
    if [[ "$haystack" == *"$needle"* ]]; then
        TESTS_PASSED=$((TESTS_PASSED + 1))
        echo -e "${GREEN}✓${NC} $test_name"
        return 0
    else
        TESTS_FAILED=$((TESTS_FAILED + 1))
        echo -e "${RED}✗${NC} $test_name"
        echo -e "  Expected to contain: $needle"
        echo -e "  Actual: $haystack"
        return 1
    fi
}

assert_not_contains() {
    local haystack="$1"
    local needle="$2"
    local test_name="$3"
    
    TESTS_RUN=$((TESTS_RUN + 1))
    
    if [[ "$haystack" != *"$needle"* ]]; then
        TESTS_PASSED=$((TESTS_PASSED + 1))
        echo -e "${GREEN}✓${NC} $test_name"
        return 0
    else
        TESTS_FAILED=$((TESTS_FAILED + 1))
        echo -e "${RED}✗${NC} $test_name"
        echo -e "  Expected NOT to contain: $needle"
        echo -e "  Actual: $haystack"
        return 1
    fi
}

assert_equals() {
    local actual="$1"
    local expected="$2"
    local test_name="$3"
    
    TESTS_RUN=$((TESTS_RUN + 1))
    
    if [[ "$actual" == "$expected" ]]; then
        TESTS_PASSED=$((TESTS_PASSED + 1))
        echo -e "${GREEN}✓${NC} $test_name"
        return 0
    else
        TESTS_FAILED=$((TESTS_FAILED + 1))
        echo -e "${RED}✗${NC} $test_name"
        echo -e "  Expected: $expected"
        echo -e "  Actual: $actual"
        return 1
    fi
}

# Run a test suite
run_test_suite() {
    local suite_name="$1"
    echo -e "\n${YELLOW}Running test suite: $suite_name${NC}"
}

# Print test summary
print_summary() {
    echo -e "\n${YELLOW}Test Summary:${NC}"
    echo -e "Tests run: $TESTS_RUN"
    echo -e "${GREEN}Passed: $TESTS_PASSED${NC}"
    echo -e "${RED}Failed: $TESTS_FAILED${NC}"
    
    if [[ $TESTS_FAILED -eq 0 ]]; then
        echo -e "\n${GREEN}All tests passed!${NC}"
        return 0
    else
        echo -e "\n${RED}Some tests failed!${NC}"
        return 1
    fi
}

# Mock l8s command for testing
mock_l8s() {
    case "$1" in
        "list"|"ls")
            # Mock output for list command
            echo "NAME              STATUS      SSH_PORT    GIT_REMOTE                                     CREATED"
            echo "dev-myproject     Running     2200        git@github.com:user/myproject.git            2024-01-15"
            echo "dev-webapp        Running     2201        git@github.com:user/webapp.git               2024-01-14"
            echo "dev-api           Stopped     2202        git@github.com:user/api.git                 2024-01-13"
            echo "dev-cli-tool      Running     2203        git@github.com:user/cli-tool.git            2024-01-12"
            ;;
    esac
}

# No need to export functions in zsh - they're already available in subshells