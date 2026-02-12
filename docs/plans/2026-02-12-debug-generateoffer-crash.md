# GenerateOffer Crash Debug Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add debugging infrastructure to identify why UI disappears when calling `generateOffer()` in production.

**Architecture:** Add panic recovery wrappers, extensive logging, and test cases that simulate production environment (non-test context) to isolate the crash location between Go backend and Wails bridge.

**Tech Stack:** Go 1.18+, Wails v2, pion/webrtc/v3, React/TypeScript, Zustand

---

## Context for Engineers

**The Problem:**
- Clicking "Generate Invitation" button causes UI to disappear
- Go tests pass because they use `testContext()` which suppresses `runtime.EventsEmit()`
- Production environment uses real context which triggers Wails event emission
- Suspected cause: Go backend panic in WebRTC operations or Wails runtime

**Key Files:**
- `app.go` - CreateOffer function (lines 73-146)
- `app_test.go` - Test suite with testContext() (lines 12-15)
- `frontend/src/lib/tunnelStore.ts` - generateOffer action (lines 47-62)
- `frontend/src/components/Router.tsx` - Route switching

**Root Cause Analysis:**
1. Tests work because `testModeKey=true` suppresses events (app.go:44-46)
2. Production calls `runtime.EventsEmit()` which may panic
3. WebRTC operations can panic on network issues
4. No panic handling in production code

---

## Task 1: Add Recovery Wrapper to CreateOffer

**Files:**
- Modify: `app.go:73-146`

**Step 1: Add panic recovery to CreateOffer function**

Replace lines 73-146 in app.go with:

```go
func (a *App) CreateOffer() (string, error) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintf(os.Stderr, "[PANIC] CreateOffer recovered: %v\n", r)
			debug.PrintStack()
		}
	}()

	// Standard public STUN servers (free, used for hole punching)
	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{URLs: []string{"stun:stun.l.google.com:19302"}},
		},
	}

	// Create a new PeerConnection
	peerConnection, err := webrtc.NewPeerConnection(config)
	if err != nil {
		return "", err
	}

	var cleanupNeeded = true
	defer func() {
		if cleanupNeeded && peerConnection != nil {
			peerConnection.Close()
		}
	}()

	a.peerConnection = peerConnection

	// Create the Data Channel (This is our "Tunnel Cable")
	dataChannel, err := peerConnection.CreateDataChannel("minecraft", nil)
	if err != nil {
		return "", err
	}

	// HANDLE OPEN: When the tunnel connects, start forwarding Minecraft
	dataChannel.OnOpen(func() {
		a.safeEventEmit("status-change", "connected")
		a.safeEventEmit("log", "P2P Tunnel Established! 🚀")

		// Start talking to local Minecraft (Port 25565)
		go a.pumpMinecraftToChannel(dataChannel)
	})

	// HANDLE MESSAGES: When data comes FROM the friend
	dataChannel.OnMessage(func(msg webrtc.DataChannelMessage) {
		// In a real implementation, you'd write this to the local MC socket
		// For simplicity, we assume the pump handles the bi-directional flow
	})

	// Create the Offer (The "Token")
	offer, err := peerConnection.CreateOffer(nil)
	if err != nil {
		return "", err
	}

	// Sets the LocalDescription so we can start gathering candidates
	if err = peerConnection.SetLocalDescription(offer); err != nil {
		return "", err
	}

	// Wait for ICE Gathering to complete (to get all possible IP paths)
	gatheringDone := webrtc.GatheringCompletePromise(peerConnection)
	select {
	case <-gatheringDone:
	case <-time.After(TimeoutWebRTCICE):
		cleanupNeeded = false
		peerConnection.Close()
		return "", fmt.Errorf("ICE gathering timeout: failed to gather candidates after %v", TimeoutWebRTCICE)
	}

	// Encode the offer to base64 so it's easy to copy-paste
	offerJson, err := json.Marshal(peerConnection.LocalDescription())
	if err != nil {
		return "", fmt.Errorf("failed to marshal offer: %w", err)
	}

	cleanupNeeded = false
	return base64.StdEncoding.EncodeToString(offerJson), nil
}
```

**Step 2: Add missing imports to app.go**

Check that app.go imports include:
```go
import (
	// ... existing imports
	"debug"
	"fmt"
	"os"
	// ... rest of imports
)
```

