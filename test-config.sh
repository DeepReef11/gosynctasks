#!/bin/bash

# Test script to verify --config flag behavior

echo "=== Building gosynctasks with debug output ==="
go build -o gosynctasks-debug ./cmd/gosynctasks

echo ""
echo "=== Test 1: Run without --config flag ==="
./gosynctasks-debug --list-backends

echo ""
echo "=== Test 2: Run with --config . ==="
./gosynctasks-debug --config . --list-backends

echo ""
echo "=== Test 3: Run with --config ./gosynctasks/config ==="
./gosynctasks-debug --config ./gosynctasks/config --list-backends

echo ""
echo "=== Test 4: Check what config file you expect to use ==="
echo "Please tell me the path to your config file:"
read CONFIG_PATH
echo "Running with: ./gosynctasks-debug --config \"$CONFIG_PATH\" --list-backends"
./gosynctasks-debug --config "$CONFIG_PATH" --list-backends

echo ""
echo "=== Cleanup ==="
rm -f gosynctasks-debug
