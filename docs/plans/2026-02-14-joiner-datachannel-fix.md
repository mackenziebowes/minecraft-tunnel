# Joiner DataChannel Fix - Design Analysis

Date: 2026-02-14

## Problem Statement

The app has never successfully tunneled Minecraft traffic. The root cause: the joiner side never handles the incoming DataChannel from the host.

## WebRTC Connection Primer

```
┌─────────────┐                    ┌─────────────┐
│    HOST     │                    │   JOINER    │
└──────┬──────┘                    └──────┬──────┘
       │                                  │
       │  1. Create DataChannel "mc"      │
       │  2. Create OFFER (SDP)           │
       │  ──────────────────────────────► │  (out of band: token paste)
       │                                  │  3. Set remote description
       │                                  │  4. Create ANSWER
       │  ◄────────────────────────────── │  (out of band: token paste)
       │  5. Set remote description       │
       │                                  │
       │  ═════════ ICE Negotiation ═════════
       │                                  │
       │  ═════════ DataChannel Opens ═══════
       │                                  │
       │  ◄═════════ Bidirectional ═══════►│
```

Key concepts:
- **SDP (Session Description Protocol):** Describes what codecs, IPs, and capabilities each peer supports
- **Offer/Answer:** Host creates offer, joiner creates answer - this is the "handshake"
- **ICE (Interactive Connectivity Establishment):** After offer/answer, peers exchange connectivity candidates to find a direct path
- **DataChannel:** Once connected, this is like a TCP socket - send/receive raw bytes

## Current Implementation Analysis

### Section 1: Host Creates Offer (WORKING)

**What the user does:** Opens app, clicks "Host", clicks "Generate Invitation"

**Code path:**

1. **Frontend** (`routes/host/index.tsx:194`): Button calls `generateOffer()`

2. **Frontend** (`tunnelStore.ts:45-68`):
```typescript
generateOffer: async () => {
  set({ status: "connecting", logs: [], offerToken: "" });
  const token = await CreateOffer();  // Calls Go backend
  set({ status: "waiting-for-answer", offerToken: token });
}
```

3. **Go Backend** (`app.go:81-148`):
```go
func (a *App) CreateOffer() (string, error) {
    config := webrtc.Configuration{
        ICEServers: []webrtc.ICEServer{
            {URLs: []string{"stun:stun.l.google.com:19302"}},
        },
    }
    peerConnection, _ := webrtc.NewPeerConnection(config)
    a.peerConnection = peerConnection
    
    // Creates the DataChannel - joiner will RECEIVE this
    dataChannel, _ := peerConnection.CreateDataChannel("minecraft", nil)
    
    // Sets up what happens when channel opens
    dataChannel.OnOpen(func() {
        a.safeEventEmit("status-change", "connected")
        go a.pumpMinecraftToChannel(dataChannel)  // Bridges to MC server
    })
    
    offer, _ := peerConnection.CreateOffer(nil)
    peerConnection.SetLocalDescription(offer)
    
    // Wait for ICE candidates to be gathered
    <-webrtc.GatheringCompletePromise(peerConnection)
    
    // Return base64-encoded SDP
    offerJson, _ := json.Marshal(peerConnection.LocalDescription())
    return base64.StdEncoding.EncodeToString(offerJson), nil
}
```

**Result:** Host's PeerConnection has a DataChannel waiting to open. The `OnOpen` callback is registered but won't fire until the connection completes.

### Section 2: Joiner Accepts Offer (INCOMPLETE)

**What the user does:** Opens app, clicks "Join", pastes offer token, clicks "Connect"

**Code path:**

1. **Frontend** (`routes/join/index.tsx:54-61`):
```typescript
const handlePasteOffer = async () => {
  await acceptOffer(offerInput);  // Calls store
  setOfferInput("");
};
```

2. **Frontend** (`tunnelStore.ts:70-89`):
```typescript
acceptOffer: async (offer) => {
  set({ status: "connecting", logs: [] });
  const answer = await AcceptOffer(offer);  // Calls Go backend
  set({ status: "waiting-for-host", answerToken: answer });
}
```

