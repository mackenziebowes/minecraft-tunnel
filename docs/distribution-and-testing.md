# Distribution and In Vivo Testing Guide

**Last Updated:** 2026-02-12T20:00:00Z

## Overview

This guide covers distributing the built Minecraft Tunnel application and conducting in vivo testing with real users.

---

## 1. Quick Start

### How to Run the Application

The application is distributed as a platform-specific binary:

**Linux:**
```bash
cd build/bin
./minecraft-tunnel
```

**macOS (Apple Silicon):**
```bash
cd build/bin/macos-arm64
./minecraft-tunnel
```

**macOS (Intel):**
```bash
cd build/bin/macos-intel
./minecraft-tunnel
```

**Windows:**
```bash
cd build/bin/windows
./minecraft-tunnel.exe
```

### What to Expect on First Launch

1. **Main Menu** displays with options to "Host Tunnel" or "Join Tunnel"
2. **Select Host** to expose your local Minecraft server
3. **Select Join** to connect to a friend's Minecraft server
4. **UI loads** with status "disconnected"
5. **Browser DevTools** can be opened (F12) to see console logs

### System Requirements

**Required:**
- Modern browser (Chrome, Firefox, Edge, Safari) with WebRTC support
- Network connectivity (for STUN/ICE gathering)
- File system access (for token import/export)

**Recommended:**
- Stable internet connection (for reliable P2P connections)
- Firewall allows outbound UDP connections (ICE candidates)
- Desktop environment (not mobile)

---

## 2. In Vivo Testing Scenarios

### Scenario 1: Host → Join (Single Player)

**Purpose:** Test basic P2P connection between two users

**Host User Steps:**
1. Launch application
2. Click "Host Tunnel"
3. Enter Minecraft server address (default: `localhost:25565`)
4. Click "Generate Invitation"
5. **Expected:**
   - Status changes to "connecting" (yellow, animated)
   - Terminal shows `[DEBUG] safeEventEmit` messages
   - After ~2-5 seconds, status changes to "waiting-for-answer" (blue)
   - Offer token appears in "Share this Offer Token" section
   - Logs show "Offer token generated successfully"

6. Copy the offer token (click "Copy" button or select text)
7. Share with Join user (email, chat, etc.)

**Join User Steps:**
1. Launch application on different device (or same device)
2. Click "Join Tunnel"
3. Paste the offer token from Host user
4. **Choose one of three methods:**
   - **Paste directly** into the input field, click "Connect"
   - **Paste from Clipboard** - click button
   - **Import from File** - click button, select file containing token
5. **Expected:**
   - Status changes to "connecting"
   - After ~2-5 seconds, status changes to "waiting-for-host" (blue)
   - Answer token appears in "Share this Answer Token" section
   - Logs show "Answer generated - share this with host"
   - **Success toast notification:** "Token imported successfully" with filename

6. Copy the answer token
7. Return to Host user, paste answer token in "Paste Answer Token from friend" field
8. Click "Connect"

**Expected Final Result:**
- Both users see status "connected" (green badge)
- Logs show "P2P Tunnel Established! 🚀"
- Toast notification: "Token imported successfully" (Join user) / no toast (Host user)
- **UI does not disappear**

**What to Verify:**
- [ ] Offer token is base64-encoded valid JSON
- [ ] Answer token is base64-encoded valid JSON
- [ ] Status transitions: `disconnected` → `connecting` → `waiting-for-answer/host` → `connected`
- [ ] No panics in terminal (look for `[PANIC]` messages)
- [ ] No errors in browser console (F12)
- [ ] Both users can see logs from both sides

---

### Scenario 2: Host → Join (Multiplayer Server)

**Purpose:** Test tunnel connecting to an actual Minecraft server with multiple clients

**Prerequisites:**
- Minecraft server running and accessible
- Server address known (e.g., `myserver.com:25565`)

**Host User Steps:**
1. Launch application
2. Click "Host Tunnel"
3. Enter actual Minecraft server address (not `localhost:25565`)
   - Example: `myserver.com:25565` or `192.168.1.100:25565`