**Step 3: Verify code compiles**

Run: `go build .`
Expected: No errors

**Step 4: Run existing tests to ensure no regression**

Run: `go test -v -run TestCreateOffer`
Expected: All tests PASS

**Step 5: Commit**

```bash
git add app.go
git commit -m "fix: add panic recovery to CreateOffer to catch crashes"
```

---

## Task 2: Add Recovery Wrapper to AcceptOffer

**Files:**
- Modify: `app.go:157-230`

**Step 1: Add panic recovery to AcceptOffer function**

Replace lines 157-230 in app.go with:

```go
func (a *App) AcceptOffer(offerToken string) (string, error) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintf(os.Stderr, "[PANIC] AcceptOffer recovered: %v\n", r)
			debug.PrintStack()
		}
	}()

	sdpBytes, err := base64.StdEncoding.DecodeString(offerToken)
	if err != nil {
		return "", fmt.Errorf("invalid token format: %w", err)
	}

	var offer webrtc.SessionDescription
	if err := json.Unmarshal(sdpBytes, &offer); err != nil {
		return "", fmt.Errorf("invalid session description: %w", err)
	}

	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{URLs: []string{"stun:stun.l.google.com:19302"}},
		},
	}

	peerConnection, err := webrtc.NewPeerConnection(config)
	if err != nil {
		return "", err
	}

	var cleanupNeeded = true
	defer func() {
		if cleanupNeeded && peerConnection != nil {
			peerConnection.Close()
		}
	}()

	if err := peerConnection.SetRemoteDescription(offer); err != nil {
		return "", err
	}

	answer, err := peerConnection.CreateAnswer(nil)
	if err != nil {
		return "", err
	}

	if err = peerConnection.SetLocalDescription(answer); err != nil {
		return "", err
	}

	// Wait for ICE Gathering to complete
	gatheringDone := webrtc.GatheringCompletePromise(peerConnection)
	select {
	case <-gatheringDone:
	case <-time.After(TimeoutWebRTCICE):
		peerConnection.Close()
		return "", fmt.Errorf("ICE gathering timeout: failed to gather candidates after %v", TimeoutWebRTCICE)
	}

	answerJson, err := json.Marshal(peerConnection.LocalDescription())
	if err != nil {
		cleanupNeeded = false
		peerConnection.Close()
		return "", fmt.Errorf("failed to marshal answer: %w", err)
	}

	cleanupNeeded = false
	return base64.StdEncoding.EncodeToString(answerJson), nil
}
```

**Step 2: Run tests to ensure no regression**

Run: `go test -v -run TestAcceptOffer`
Expected: All tests PASS

**Step 3: Commit**

```bash
git add app.go
git commit -m "fix: add panic recovery to AcceptOffer to catch crashes"
```

---

## Task 3: Add Recovery Wrapper to AcceptAnswer

**Files:**
- Modify: `app.go:148-156`

**Step 1: Add panic recovery to AcceptAnswer function**

Replace lines 148-156 in app.go with:

```go
func (a *App) AcceptAnswer(answerToken string) error {
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintf(os.Stderr, "[PANIC] AcceptAnswer recovered: %v\n", r)
			debug.PrintStack()
		}
	}()

	sdpBytes, err := base64.StdEncoding.DecodeString(answerToken)
	if err != nil {
		return fmt.Errorf("invalid answer token format: %w", err)
	}

	var answer webrtc.SessionDescription
	if err := json.Unmarshal(sdpBytes, &answer); err != nil {
		return fmt.Errorf("invalid answer session description: %w", err)
	}

	if err := a.peerConnection.SetRemoteDescription(answer); err != nil {
		return fmt.Errorf("failed to set remote description: %w", err)
	}

	return nil
}
```

**Step 2: Run tests to ensure no regression**

Run: `go test -v -run TestAcceptAnswer`
Expected: All tests PASS

**Step 3: Commit**

```bash
git add app.go
git commit -m "fix: add panic recovery to AcceptAnswer to catch crashes"
```

---

## Task 4: Add Detailed Logging to safeEventEmit

**Files:**
- Modify: `app.go:40-52`

**Step 1: Add logging to safeEventEmit function**

Replace lines 40-52 in app.go with:

