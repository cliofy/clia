#!/bin/bash

# Run OpenRouter integration tests
# This script loads environment variables from .env and runs the integration tests

echo "Running OpenRouter Integration Tests"
echo "===================================="

# Run with integration build tag
go test -v -tags=integration ./core/provider/openrouter -run TestOpenRouterIntegration

echo ""
echo "To run these tests manually:"
echo "  go test -v -tags=integration ./core/provider/openrouter"
echo ""
echo "Make sure you have set:"
echo "  - OPENROUTER_KEY in .env file"
echo "  - MODEL_NAME in .env file (optional)"