4. Click "Generate Invitation"
5. Share offer token with first Join user

**First Join User Steps:**
1. Paste offer token
2. Click "Connect"
3. Copy answer token
4. Share with Host

**Host User (After Receiving Answer):**
1. Paste answer token from first Join user
2. Click "Connect"
3. **Expected:**
   - Status changes to "connected"
   - Logs show "P2P Tunnel Established! 🚀"
   - Tunnel proxy starts forwarding to actual Minecraft server

**Verification Steps:**
1. First Join user connects to Minecraft via `localhost:25565`
2. Verify they can see server list, join world
3. Second Join user launches Minecraft
4. Enter Host user's IP/hostname (not `localhost`)
5. Join via tunnel
6. Verify they can see server list, join world, interact normally

**What to Verify:**
- [ ] First Join user connects via localhost (direct connection)
- [ ] Second Join user connects via tunnel (proxied connection)
- [ ] Both users can play simultaneously
- [ ] No noticeable latency introduced by tunnel
- [ ] Tunnel logs show data flow (once implemented)
- [ ] Minecraft client shows expected server

---

### Scenario 3: Error Cases

**Purpose:** Test error handling and user feedback

#### Test 3A: Invalid Token

**Steps:**
1. Try to paste invalid token (e.g., "not-a-valid-token")
2. Click "Connect"

**Expected:**
- Error status (red badge)
- Log message: "Error: invalid session description"
- Toast notification: "Failed to accept offer" with error details
- **UI does not disappear**

#### Test 3B: Network Disconnection

**Steps:**
1. Start Host → Join connection
2. Disconnect from network (disable WiFi, unplug ethernet)
3. Try to use UI (click buttons, navigate)

**Expected:**
- UI remains responsive (no crashes)
- Status may show "error" or remain disconnected
- Browser console may show network errors
- Reconnection attempt after network restored should work

#### Test 3C: Firewall Blocking STUN

**Steps:**
1. Configure firewall to block UDP port 19302 (STUN server)
2. Try to generate offer
3. Observe behavior

**Expected:**
- ICE gathering timeout after 30 seconds
- Error status shown
- Log message: "ICE gathering timeout: failed to gather candidates after 30s"
- Toast notification with error
- **No UI crash**
- Connection properly cleaned up (defer cleanup in Go code)

#### Test 3D: File Import/Export

**Steps for Export:**
1. Generate offer or answer token
2. Click "Export" button
3. **Expected:**
   - Success toast: "Token exported successfully"
   - Toast message: "Saved to your browser's Downloads folder"
   - File `covenant-offer.txt` or `covenant-answer.txt` appears in Downloads
   - Log: "Token exported to file"
   - File contains base64-encoded token

**Steps for Import:**
1. Click "Import File" button
2. Select token file from Downloads
3. **Expected:**
   - File dialog opens
   - Success toast: "Token imported successfully"
   - Toast message: "Loaded from {filename}"
   - Log: "Token imported from {filename}"
   - Token appears in input field

---

## 3. Debugging During Testing

### How to Use Debug Mode

Run the application with full logging enabled:

```bash
./scripts/debug-wails.sh
```

This script:
- Captures all terminal output
- Saves to `/tmp/wails-debug.log`
- Shows Go and Wails runtime messages

### Collecting Debug Information

**Terminal Output:**
- Look for `[DEBUG]` messages showing event emissions
- Look for `[PANIC]` messages (should be caught with recovery)
- Look for `[WARN]` messages about missing context or test mode

**Sample Debug Output (Normal Operation):**
```
[DEBUG] safeEventEmit: event='log', ctx=true, testMode=false
[DEBUG] safeEventEmit: emitting event 'log' to Wails runtime
[DEBUG] safeEventEmit: event 'log' emitted successfully
[FRONTEND] generateOffer called
[FRONTEND] Calling CreateOffer()...
[FRONTEND] CreateOffer returned, token length: 1916
[FRONTEND] State updated to waiting-for-answer
```

