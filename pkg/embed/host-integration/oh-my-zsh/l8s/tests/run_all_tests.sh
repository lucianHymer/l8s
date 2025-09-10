#!/usr/bin/env zsh

# Main test runner for l8s zsh plugin

echo "🧪 Running l8s ZSH Plugin Tests"
echo "==============================="

# Find all test files
test_files=(test_*.sh)

# Track overall results
total_tests=0
total_passed=0
total_failed=0

# Run each test file
for test_file in $test_files; do
    if [[ "$test_file" == "test_framework.sh" ]]; then
        continue  # Skip the framework file
    fi
    
    echo -e "\n📋 Running $test_file"
    echo "----------------------------"
    
    # Run test and capture output
    output=$(zsh $test_file 2>&1)
    exit_code=$?
    
    # Display output
    echo "$output"
    
    # Extract test counts from output
    if [[ $output =~ "Tests run: ([0-9]+)" ]]; then
        tests_run=${match[1]}
        total_tests=$((total_tests + tests_run))
    fi
    
    if [[ $output =~ "Passed: ([0-9]+)" ]]; then
        passed=${match[1]}
        total_passed=$((total_passed + passed))
    fi
    
    if [[ $output =~ "Failed: ([0-9]+)" ]]; then
        failed=${match[1]}
        total_failed=$((total_failed + failed))
    fi
done

# Print overall summary
echo -e "\n==============================="
echo "📊 Overall Test Summary"
echo "==============================="
echo "Total tests run: $total_tests"
echo -e "\033[0;32mTotal passed: $total_passed\033[0m"
echo -e "\033[0;31mTotal failed: $total_failed\033[0m"

if [[ $total_failed -eq 0 ]]; then
    echo -e "\n\033[0;32m✅ All tests passed!\033[0m"
    exit 0
else
    echo -e "\n\033[0;31m❌ Some tests failed!\033[0m"
    exit 1
fi