3. **Go Backend** (`app.go:175-243`):
```go
func (a *App) AcceptOffer(offerToken string) (string, error) {
    sdpBytes, _ := base64.StdEncoding.DecodeString(offerToken)
    var offer webrtc.SessionDescription
    json.Unmarshal(sdpBytes, &offer)
    
    config := webrtc.Configuration{...}
    peerConnection, _ := webrtc.NewPeerConnection(config)
    a.peerConnection = peerConnection
    
    // Set the host's SDP as remote description
    peerConnection.SetRemoteDescription(offer)
    
    // Create answer
    answer, _ := peerConnection.CreateAnswer(nil)
    peerConnection.SetLocalDescription(answer)
    
    // Wait for ICE gathering
    <-webrtc.GatheringCompletePromise(peerConnection)
    
    // Return base64-encoded answer
    answerJson, _ := json.Marshal(peerConnection.LocalDescription())
    return base64.StdEncoding.EncodeToString(answerJson), nil
    
    // MISSING: peerConnection.OnDataChannel() handler!
}
```

**The gap:** The host created a DataChannel named "minecraft" in `CreateOffer()`. When the connection establishes, WebRTC will deliver that channel to the joiner via `OnDataChannel`. But this callback is never registered.

### Section 3: Host Accepts Answer (WORKING)

**What the user does:** Host pastes the answer token from friend, clicks "Connect"

**Code path:**

1. **Frontend** (`routes/host/index.tsx:135-140`):
```typescript
<Button onClick={() => {
  const input = document.getElementById("answer-input") as HTMLInputElement;
  if (input.value) acceptAnswer(input.value);
}}>
  Connect
</Button>
```

2. **Frontend** (`tunnelStore.ts:91-109`):
```typescript
acceptAnswer: async (answer) => {
  await AcceptAnswer(answer);  // Calls Go backend
  set({ status: "connected" });  // Premature! Not connected yet
  get().addLog("Tunnel established!");
}
```

3. **Go Backend** (`app.go:150-173`):
```go
func (a *App) AcceptAnswer(answerToken string) error {
    sdpBytes, _ := base64.StdEncoding.DecodeString(answerToken)
    var answer webrtc.SessionDescription
    json.Unmarshal(sdpBytes, &answer)
    
    // Set the joiner's SDP as remote description
    a.peerConnection.SetRemoteDescription(answer)
    
    return nil
    // Note: This completes the handshake, but ICE negotiation
    // happens asynchronously after this returns
}
```

**What happens next:** After `AcceptAnswer()` returns, WebRTC begins ICE negotiation. When ICE completes, the DataChannel transitions to `open` state and the host's `OnOpen` callback fires.

### Section 4: DataChannel Opens - Traffic Flow

#### Host Side (WORKING)

When the DataChannel opens, the host's callback fires (`app.go:114-118`):

```go
dataChannel.OnOpen(func() {
    a.safeEventEmit("status-change", "connected")
    a.safeEventEmit("log", "P2P Tunnel Established!")
    go a.pumpMinecraftToChannel(dataChannel)
})
```

`pumpMinecraftToChannel()` (`app.go:246-275`):
```go
func (a *App) pumpMinecraftToChannel(dc *webrtc.DataChannel) {
    // Connect to local Minecraft server
    mcConn, _ := DialTimeout("tcp", "localhost:25565", TimeoutTCPConnect)
    defer mcConn.Close()

    // Minecraft -> WebRTC
    go func() {
        buf := make([]byte, 1500)
        for {
            n, _ := mcConn.Read(buf)
            dc.Send(buf[:n])  // Send to joiner
        }
    }()

    // WebRTC -> Minecraft
    dc.OnMessage(func(msg webrtc.DataChannelMessage) {
        mcConn.Write(msg.Data)  // Receive from joiner
    })

    select {}  // Block forever
}
```