**Sample Debug Output (Panic Recovery):**
```
[PANIC] CreateOffer recovered: runtime error: invalid memory address or nil pointer dereference
goroutine 1 [running]:
github.com/pion/webrtc/v3@v3.2.48/example.org/webrtc.(*PeerConnection).CreateOffer(0x0, 0xc0000a00c0)
	/usr/local/go/src/github.com/pion/webrtc/v3/peerconnection.go:123 +0x123
```

**Browser DevTools (F12):**

Open DevTools to see frontend console logs:

**Frontend Logs (Expected):**
```
[FRONTEND] generateOffer called
[FRONTEND] Calling CreateOffer()...
[FRONTEND] CreateOffer returned, token length: 1916
[FRONTEND] Token preview: eyJzdGFh...
[FRONTEND] State updated to waiting-for-answer
```

**Frontend Errors (Bad):**
```
[FRONTEND] CreateOffer error: TypeError: window.go.main.App.CreateOffer is not a function
[FRONTEND] Error message: undefined
[FRONTEND] Error stack: TypeError...
```

**React Errors (Bad - Fixed):**
```
Warning: React has detected a change in the number of hooks
Warning: This is usually caused by calling a hook inside a render or condition
```

### Terminal Output Interpretation

| Message Pattern | Meaning | Action |
|----------------|----------|--------|
| `[DEBUG] safeEventEmit: event='log'` | Log being sent to frontend | Normal |
| `[DEBUG] safeEventEmit: event='status-change'` | Status changing | Normal |
| `[DEBUG] safeEventEmit: emitting event` | Wails runtime processing | Normal |
| `[WARN] safeEventEmit: ctx is nil` | Context not initialized | Check `startup()` |
| `[PANIC]` followed by panic details | Panic caught by recovery | Review stack trace |
| `[FRONTEND]` followed by action | Frontend action executed | Normal |
| `Error:` followed by message | Error occurred | Check logs for details |

---

## 4. Distribution Steps

### For Users

#### Which Binary to Download

Users should download the binary for their platform:

| Platform | Binary Name | Location |
|-----------|--------------|----------|
| Linux AMD64 | `minecraft-tunnel` | `build/bin/` |
| macOS Apple Silicon | `minecraft-tunnel` | `build/bin/macos-arm64/` |
| macOS Intel | `minecraft-tunnel` | `build/bin/macos-intel/` |
| Windows AMD64 | `minecraft-tunnel.exe` | `build/bin/windows/` |

**File Size:** ~13-15 MB

#### How to Verify Binary Integrity

After downloading, verify the SHA256 checksum:

**Linux/macOS:**
```bash
sha256sum minecraft-tunnel
```

**Windows:**
```powershell
certutil -hashfile minecraft-tunnel.exe -algorithm SHA256
```

Compare with published checksums (if provided).

#### Installation

**No Installation Required:**
- Binary is standalone
- Double-click to run (Linux/macOS/Windows)
- Executable flag: Already set during build

**macOS Note:**
On first launch, macOS may show security warning ("unverified developer"):
1. Open System Preferences → Security & Privacy
2. Click "Open Anyway"
3. Application will launch normally thereafter

**Windows Note:**
- Windows Defender may flag on first run
- Click "More info" → "Run anyway"

---

### For Developers

#### Cross-Platform Build Commands

**Build All Platforms:**
```bash
./scripts/build-all.sh
```

This builds for:
- Linux AMD64
- macOS Apple Silicon
- macOS Intel
- macOS Universal
- Windows

**Individual Builds:**
```bash
# Linux
./scripts/build-linux.sh

# macOS Apple Silicon
./scripts/build-macos-arm.sh

# macOS Intel
./scripts/build-macos-intel.sh

# macOS Universal
./scripts/build-macos-universal.sh

# Windows
./scripts/build-windows.sh
```

#### Where Binaries End Up

All binaries are placed in `build/bin/` with platform-specific subdirectories:

```
build/
├── bin/
│   ├── minecraft-tunnel          # Linux binary
│   ├── macos-arm64/
│   │   └── minecraft-tunnel      # macOS Apple Silicon
│   ├── macos-intel/
│   │   └── minecraft-tunnel      # macOS Intel
│   └── windows/
│       └── minecraft-tunnel.exe    # Windows executable
```