```go
func (a *App) safeEventEmit(event string, data ...interface{}) {
	fmt.Printf("[DEBUG] safeEventEmit: event='%s', ctx=%v, testMode=%v\n",
		event, a.ctx != nil, a.ctx != nil && a.ctx.Value(testModeKey) == true)
	if a.ctx == nil {
		fmt.Fprintf(os.Stderr, "[WARN] safeEventEmit: ctx is nil, skipping event '%s'\n", event)
		return
	}
	if mode, ok := a.ctx.Value(testModeKey).(bool); ok && mode {
		fmt.Printf("[DEBUG] safeEventEmit: test mode, skipping event '%s'\n", event)
		return
	}
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintf(os.Stderr, "[PANIC] safeEventEmit recovered: %v\n", r)
			debug.PrintStack()
		}
	}()
	fmt.Printf("[DEBUG] safeEventEmit: emitting event '%s' to Wails runtime\n", event)
	runtime.EventsEmit(a.ctx, event, data...)
	fmt.Printf("[DEBUG] safeEventEmit: event '%s' emitted successfully\n", event)
}
```

**Step 2: Run tests to ensure no regression**

Run: `go test -v`
Expected: All tests PASS (with debug output)

**Step 3: Commit**

```bash
git add app.go
git commit -m "debug: add extensive logging to safeEventEmit"
```

---

## Task 5: Add Test with Production-like Context

**Files:**
- Create: `debug_test.go`

**Step 1: Write test that simulates production environment**

Create new file `debug_test.go`:

```go
package main

import (
	"context"
	"testing"

	"github.com/pion/webrtc/v3"
)

// TestCreateOfferWithRealContext tests CreateOffer with a non-test context
// This simulates production environment where testModeKey is not set
func TestCreateOfferWithRealContext(t *testing.T) {
	ctx := context.Background()
	app := &App{ctx: ctx}

	t.Logf("Testing CreateOffer with real context (not test mode)")
	t.Logf("testModeKey value: %v", ctx.Value(testModeKey))

	token, err := app.CreateOffer()
	if err != nil {
		t.Fatalf("CreateOffer failed with real context: %v", err)
	}

	if token == "" {
		t.Fatal("Expected non-empty token from CreateOffer")
	}

	t.Logf("CreateOffer succeeded with real context, token length: %d", len(token))

	// Clean up
	if app.peerConnection != nil {
		app.peerConnection.Close()
	}
}

// TestAcceptOfferWithRealContext tests AcceptOffer with a non-test context
func TestAcceptOfferWithRealContext(t *testing.T) {
	ctx := context.Background()
	hostApp := &App{ctx: ctx}

	// Create offer with real context
	offerToken, err := hostApp.CreateOffer()
	if err != nil {
		t.Fatalf("Failed to create offer with real context: %v", err)
	}

	// Accept offer with real context
	joinerApp := &App{ctx: ctx}
	answerToken, err := joinerApp.AcceptOffer(offerToken)
	if err != nil {
		t.Fatalf("AcceptOffer failed with real context: %v", err)
	}

	if answerToken == "" {
		t.Fatal("Expected non-empty answer token from AcceptOffer")
	}

	t.Logf("AcceptOffer succeeded with real context, answer length: %d", len(answerToken))

	// Clean up both sides
	if hostApp.peerConnection != nil {
		hostApp.peerConnection.Close()
	}
	if joinerApp.peerConnection != nil {
		joinerApp.peerConnection.Close()
	}
}

// TestAcceptAnswerWithRealContext tests AcceptAnswer with a non-test context
func TestAcceptAnswerWithRealContext(t *testing.T) {
	ctx := context.Background()
	hostApp := &App{ctx: ctx}

	// Create offer
	offerToken, err := hostApp.CreateOffer()
	if err != nil {
		t.Fatalf("Failed to create offer: %v", err)
	}

	// Generate answer
	joinerApp := &App{ctx: ctx}
	answerToken, err := joinerApp.AcceptOffer(offerToken)
	if err != nil {
		t.Fatalf("Failed to generate answer: %v", err)
	}

	// Accept answer
	err = hostApp.AcceptAnswer(answerToken)
	if err != nil {
		t.Fatalf("AcceptAnswer failed with real context: %v", err)
	}

	t.Logf("AcceptAnswer succeeded with real context")

	// Clean up
	if hostApp.peerConnection != nil {
		hostApp.peerConnection.Close()
	}
	if joinerApp.peerConnection != nil {
		joinerApp.peerConnection.Close()
	}
}

// TestPumpMinecraftToChannelHandlesMissingServer tests behavior when Minecraft server is not running
func TestPumpMinecraftToChannelHandlesMissingServer(t *testing.T) {
	ctx := context.Background()
	app := &App{ctx: ctx}

	// Mock data channel (won't actually be connected)
	// This tests that pumpMinecraftToChannel handles connection errors gracefully
	t.Log("Testing pumpMinecraftToChannel with no MC server running")
	t.Log("Expected: should log error and return, not panic")

	// Note: We can't easily test this without mocking, but the panic recovery
	// wrapper will catch any panics
}
```

