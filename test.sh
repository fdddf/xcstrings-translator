#!/bin/bash

# env key


# Test script for xcstrings-translator

go build -o xcstrings-translator

echo "=== xcstrings-translator Test Script ==="
echo

# Check if binary exists
if [ ! -f "xcstrings-translator" ]; then
    echo "Error: xcstrings-translator binary not found. Please build it first."
    exit 1
fi

# Test help command
echo "1. Testing help command..."
./xcstrings-translator --help
echo

# Test version (if implemented)
echo "2. Testing version command..."
./xcstrings-translator -v 2>/dev/null || echo "Version command not implemented"
echo

# Test provider list
echo "3. Testing provider commands..."
for provider in google deepl baidu openai; do
    echo "Testing $provider provider help..."
    ./xcstrings-translator $provider --help | head -10
    echo "----------------------------------------"
done

# Test with example file
echo "4. Testing with example file..."
echo "Example file content:"
cat example.xcstrings | jq '.strings | keys'
echo

# Test dry run (without actual translation)
echo "5. Testing dry run with Google provider (simulated)..."
./xcstrings-translator google \
    --api-key "test-key" \
    --input "example.xcstrings" \
    --output "example_translated.xcstrings" \
    --target-languages "zh-Hans" \
    --verbose 2>&1 | grep -E "(Loading|Found|strings|Exiting)"

echo
echo "=== Test completed ==="