#### Signing Requirements

**macOS:**
- Requires Apple Developer account for code signing
- Use `codesign` to sign binaries
- Prevents security warnings on first launch
- Required for App Store distribution

**Windows:**
- Requires code signing certificate
- Use `signtool` or `SignTool`
- Prevents SmartScreen warnings
- Required for distribution outside local testing

**Linux:**
- No signing required for personal use
- Consider GPG signing for official releases
- Build with `CGO_ENABLED=0` to minimize dependencies

---

## 5. Common Issues & Troubleshooting

### Issue: "UI Disappears When Clicking Generate"

**Status:** ✅ FIXED

**Root Cause:** React Rules of Hooks violation - calling `useTunnelStore()` inside JSX render caused hook count mismatch between renders.

**Fix Applied:**
- Destructured `offerToken` and `answerToken` at component top
- Removed `useTunnelStore()` calls from JSX
- Both `HostView` and `JoinView` fixed

**Verification:**
- No more "React has detected a change in the number of hooks" warnings
- Status changes trigger properly
- UI remains visible after button clicks

---

### Issue: "Import/Export Buttons Don't Work"

**Status:** ✅ FIXED

**Root Cause:** Wails runtime doesn't have `SaveFileDialog()` and `OpenFileDialog()` methods, causing frontend to crash when calling undefined functions.

**Fix Applied:**
- **Import:** Hidden `<input type="file">` triggered via button, reads File object
- **Export:** Browser Blob API creates downloadable file (`covenant-offer.txt` or `covenant-answer.txt`)
- Removed non-existent Wails dialog Go functions

**Verification:**
- Import button opens file dialog, reads token successfully
- Export button triggers download to Downloads folder
- Success toasts appear
- No crashes

---

### Issue: Connection Timeout

**Status:** PARTIALLY ADDRESSED

**Cause:** ICE gathering can fail if STUN servers are unreachable or firewall blocks UDP.

**Current Behavior:**
- 30-second timeout (defined in `timeout.go`)
- Error message: "ICE gathering timeout"
- Connection cleanup with defer

**Troubleshooting Steps:**

1. **Check Network Connectivity:**
   ```bash
   # Test STUN server reachability
   nc -zv stun.l.google.com 19302
   ```
   Should succeed (port 19302 is UDP, so may show filtered)

2. **Check Firewall:**
   - Ensure outbound UDP traffic allowed
   - No strict firewall blocking STUN servers
   - Test with VPN disabled

3. **Test with Different STUN Servers:**

   Modify `app.go` to use alternative STUN servers:
   ```go
   config := webrtc.Configuration{
       ICEServers: []webrtc.ICEServer{
           {URLs: []string{"stun:stun1.l.google.com:19302"}},
           {URLs: []string{"stun:stun2.l.google.com:19302"}},
       },
   }
   ```

4. **Increase Timeout (Temporary):**

   Modify `timeout.go`:
   ```go
   const TimeoutWebRTCICE = 60 * time.Second  // Increase from 30s to 60s
   ```

---

### Issue: "No Panics But App Still Crashes"

**Possible Causes:**

1. **Context Not Initialized:**
   - `a.ctx` is nil when WebRTC operations run
   - **Symptom:** `[WARN] safeEventEmit: ctx is nil` logs
   - **Fix:** Ensure `startup()` is called by Wails

2. **Wails Runtime Error:**
   - Wails bridge failure
   - **Symptom:** Frontend console shows undefined Wails functions
   - **Fix:** Rebuild Wails bindings with `wails generate module`

3. **Browser Compatibility:**
   - Browser doesn't support WebRTC
   - **Symptom:** Browser console errors about `RTCPeerConnection`
   - **Fix:** Test in modern browser (Chrome 90+, Firefox 85+)

4. **Memory Exhaustion:**
   - Too many simultaneous connections
   - **Symptom:** App freezes or crashes after multiple operations
   - **Fix:** Limit concurrent connections, close unused peer connections

---

## 6. Verification Checklist

Use this checklist when testing:

### Before Testing