#### Joiner Side (MISSING)

**What should happen:**
```go
// In AcceptOffer(), add:
peerConnection.OnDataChannel(func(dc *webrtc.DataChannel) {
    dc.OnOpen(func() {
        a.safeEventEmit("status-change", "connected")
        go a.StartJoinerProxy(dc, "25565")
    })
})
```

`StartJoinerProxy()` already exists (`app.go:305-331`):
```go
func (a *App) StartJoinerProxy(dc *webrtc.DataChannel, port string) error {
    listener, _ := ListenTimeout("tcp", ":"+port, TimeoutNetwork)
    
    go func() {
        for {
            conn, _ := listener.Accept()
            go a.handleJoinerConnection(conn, dc)
        }
    }()
    
    return nil
}
```

This:
1. Listens on `localhost:25565`
2. When Minecraft client connects, bridges traffic to DataChannel

## The Gap

| Component | Host | Joiner |
|-----------|------|--------|
| Create PeerConnection | ✅ `CreateOffer()` | ✅ `AcceptOffer()` |
| Create/set DataChannel | ✅ `CreateDataChannel()` | ❌ Missing `OnDataChannel` |
| OnOpen handler | ✅ `pumpMinecraftToChannel()` | ❌ Never called |
| Local proxy | ✅ Connect to `localhost:25565` | ❌ Never starts |

**Root cause:** `AcceptOffer()` does not register `OnDataChannel` callback, so when the host's DataChannel arrives, the joiner has no code to handle it.

## The Fix

Add to `AcceptOffer()` in `app.go`, after creating the PeerConnection:

```go
func (a *App) AcceptOffer(offerToken string) (string, error) {
    // ... existing setup code ...
    
    peerConnection, err := webrtc.NewPeerConnection(config)
    if err != nil {
        return "", err
    }
    a.peerConnection = peerConnection
    
    // ADD THIS: Handle incoming DataChannel from host
    peerConnection.OnDataChannel(func(dc *webrtc.DataChannel) {
        dc.OnOpen(func() {
            a.safeEventEmit("status-change", "connected")
            a.safeEventEmit("log", "P2P Tunnel Established!")
            go a.StartJoinerProxy(dc, a.proxyPort)  // Need to store proxyPort in App
        })
        
        dc.OnClose(func() {
            a.safeEventEmit("status-change", "disconnected")
            a.safeEventEmit("log", "Connection closed")
        })
        
        dc.OnError(func(err error) {
            a.safeEventEmit("status-change", "error")
            a.safeEventEmit("log", fmt.Sprintf("DataChannel error: %v", err))
        })
    })
    
    // ... rest of existing code ...
}
```

### Additional Changes Needed

1. **Store proxyPort in App struct** - The joiner needs to know which port to listen on (default: 25565)

2. **Update frontend** - The joiner's `proxyPort` state exists in tunnelStore but isn't used

3. **Consider configurable port** - Allow joiner to specify which local port to listen on (in case 25565 is in use)

## Testing the Fix

1. **Unit test**: Create offer, accept offer, verify `OnDataChannel` is called
2. **Integration test**: Host creates offer, joiner accepts, verify DataChannel opens on both sides
3. **E2E test**: Run two app instances, exchange tokens, verify Minecraft traffic flows

---

## Edge Case Analysis

### 1. Connection Failures

| Scenario | Current Handling | Issue |
|----------|------------------|-------|
| ICE timeout | ✅ Returns error after 30s | None |
| MC server offline (host) | ⚠️ Logs error, returns | No status change to "error" |
| Port in use (joiner) | ⚠️ Returns error | Handler not wired |
| Invalid token | ✅ Returns descriptive error | None |

**Fix for MC server offline:**
```go
// In pumpMinecraftToChannel(), after connection failure:
if err != nil {
    a.safeEventEmit("status-change", "error")
    a.safeEventEmit("log", fmt.Sprintf("Error connecting to Minecraft server: %v", err))
    return
}
```