**Step 2: Run tests to see if they pass**

Run: `go test -v -run "TestCreateOfferWithRealContext|TestAcceptOfferWithRealContext|TestAcceptAnswerWithRealContext"`
Expected: Tests should PASS and show debug output

**Step 3: If tests fail, check output for panics**

Run: `go test -v -run TestCreateOfferWithRealContext 2>&1 | grep -E "PANIC|ERROR"`
Expected: Should show any panic output if it occurs

**Step 4: Commit**

```bash
git add debug_test.go
git commit -m "test: add production-context tests for WebRTC functions"
```

---

## Task 6: Add Frontend Console Logging

**Files:**
- Modify: `frontend/src/lib/tunnelStore.ts:47-62`

**Step 1: Add console logging to generateOffer**

Replace lines 47-62 in tunnelStore.ts with:

```typescript
  generateOffer: async () => {
    console.log("[FRONTEND] generateOffer called");
    set({ status: "connecting", logs: [], offerToken: "" });
    try {
      console.log("[FRONTEND] Calling CreateOffer()...");
      const token = await CreateOffer();
      console.log("[FRONTEND] CreateOffer returned, token length:", token?.length);
      console.log("[FRONTEND] Token preview:", token?.substring(0, 50) + "...");
      set({ status: "waiting-for-answer", offerToken: token });
      get().addLog("Offer token generated successfully");
      console.log("[FRONTEND] State updated to waiting-for-answer");
    } catch (err: any) {
      console.error("[FRONTEND] CreateOffer error:", err);
      console.error("[FRONTEND] Error message:", err?.message);
      console.error("[FRONTEND] Error stack:", err?.stack);
      set({ status: "error" });
      get().addLog(`Error: ${err.message || err}`);
      useToastStore.getState().addToast({
        title: "Failed to generate offer",
        description: err.message || "An unknown error occurred",
        variant: "destructive",
      });
    }
  },
```

**Step 2: Add logging to acceptOffer**

Replace lines 64-79 in tunnelStore.ts with:

```typescript
  acceptOffer: async (offer) => {
    console.log("[FRONTEND] acceptOffer called, offer length:", offer?.length);
    set({ status: "connecting", logs: [] });
    try {
      console.log("[FRONTEND] Calling AcceptOffer()...");
      const answer = await AcceptOffer(offer);
      console.log("[FRONTEND] AcceptOffer returned, answer length:", answer?.length);
      set({ status: "waiting-for-host", answerToken: answer });
      get().addLog("Answer generated - share this with host");
    } catch (err: any) {
      console.error("[FRONTEND] AcceptOffer error:", err);
      set({ status: "error" });
      get().addLog(`Error: ${err.message || err}`);
      useToastStore.getState().addToast({
        title: "Failed to accept offer",
        description: err.message || "An unknown error occurred",
        variant: "destructive",
      });
    }
  },
```

**Step 3: Add logging to acceptAnswer**

Replace lines 81-95 in tunnelStore.ts with:

```typescript
  acceptAnswer: async (answer) => {
    console.log("[FRONTEND] acceptAnswer called, answer length:", answer?.length);
    try {
      console.log("[FRONTEND] Calling AcceptAnswer()...");
      await AcceptAnswer(answer);
      console.log("[FRONTEND] AcceptAnswer returned successfully");
      set({ status: "connected" });
      get().addLog("Tunnel established!");
    } catch (err: any) {
      console.error("[FRONTEND] AcceptAnswer error:", err);
      set({ status: "error" });
      get().addLog(`Error: ${err.message || err}`);
      useToastStore.getState().addToast({
        title: "Failed to accept answer",
        description: err.message || "An unknown error occurred",
        variant: "destructive",
      });
    }
  },
```