- [ ] Binary builds without errors
- [ ] Frontend builds without TypeScript errors
- [ ] All tests pass: `go test -v`
- [ ] All tests pass: `npm run test` (if configured)
- [ ] No console errors in browser DevTools
- [ ] No warnings in terminal
- [ ] Binary file size reasonable (<20 MB)

### During Testing

- [ ] Scenario 1 (Host → Join Single) works end-to-end
- [ ] Scenario 2 (Host → Join Multiplayer) connects successfully
- [ ] Scenario 3A (Invalid Token) shows proper error
- [ ] Scenario 3B (Network Disconnect) handles gracefully
- [ ] Scenario 3C (Firewall) shows timeout error
- [ ] Scenario 3D (File I/O) import/export work correctly

### After Testing

- [ ] No panics observed (check terminal for `[PANIC]`)
- [ ] No React hook warnings
- [ ] No browser console errors
- [ ] No Wails runtime errors
- [ ] Status transitions follow expected path
- [ ] Tokens are valid base64
- [ ] UI remains responsive throughout
- [ ] Logs show expected information
- [ ] Toast notifications appear and dismiss correctly
- [ ] Export files have correct names (`covenant-offer.txt` / `covenant-answer.txt`)

### Performance

- [ ] Page load time < 2 seconds
- [ ] Generate Offer completes in < 5 seconds
- [ ] Accept Offer completes in < 5 seconds
- [ ] Import/Export completes in < 1 second
- [ ] No noticeable UI lag during operations
- [ ] Memory usage stable (monitor with Activity Monitor / Task Manager)

---

## 7. Reporting Issues

When reporting bugs or issues during testing, include:

### Required Information

**Environment:**
- Platform: [Linux / macOS / Windows] (specify version if known)
- Architecture: [AMD64 / ARM64 / x86]
- Browser: [Chrome / Firefox / Edge / Safari] (include version)
- Go version: `go version`
- Node version: `node -v` (if testing frontend separately)

**Steps to Reproduce:**
1. Exact sequence of actions that led to the issue
2. What was expected to happen
3. What actually happened
4. Error messages (copy from terminal/browser console)

**Logs:**
- Terminal output (copy full error and 5 lines before/after)
- Browser console output (copy errors from DevTools)
- Screenshot (if UI-related issue)

**Additional Context:**
- Network connection (WiFi / Ethernet / mobile data)
- Firewall configuration (if known)
- VPN status
- Number of simultaneous users

### Bug Report Template

```
**Summary:** [Brief 1-2 sentence description]

**Environment:**
- Platform: macOS 14.5 (Sonoma)
- Browser: Chrome 121.0.6334.90
- Go version: go1.21.6

**Steps to Reproduce:**
1. Launch application
2. Click "Host Tunnel"
3. Enter server address: localhost:25565
4. Click "Generate Invitation"
5. UI disappears

**Expected Behavior:**
- Status changes to "connecting" → "waiting-for-answer"
- Offer token appears

**Actual Behavior:**
- UI immediately disappears after clicking "Generate Invitation"

**Error Messages:**
Terminal:
```
[PANIC] CreateOffer recovered: runtime error: invalid memory address
...
```

Browser Console:
```
Warning: React has detected a change in the number of hooks
```

**Additional Context:**
- First time running after fresh build
- Network: WiFi, connected
- No VPN enabled
```

---

## 8. Technical Notes

### ICE Gathering Timeout

**Current Setting:** 30 seconds
**Location:** `timeout.go:11`
```go
const TimeoutWebRTCICE = 30 * time.Second
```

**Why:** WebRTC needs time to gather all ICE candidates (local IP addresses, STUN, TURN).

**Behavior:**
- First candidate appears immediately (usually < 1s)
- Additional candidates trickle in over time
- GatheringCompletePromise resolves when all candidates collected
- Timeout prevents indefinite hanging

---

### STUN Server

**Current Setting:** Google's public STUN server
**Location:** `app.go:77`
```go
{URLs: []string{"stun:stun.l.google.com:19302"}}
```

**Purpose:** Enables NAT traversal by discovering public IP addresses.

