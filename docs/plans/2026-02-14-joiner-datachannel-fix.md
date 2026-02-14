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