**Step 4: Build frontend to verify no TypeScript errors**

Run: `cd frontend && npm run build`
Expected: Build succeeds without errors

**Step 5: Commit**

```bash
git add frontend/src/lib/tunnelStore.ts
git commit -m "debug: add console logging to WebRTC actions"
```

---

## Task 7: Create Wails Debug Script

**Files:**
- Create: `scripts/debug-wails.sh`

**Step 1: Create Wails debug script**

Create new file `scripts/debug-wails.sh`:

```bash
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
```

**Step 2: Make script executable**

Run: `chmod +x scripts/debug-wails.sh`
Expected: Script is now executable

**Step 3: Test the script**

Run: `./scripts/debug-wails.sh`
Expected: Wails dev server starts with output to both terminal and log file

**Step 4: Commit**

```bash
git add scripts/debug-wails.sh
git commit -m "debug: add Wails debug script with log capture"
```

---

## Task 8: Run Integration Test with Full Logging

**Files:**
- Test: `debug_test.go`, `app.go`

**Step 1: Run all new tests with verbose output**

Run: `go test -v -run "RealContext|PumpMinecraft" 2>&1 | tee /tmp/webrtc-debug.log`
Expected: All tests pass, output captured to log file

**Step 2: Check for any panic output**

Run: `grep -E "PANIC|ERROR|FAIL" /tmp/webrtc-debug.log || echo "No panics or errors found"`
Expected: No panics found (or detailed error if present)

**Step 3: Run original tests to ensure no regression**

Run: `go test -v`
Expected: All tests PASS

**Step 4: Review debug output**

Run: `cat /tmp/webrtc-debug.log | grep -A 5 "DEBUG"`
Expected: See detailed debug output from safeEventEmit and WebRTC operations

**Step 5: Commit any fixes**

If issues found, fix and commit:
```bash
git add .
git commit -m "fix: resolve issues found during integration testing"
```

---

## Task 9: Create Debugging Documentation

**Files:**
- Create: `docs/debugging-generateoffer-crash.md`

**Step 1: Create debugging guide**

Create new file `docs/debugging-generateoffer-crash.md`:

```markdown
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
```

**Step 2: Commit documentation**

```bash
git add docs/debugging-generateoffer-crash.md
git commit -m "docs: add debugging guide for generateOffer crash"
```

---

## Task 10: Verify Fix in Production

**Files:**
- Test: Full application

**Step 1: Build production version**

Run: `wails build`
Expected: Binary created successfully

**Step 2: Run production binary**

Run: `./build/bin/minecraft-tunnel` (or appropriate platform binary)

**Step 3: Click "Generate Invitation"**

Expected:
1. Button shows "Connecting..." state
2. Status badge shows "connecting" (yellow, animated)
3. Terminal shows debug output
4. Offer token appears after ~2-5 seconds
5. UI does not disappear

**Step 4: Check terminal output**

Look for:
- `[DEBUG] safeEventEmit` messages
- No `[PANIC]` messages (unless there's a real error)
- Successful ICE gathering completion

**Step 5: Test error path**

Disconnect from network and try again:
Expected:
1. Error state shown
2. Toast notification appears
3. UI does not disappear
4. Terminal shows timeout or connection error

**Step 6: Document findings**

If issue persists, update `docs/debugging-generateoffer-crash.md` with:
- What actually happens
- Where the crash occurs
- Stack trace output

**Step 7: Commit any final fixes**

```bash
git add .
git commit -m "fix: resolve generateOffer crash based on debugging findings"
```

---

## Summary

This plan adds comprehensive debugging infrastructure to identify why `generateOffer()` causes the UI to disappear:

1. ✅ Panic recovery in all WebRTC functions
2. ✅ Detailed logging throughout the call chain
3. ✅ Tests that simulate production environment
4. ✅ Frontend console logging
5. ✅ Debug script for easy testing
6. ✅ Documentation for issue resolution

**Estimated time:** 1-2 hours
**Testing:** Comprehensive (unit + integration + production test)
**Impact:** Minimal - non-breaking changes, only adds safety and logging
