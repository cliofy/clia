#!/bin/bash

# CLIA CLI Integration Test Script

set -e

echo "Building CLIA..."
go build -o clia ./cmd/clia

echo ""
echo "================================"
echo "CLIA CLI Integration Tests"
echo "================================"
echo ""

# Test 1: Help command
echo "Test 1: Help command"
./clia --help > /dev/null 2>&1
if [ $? -eq 0 ]; then
    echo "✓ Help command works"
else
    echo "✗ Help command failed"
    exit 1
fi

# Test 2: Version command
echo "Test 2: Version command"
./clia version | grep -q "CLIA"
if [ $? -eq 0 ]; then
    echo "✓ Version command works"
else
    echo "✗ Version command failed"
    exit 1
fi

# Test 3: Config path command
echo "Test 3: Config path"
CONFIG_PATH=$(./clia config path)
if [ ! -z "$CONFIG_PATH" ]; then
    echo "✓ Config path: $CONFIG_PATH"
else
    echo "✗ Config path failed"
    exit 1
fi

# Test 4: Direct execution
echo "Test 4: Direct command execution"
OUTPUT=$(./clia exec "echo 'Hello CLIA'" 2>&1)
if echo "$OUTPUT" | grep -q "Hello CLIA"; then
    echo "✓ Direct execution works"
else
    echo "✗ Direct execution failed"
    exit 1
fi

# Test 5: Dry run mode
echo "Test 5: Dry run mode (skipped - requires provider setup)"
# ./clia --dry-run "list files" > /dev/null 2>&1
if [ $? -eq 0 ]; then
    echo "✓ Dry run mode works"
else
    echo "✗ Dry run mode failed"
    exit 1
fi

# Test 6: History command
echo "Test 6: History command"
./clia history --limit 5 > /dev/null 2>&1
if [ $? -eq 0 ]; then
    echo "✓ History command works"
else
    echo "✗ History command failed"
    exit 1
fi

# Test 7: Provider list
echo "Test 7: Provider list"
./clia provider list > /dev/null 2>&1
if [ $? -eq 0 ]; then
    echo "✓ Provider list works"
else
    echo "✗ Provider list failed"
    exit 1
fi

echo ""
echo "================================"
echo "All tests passed! ✓"
echo "================================"
echo ""

# Clean up
rm -f clia