### 2. Connection Lifecycle (GAPS)

| Scenario | Current Handling | Issue |
|----------|------------------|-------|
| Peer disconnects | ❌ Nothing | No `OnClose` on DataChannel |
| Connection state changes | ❌ Nothing | No `OnConnectionStateChange` |
| App shutdown | ⚠️ Closes PC only | TCP connections leak, listener leaks |

**Fix: Add connection state handlers**

For host (in `CreateOffer()`):
```go
dataChannel.OnClose(func() {
    a.safeEventEmit("status-change", "disconnected")
    a.safeEventEmit("log", "DataChannel closed")
})

peerConnection.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
    if state == webrtc.PeerConnectionStateDisconnected || 
       state == webrtc.PeerConnectionStateFailed {
        a.safeEventEmit("status-change", "error")
        a.safeEventEmit("log", fmt.Sprintf("Connection %s", state))
    }
})
```

For joiner (in `AcceptOffer()` - same pattern).

### 3. Multiple Connections (CRITICAL BUG)

**Issue in `handleJoinerConnection()` (app.go:349-351):**

```go
dc.OnMessage(func(msg webrtc.DataChannelMessage) {
    conn.Write(msg.Data)  // ← Overwrites previous handler!
})
```

If two Minecraft clients connect to the joiner:
- Both call `dc.OnMessage()`
- The **last one wins** - only one client receives data
- The other client's traffic still goes out but nothing comes back

**Fix: Single OnMessage handler with connection tracking**

```go
type App struct {
    ctx            context.Context
    cancel         context.CancelFunc
    peerConnection *webrtc.PeerConnection
    joinerConns    map[net.Conn]struct{}  // Track active connections
    joinerConnMu   sync.Mutex
}

func (a *App) StartJoinerProxy(dc *webrtc.DataChannel, port string) error {
    a.joinerConns = make(map[net.Conn]struct{})
    
    // Set up single message handler that broadcasts to all connections
    dc.OnMessage(func(msg webrtc.DataChannelMessage) {
        a.joinerConnMu.Lock()
        defer a.joinerConnMu.Unlock()
        for conn := range a.joinerConns {
            conn.Write(msg.Data)
        }
    })
    
    listener, _ := ListenTimeout("tcp", ":"+port, TimeoutNetwork)
    // ... accept loop ...
}

func (a *App) handleJoinerConnection(conn net.Conn, dc *webrtc.DataChannel) {
    a.joinerConnMu.Lock()
    a.joinerConns[conn] = struct{}{}
    a.joinerConnMu.Unlock()
    
    defer func() {
        conn.Close()
        a.joinerConnMu.Lock()
        delete(a.joinerConns, conn)
        a.joinerConnMu.Unlock()
    }()
    
    // Only read from conn -> dc, no OnMessage here
    buf := make([]byte, 4096)
    for {
        n, err := conn.Read(buf)
        if err != nil {
            return
        }
        dc.Send(buf[:n])
    }
}
```

### 4. Resource Leaks

| Resource | Issue | Fix |
|----------|-------|-----|
| `select {}` in `pumpMinecraftToChannel` | Blocks forever, even if TCP closes | Use channel to signal shutdown |
| Listener in `StartJoinerProxy` | Never closed, no reference stored | Store in App, close on shutdown |
| Old PeerConnection | Calling `CreateOffer()` twice leaks first | Close existing before creating new |

**Fix: Store listener and close on shutdown**

```go
type App struct {
    // ... existing fields ...
    listener net.Listener
}

func (a *App) shutdown(ctx context.Context) {
    if a.cancel != nil {
        a.cancel()
    }
    if a.listener != nil {
        a.listener.Close()
    }
    if a.peerConnection != nil {
        a.peerConnection.Close()
        a.peerConnection = nil
    }
}
```

**Fix: Check for existing PeerConnection**