**Limitations:**
- Doesn't work behind symmetric NAT without TURN
- Rate limited (don't spam create offers)
- Publicly accessible, not encrypted

---

### Default Ports

**Minecraft Default:** 25565
**Location:** `app.go:233`
```go
mcConn, err := DialTimeout("tcp", "localhost:25565", TimeoutTCPConnect)
```

**Usage:**
- Host mode: Connects to local Minecraft server on port 25565
- Join mode: Listens on port 25565 for incoming connections

**Customization:**
- User can change server address in HostView
- User can change proxy port in JoinView
- Used when running Minecraft on non-default port

---

### Data Channel

**Name:** `"minecraft"`
**Location:** `app.go:97`
```go
dataChannel, err := peerConnection.CreateDataChannel("minecraft", nil)
```

**Purpose:** Bidirectional reliable data channel for Minecraft protocol traffic.

**Properties:**
- Ordered message delivery
- Reliable transport
- Maximum message size: 16KB (browser limit)

---

### Token Encoding

**Format:** Base64-encoded JSON
**Structure:**
```json
{
  "type": "offer" | "answer",
  "sdp": "v=0\r\no=...",
  "candidates": [...]
}
```

**Location in code:**
- `app.go:143` (CreateOffer): `base64.StdEncoding.EncodeToString(offerJson)`
- `app.go:228` (AcceptOffer): `base64.StdEncoding.EncodeToString(answerJson)`

**Why Base64:**
- Easy to copy-paste (single string)
- URL-safe characters only
- Survives email/chat transmission

---

### Filename Convention

**Export Files:**
- Offer tokens: `covenant-offer.txt`
- Answer tokens: `covenant-answer.txt`

**Prefix Meaning:** "Covenant" refers to the reciprocal agreement between two peers establishing a connection. This theme runs throughout the application (tunnel, offer/answer tokens).

---

## 9. Quick Reference

### Console Commands

```bash
# Run app in debug mode
./scripts/debug-wails.sh

# Run all tests
go test -v

# Run specific test
go test -v -run TestCreateOffer

# Build all platforms
./scripts/build-all.sh

# Clean build artifacts
go clean
rm -rf build/
rm -rf frontend/dist/
```

### Browser DevTools Shortcuts

| Action | Shortcut | Browser |
|--------|----------|----------|
| Open DevTools | F12 | Most browsers |
| Toggle device mode | Ctrl+Shift+M | Chrome/Edge |
| Focus console | Ctrl+Shift+J | Chrome/Edge |
| Reload page | Ctrl+R / Cmd+R | Most browsers |

### Log Levels

| Level | Prefix | Usage |
|-------|---------|-------|
| Debug | `[DEBUG]` | Normal operation info |
| Warning | `[WARN]` | Non-critical issues |
| Panic | `[PANIC]` | Recovered crashes |
| Frontend | `[FRONTEND]` | Client-side actions |
| Error | `Error:` | General failures |

---

## 10. Success Criteria

A successful in vivo test session should demonstrate:

1. **Functional P2P Connection:**
   - Two users can establish tunnel successfully
   - Status progresses through all expected states
   - No crashes or errors

2. **Token Management:**
   - Offer tokens generate correctly
   - Answer tokens generate correctly
   - Import/export works reliably
   - Filenames correctly indicate type

3. **Error Handling:**
   - Invalid tokens show clear error messages
   - Timeouts handled gracefully
   - Network disruptions don't crash app

4. **User Experience:**
   - Clear status indicators
   - Informative toast notifications
   - Helpful logs for debugging
   - Responsive UI throughout

5. **Stability:**
   - No panics in Go code
   - No React hook violations
   - No memory leaks
   - Proper cleanup on all error paths

---

## 11. Next Steps After Testing

After completing in vivo testing:

1. **Collect Feedback:** Document what worked well and what needs improvement
2. **Address Issues:** Fix any bugs discovered during testing
3. **Update Documentation:** Revise this guide based on real-world findings
4. **Release Preparation:** Prepare release notes with tested features
5. **Distribution:** Create release assets (README, checksums, screenshots)
