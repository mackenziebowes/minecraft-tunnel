# WebRTC P2P Minecraft Tunnel Design

Date: 2025-02-12

## Overview

WebRTC-based P2P tunnel enabling Minecraft server hosting through NAT/firewall without port forwarding. Host exposes local Minecraft server to joiner via encrypted data channel.

## Architecture

### Roles

**Host (Server Side):**
1. Generates WebRTC Offer (SDP) → Encodes as base64 token
2. Shares token via copy/paste, file export, or deeplink
3. Forwards traffic from local Minecraft server (`localhost:25565`) through WebRTC data channel
4. Accepts Answer from joiner → Completes handshake → Tunnel active

**Joiner (Client Side):**
1. Imports/pastes offer → Auto-creates WebRTC Answer
2. Displays answer token for user to share back
3. Listens on `localhost:25565` as local proxy for Minecraft client
4. Forwards all traffic through WebRTC data channel to host

### Tech Stack

**Go Backend:**
- `pion/webrtc` - WebRTC implementation
- Standard `net` package - TCP proxying
- Wails - System integration (file dialogs, protocol registration)

**Frontend:**
- React 18.3 - UI components
- TypeScript 5.7 - Type safety
- Tailwind CSS v4 - Styling
- shadcn/ui - Component library
- Zustand - State management

## Components

### Go Backend

**`PeerConnectionManager`**
- `CreateOffer()` → base64 SDP token
- `AcceptOffer(offerToken)` → generates and returns answer
- `AcceptAnswer(answerToken)` → completes handshake
- `Close()` - cleanup resources

**`TunnelProxy`**
- Host mode: Connects to `localhost:25565`, pumps bi-directional to data channel
- Joiner mode: Listens on `localhost:25565`, accepts MC client connections
- Handles connection errors and reconnection attempts

**`TokenManager`**
- `ExportToFile(token)` - writes `.mc-tunnel-invite`
- `ImportFromFile()` - reads and validates tokens
- `GenerateDeeplink(token)` - creates `minecraft-tunnel://` URLs

### Frontend

**HostView** (`/host`)
- Input: Minecraft server address (default `localhost:25565`)
- Action: "Generate Invitation" button
- Output: Token display with copy button, export file button
- Status: Waiting for Answer → Connected with peer count
- Logs: Real-time event feed

**JoinView** (`/join`)
- Input: Paste token, file import, or deeplink detection
- Auto-action: Generates answer immediately
- Output: Answer token display with copy button
- Instructions: "Share this back with host"
- Status: Connecting → Connected

**Shared Components**
- `TokenCard` - Displays token with copy/export actions
- `ConnectionStatus` - Badge with state (waiting/connected/error)
- `LogViewer` - Scrollable event log with timestamps

## Data Flow

### Connection Handshake

1. **Host:** "Generate Invitation" → `CreateOffer()` → WebRTC offer SDP → base64 token
2. **Host:** Shares token → Joiner receives
3. **Joiner:** Imports token → `AcceptOffer(token)` → Generates answer → Displays
4. **Joiner:** Shares answer back → Host receives
5. **Host:** `AcceptAnswer(answer)` → ICE negotiation → Data channel opens

### Active Tunnel (Host → Joiner)

1. Minecraft client connects to joiner's `localhost:25565`
2. Joiner's `TunnelProxy` accepts TCP connection
3. MC client packets → Joiner reads TCP → `dataChannel.Send()` over WebRTC
4. Host receives on data channel → Writes to local MC server
5. MC server responds → Host reads TCP → `dataChannel.Send()` back
6. Joiner receives → Writes to MC client TCP socket

### State Management

- Frontend: `tunnelStore` tracks status (`disconnected`/`connecting`/`connected`/`error`), logs array, current token
- Backend: Holds `peerConnection` reference, active TCP connections
- Wails events: `log` (message), `status-change` (state), `peer-connected` (count)

## Error Handling

### Connection Failures

- ICE negotiation timeout (30s): Error message "Connection timed out - ensure both are online"
- Peer disconnected: `dataChannel.OnClose` → status `disconnected`, cleanup state
- STUN server unreachable: Emit error with fallback STUN list
- Invalid token format: UI message "Invalid invitation format"

### TCP Proxy Errors

- Host: Minecraft server not running → Error "Cannot connect to Minecraft server - is it running?"
- Joiner: Port `25565` in use → Error "Port already in use - close other apps"
- TCP read/write errors: Log with timestamp, graceful close
- Connection drops: Cleanup resources, return to `disconnected` state

### Token Handling

- Corrupted base64: Display "Failed to decode token"
- Invalid SDP: "Invalid WebRTC session description"
- File import errors: "Cannot read file"
- Deeplink malformed: Log error, stay on current view

### Frontend UX

- All errors in log viewer with red highlight
- Non-fatal errors as toast notifications
- Fatal errors reset to initial state
- Retry buttons where appropriate
- Loading spinners during long operations

### Backend Resilience

- Exported functions return errors
- Context cancellation checks in goroutines
- `defer` cleanup on peer connections and TCP sockets
- Recover from panics with log emission

## Testing Strategy

### Go Backend Tests

- `webrtc_test.go`: Mock WebRTC connections, test offer/answer exchange
- `tunnel_proxy_test.go`: TCP proxy with mock sockets, test byte forwarding
- `token_test.go`: Base64 encode/decode, validation, file import/export
- Integration: Local host ↔ joiner pair in same process

### Frontend Tests (Bun)

- Component tests: HostView token generation, JoinView answer flow, copy buttons
- Store tests: Zustand state transitions, error handling, log accumulation
- Integration: Wails event emission → store updates

### Test Infrastructure

- Go: Standard `testing` package, table-driven tests
- Frontend: Bun test runner, `@testing-library/react`
- Mock Wails runtime for isolated frontend tests
- Test data: Sample valid/invalid tokens, mock SDP responses

### Manual Testing

1. Happy path: Host offer → Joiner accept → Answer shared → Connected → MC traffic flows
2. Token exchange: Copy/paste, file export/import, deeplink URL
3. Error cases: Invalid token, MC server offline, port in use, connection drop
4. Reconnection: Kill one side, restart, verify cleanup

### End-to-End

- Scripted: Launch two app instances, automate token exchange, verify connection
- Network conditions: Simulate packet loss with `tc` (Linux)
- Minecraft integration: Standalone MC server + client, verify gameplay

## Connection Lifecycle

### Tokens: Ephemeral

- Connection lost or app restart → Generate new tokens, reshare
- No persistence to disk
- Simplifies security model

### Exchange: Auto-Answer, Manual Reply

- Host shares offer → Joiner auto-generates answer → Joiner shares answer back
- 2 tokens total (offer + answer)
- Manual sharing but automatic generation