```go
func (a *App) CreateOffer() (string, error) {
    // Close existing connection if any
    if a.peerConnection != nil {
        a.peerConnection.Close()
        a.peerConnection = nil
    }
    // ... rest of function ...
}
```

### 5. Context Cancellation

Goroutines in `pumpMinecraftToChannel()` and `StartJoinerProxy()` don't check context.

**Fix: Use context for graceful shutdown**

```go
func (a *App) pumpMinecraftToChannel(dc *webrtc.DataChannel) {
    mcConn, err := DialTimeout("tcp", "localhost:25565", TimeoutTCPConnect)
    if err != nil {
        a.safeEventEmit("status-change", "error")
        a.safeEventEmit("log", fmt.Sprintf("Error: %v", err))
        return
    }
    defer mcConn.Close()

    done := make(chan struct{})
    defer close(done)

    go func() {
        buf := make([]byte, 1500)
        for {
            select {
            case <-done:
                return
            default:
                n, err := mcConn.Read(buf)
                if err != nil {
                    return
                }
                dc.Send(buf[:n])
            }
        }
    }()

    select {
    case <-a.ctx.Done():
        a.safeEventEmit("log", "Shutting down tunnel")
    }
}
```

---

## Implementation Priority

### Critical (must fix for MVP)

1. **Add `OnDataChannel` handler** - The joiner won't work without this
2. **Fix OnMessage race** - Multiple clients will cause data loss

### High Priority (should fix before release)

3. Add `OnClose` and `OnConnectionStateChange` handlers
4. Store and close listener on shutdown
5. Handle MC server connection failure with status update

### Medium Priority (nice to have)

6. Use context for goroutine cancellation
7. Check for existing PeerConnection before creating new
8. Add reconnect capability

---

## Implementation Plan

### Phase 1: Critical Fixes (MVP)

**Task 1.1: Add OnDataChannel handler to AcceptOffer()**

File: `app.go`

Changes:
```go
func (a *App) AcceptOffer(offerToken string) (string, error) {
    // ... existing code ...
    
    peerConnection, err := webrtc.NewPeerConnection(config)
    if err != nil {
        return "", err
    }
    a.peerConnection = peerConnection
    
    // NEW: Handle incoming DataChannel from host
    peerConnection.OnDataChannel(func(dc *webrtc.DataChannel) {
        dc.OnOpen(func() {
            a.safeEventEmit("status-change", "connected")
            a.safeEventEmit("log", "P2P Tunnel Established!")
            go a.StartJoinerProxy(dc, "25565")
        })
        
        dc.OnClose(func() {
            a.safeEventEmit("status-change", "disconnected")
            a.safeEventEmit("log", "Connection closed")
        })
    })
    
    // ... rest of existing code ...
}
```

**Task 1.2: Fix OnMessage race condition in StartJoinerProxy()**

File: `app.go`

Changes:
1. Add `joinerConns` map and mutex to App struct
2. Move `OnMessage` to `StartJoinerProxy()` (single handler)
3. Broadcast messages to all connections
4. Track connections in `handleJoinerConnection()`

```go
type App struct {
    ctx            context.Context
    cancel         context.CancelFunc
    peerConnection *webrtc.PeerConnection
    joinerConns    map[net.Conn]struct{}
    joinerConnMu   sync.Mutex
    listener       net.Listener
}

func (a *App) StartJoinerProxy(dc *webrtc.DataChannel, port string) error {
    a.joinerConns = make(map[net.Conn]struct{})
    
    // Single message handler that broadcasts to all connections
    dc.OnMessage(func(msg webrtc.DataChannelMessage) {
        a.joinerConnMu.Lock()
        defer a.joinerConnMu.Unlock()
        for conn := range a.joinerConns {
            conn.Write(msg.Data)
        }
    })
    
    listener, err := ListenTimeout("tcp", ":"+port, TimeoutNetwork)
    if err != nil {
        return fmt.Errorf("failed to listen on port %s: %w", port, err)
    }
    a.listener = listener
    
    go func() {
        for {
            conn, err := listener.Accept()
            if err != nil {
                return
            }
            go a.handleJoinerConnection(conn, dc)
        }
    }()
    
    return nil
}

func (a *App) handleJoinerConnection(conn net.Conn, dc *webrtc.DataChannel) {
    a.joinerConnMu.Lock()
    a.joinerConns[conn] = struct{}{}
    a.joinerConnMu.Unlock()
    
    defer func() {
        conn.Close()
        a.joinerConnMu.Lock()
        delete(a.joinerConns, conn)
        a.joinerConnMu.Unlock()
    }()
    
    buf := make([]byte, 4096)
    for {
        n, err := conn.Read(buf)
        if err != nil {
            return
        }
        dc.Send(buf[:n])
    }
}
```

