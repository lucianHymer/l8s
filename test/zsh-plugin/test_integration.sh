#!/usr/bin/env zsh

# Integration test for l8s zsh plugin
# Tests the plugin in a more realistic environment

echo "🔧 Running Integration Test"
echo "=========================="

# Create a temporary home directory
TEST_HOME=$(mktemp -d)
export HOME=$TEST_HOME
export ZDOTDIR=$TEST_HOME

# Create minimal zsh config
cat > $TEST_HOME/.zshrc << 'EOF'
# Load completion system
autoload -U compinit && compinit

# Add plugin to fpath and load it
fpath=($PWD/../../pkg/embed/host-integration/oh-my-zsh/l8s $fpath)
source ../../pkg/embed/host-integration/oh-my-zsh/l8s/l8s.plugin.zsh
EOF

# Test that the plugin loads without errors
echo -n "Testing plugin load... "
if zsh -c "source $TEST_HOME/.zshrc" 2>/dev/null; then
    echo "✓ Plugin loads successfully"
else
    echo "✗ Plugin failed to load"
    exit 1
fi

# Test that completion function is available
echo -n "Testing completion function... "
if zsh -c "source $TEST_HOME/.zshrc && type _l8s > /dev/null" 2>/dev/null; then
    echo "✓ Completion function is available"
else
    echo "✗ Completion function not found"
    exit 1
fi

# Test basic completion in interactive-like environment
echo -n "Testing basic completion... "
cat > $TEST_HOME/test_completion.zsh << 'EOF'
source $HOME/.zshrc

# Capture completions
capture_completions() {
    local -a matches
    compadd() {
        while [[ $1 == -* ]]; do shift; done
        matches+=("$@")
    }
    
    # Set up completion context
    local -a words
    words=(l8s "")
    local CURRENT=2
    _l8s
    
    echo "${matches[@]}"
}

result=$(capture_completions)
if [[ "$result" == *"init"* ]] && [[ "$result" == *"create"* ]]; then
    exit 0
else
    exit 1
fi
EOF

if zsh $TEST_HOME/test_completion.zsh 2>/dev/null; then
    echo "✓ Basic completion works"
else
    echo "✗ Basic completion failed"
    exit 1
fi

# Clean up
rm -rf $TEST_HOME

echo -e "\n✅ Integration test passed!"