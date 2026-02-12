#!/bin/bash
# Script to run Wails app with debug logging enabled

echo "Starting Wails app in debug mode..."
echo "Debug output will be captured to terminal"
echo ""
echo "When you click 'Generate Invitation', watch for:"
echo "  - [DEBUG] safeEventEmit messages"
echo "  - [PANIC] recovered panic messages"
echo "  - Frontend console logs (open browser DevTools)"
echo ""

# Run Wails in debug mode
# Note: Wails doesn't have a built-in -debug flag, but we can capture stderr/stdout
wails dev 2>&1 | tee /tmp/wails-debug.log