### Phase 2: High Priority Fixes

**Task 2.1: Add connection state handlers (host side)**

File: `app.go`

In `CreateOffer()`, add after DataChannel creation:
```go
dataChannel.OnClose(func() {
    a.safeEventEmit("status-change", "disconnected")
    a.safeEventEmit("log", "DataChannel closed")
})

peerConnection.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
    switch state {
    case webrtc.PeerConnectionStateConnected:
        // Already handled by OnOpen
    case webrtc.PeerConnectionStateDisconnected:
        a.safeEventEmit("status-change", "disconnected")
        a.safeEventEmit("log", "Peer disconnected")
    case webrtc.PeerConnectionStateFailed:
        a.safeEventEmit("status-change", "error")
        a.safeEventEmit("log", "Connection failed")
    }
})
```

**Task 2.2: Add connection state handlers (joiner side)**

File: `app.go`

In `AcceptOffer()`, add same pattern after PeerConnection creation.

**Task 2.3: Close listener on shutdown**

File: `app.go`

```go
func (a *App) shutdown(ctx context.Context) {
    if a.cancel != nil {
        a.cancel()
    }
    if a.listener != nil {
        a.listener.Close()
        a.listener = nil
    }
    if a.peerConnection != nil {
        a.peerConnection.Close()
        a.peerConnection = nil
    }
}
```

**Task 2.4: Emit error status when MC server offline**

File: `app.go`

In `pumpMinecraftToChannel()`:
```go
mcConn, err := DialTimeout("tcp", "localhost:25565", TimeoutTCPConnect)
if err != nil {
    a.safeEventEmit("status-change", "error")  // ADD THIS
    a.safeEventEmit("log", fmt.Sprintf("Error connecting to Minecraft server: %v", err))
    return
}
```

### Phase 3: Medium Priority (Polish)

**Task 3.1: Check for existing PeerConnection**

In `CreateOffer()` and `AcceptOffer()`:
```go
if a.peerConnection != nil {
    a.peerConnection.Close()
    a.peerConnection = nil
}
```

**Task 3.2: Use context for graceful shutdown**

This is more involved - requires refactoring the pump functions to select on `ctx.Done()`.

### Testing Plan

1. **Unit test**: `AcceptOffer` sets `OnDataChannel` handler
2. **Unit test**: `StartJoinerProxy` creates listener and single `OnMessage` handler
3. **Integration test**: Full offer/answer exchange, verify DataChannel opens both sides
4. **Manual test**: Two app instances, exchange tokens, connect Minecraft client

### Files Changed

| File | Changes |
|------|---------|
| `app.go` | All fixes (OnDataChannel, OnMessage refactor, state handlers, shutdown) |
| `frontend/src/lib/tunnelStore.ts` | Already has `proxyPort` state, may need to pass to backend |

### Estimated Effort

| Phase | Tasks | Estimate |
|-------|-------|----------|
| Phase 1 | 1.1, 1.2 | 30-60 min |
| Phase 2 | 2.1-2.4 | 30 min |
| Phase 3 | 3.1, 3.2 | 30 min |
| Testing | Unit + manual | 30 min |
| **Total** | | **2-2.5 hours** |
