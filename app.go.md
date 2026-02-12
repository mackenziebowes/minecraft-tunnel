# app.go

Last Updated: 2026-02-12T19:00:00Z

## Purpose

Core application logic for P2P Minecraft tunnel using WebRTC. Manages peer connections, handles WebRTC signaling (offer/answer exchange), and proxies Minecraft traffic between local server/client and remote peers.

## Stage-Actor-Prop Overview

The WebRTC PeerConnection is the Stage, the App struct acts as the Director managing tunnel lifecycle, and data flows (Minecraft packets) are the Props being bidirectionally pumped between local sockets and WebRTC data channels.

## Components

### `App` struct
- **Stage**: Holds context and peer connection
- **Actor**: Coordinates WebRTC handshake and proxying
- **Props**: Context, PeerConnection, cancel function

Main application state container. Tracks WebRTC connection and provides exported methods for frontend.

### `CreateOffer()` → (string, error)
- **Stage**: WebRTC ICE gathering process
- **Actor**: Host peer initiates connection
- **Props**: Base64-encoded SDP offer token

Generates WebRTC offer and data channel, waits for ICE gathering to collect network paths, returns base64-encoded offer for sharing with joiner.

### `AcceptAnswer(answerToken string)` → error
- **Stage**: WebRTC connection establishment
- **Actor**: Host peer accepts joiner's response
- **Props**: Base64-encoded answer token

Decodes and applies joiner's answer to complete P2P connection.

### `AcceptOffer(offerToken string)` → (string, error)
- **Stage**: WebRTC handshake response
- **Actor**: Joiner peer responds to host
- **Props**: Base64-encoded answer token

Decodes host's offer, creates WebRTC connection, returns answer token.

### `pumpMinecraftToChannel(dc *webrtc.DataChannel)`
- **Stage**: Bidirectional data pipe
- **Actor**: Goroutine coordinator
- **Props**: TCP socket + WebRTC data channel

Forwards data between local Minecraft server (localhost:25565) and WebRTC tunnel. Runs two goroutines: MC→WebRTC and WebRTC→MC.

### `StartHostProxy(dc *webrtc.DataChannel, targetAddress string)` → error
- **Stage**: Host-side proxy connection
- **Actor**: Proxy coordinator
- **Props**: Target Minecraft server address

Connects to specified Minecraft server and proxies traffic through WebRTC to joiner.

### `StartJoinerProxy(dc *webrtc.DataChannel, port string)` → error
- **Stage**: Joiner-side proxy listener
- **Actor**: Proxy listener
- **Props**: Local port for Minecraft clients

Listens on local port, accepts Minecraft client connections, and proxies them through WebRTC to host.

### `ExportToFile(token string, filepath string)` → error
- **Stage**: File system I/O
- **Actor**: Token persistence helper
- **Props**: Token content + target path

Saves token to file with timeout protection.

### `ImportFromFile(filepath string)` → (string, error)
- **Stage**: File system I/O
- **Actor**: Token retrieval helper
- **Props**: Source file path

Reads token from file with timeout protection.

## Usage

```go
// Host workflow:
offer, err := app.CreateOffer()  // share this string
err := app.AcceptAnswer(answerFromFriend)  // paste friend's response

// Joiner workflow:
answer, err := app.AcceptOffer(offerFromFriend)  // paste friend's offer, share response
```

## Dependencies

- `github.com/pion/webrtc/v3` - WebRTC implementation
- `github.com/wailsapp/wails/v2/pkg/runtime` - Event emission to frontend
- `timeout.go` - Network and file I/O timeout constants

## Notes

- Uses Google's public STUN server for NAT traversal
- Data channel named "minecraft"
- All file/network operations protected by timeouts from timeout.go
- `safeEventEmit` prevents crashes when context is nil or in test mode
- Proxy buffers: 1500 bytes (host→MC), 4096 bytes (joiner side)
- Status changes and logs emitted to frontend via Wails events
