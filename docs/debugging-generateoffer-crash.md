# Debugging generateOffer() Crash Guide

**Last Updated:** 2026-02-12T19:45:00Z

## Problem Description

Clicking "Generate Invitation" button causes UI to disappear. This is likely due to a panic in the Go backend that isn't caught by the test suite.

## Root Cause

Tests use `testContext()` which sets `testModeKey=true`, suppressing `runtime.EventsEmit()` calls. Production uses a real context that triggers Wails event emission, which may panic.

## Debugging Steps

### 1. Run with Debug Logging

```bash
./scripts/debug-wails.sh
```

This captures all Go output to terminal and `/tmp/wails-debug.log`.

### 2. Check Browser Console

Open DevTools (F12) and watch console for:
- `[FRONTEND] generateOffer called`
- `[FRONTEND] Calling CreateOffer()...`
- `[FRONTEND] CreateOffer returned, token length: N`
- Any errors: `[FRONTEND] CreateOffer error:`

If you see "Calling CreateOffer()" but no "CreateOffer returned", the Go function panicked.

### 3. Check Go Output

Look for these patterns in the terminal:
- `[DEBUG] safeEventEmit: event='log', ctx=true, testMode=false`
- `[PANIC] CreateOffer recovered: ...` - Indicates a panic was caught
- `[WARN] safeEventEmit: ctx is nil` - Context initialization issue

### 4. Run Production-Context Tests

```bash
go test -v -run "RealContext"
```

These tests simulate production environment and may reveal issues that normal tests miss.

## Common Issues

### Panic in WebRTC Operations

**Symptoms:** `[PANIC] CreateOffer recovered: panic: ...` in output

**Possible Causes:**
- Network configuration issues
- ICE gathering failures
- STUN server unreachable
- Missing required permissions

**Solutions:**
- Check network connectivity
- Try different STUN servers
- Verify firewall settings
- Check system logs for denied permissions

### Context Not Initialized

**Symptoms:** `[WARN] safeEventEmit: ctx is nil` in output

**Solution:** Verify `startup()` is called by Wails during app initialization.

### Event Emission Panics

**Symptoms:** Panic occurs after `[DEBUG] safeEventEmit: emitting event...`

**Solution:** Wails runtime may be unavailable. Check if frontend is properly mounted.

## Next Steps

1. Run app with debug logging
2. Click "Generate Invitation"
3. Collect console output from browser and terminal
4. Identify panic location from stack trace
5. Fix root cause
6. Remove debug logging after fix

## Files Modified

- `app.go` - Added panic recovery and logging
- `debug_test.go` - Production-context tests
- `frontend/src/lib/tunnelStore.ts` - Frontend logging
- `scripts/debug-wails.sh` - Debug script

## Recovery Mechanisms

All WebRTC functions now have:
1. Panic recovery with stack trace
2. Detailed debug logging
3. Error handling that doesn't crash the app

The app should continue running even if a panic occurs in one function.
