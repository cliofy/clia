#!/bin/bash

# Test script for interactive commands

echo "Building CLIA..."
go build -o clia ./cmd/clia

echo ""
echo "================================"
echo "Testing Interactive Commands"
echo "================================"
echo ""

echo "Note: Interactive commands will take control of your terminal."
echo "Press 'q' to exit top, ':q' to exit vim, etc."
echo ""

echo "Test 1: Testing 'top' command"
echo "Command: ./clia exec top"
echo "Press any key to start (will auto-exit after 2 seconds)..."
read -n 1

# Use timeout to auto-exit after 2 seconds
timeout 2 ./clia exec "top" || true

echo ""
echo "Test 2: Testing 'ls' (non-interactive) command"
./clia exec "ls -la | head -5"

echo ""
echo "Test 3: Testing detection of interactive vs non-interactive"
echo "This should use interactive mode:"
./clia exec "vim --version | head -1" 

echo ""
echo "================================"
echo "Interactive tests completed!"
echo "================================"
echo ""

# Clean up
rm -f clia