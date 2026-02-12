# timeout.go

Last Updated: 2026-02-12T19:00:00Z

## Purpose

Provides timeout constants and generic timeout wrapper functions for network operations, preventing indefinite hangs during WebRTC ICE gathering, TCP connections, and file I/O.

## Stage-Actor-Prop Overview

The timeout context is the Stage that constrains execution time, the generic `RunWithTimeout` function is the Actor that enforces limits, and the wrapped operations are Props with bounded lifetimes.

## Components

### Constants
- `TimeoutWebRTCICE` (30s) - ICE candidate gathering duration
- `TimeoutTCPConnect` (10s) - TCP connection establishment
- `TimeoutTCPOperation` (5s) - Individual TCP operations
- `TimeoutFileIO` (5s) - File read/write operations
- `TimeoutNetwork` (10s) - Network listener setup

Define maximum allowable durations for various I/O operations.

### `RunWithTimeout[T any](operation string, timeout time.Duration, fn func() (T, error))` → (T, error)
- **Stage**: Goroutine with timeout channel
- **Actor**: Generic timeout enforcer
- **Props**: Operation name, timeout duration, wrapped function

Generic wrapper that executes any function returning `(T, error)` with a timeout guard. Uses channels to communicate result, error, or timeout.

### `DialTimeout(network, address string, timeout time.Duration)` → (net.Conn, error)
- **Stage**: TCP connection attempt
- **Actor**: Timeout-protected dialer
- **Props**: Network type, address, timeout

Wraps `net.DialTimeout` with generic timeout protection.

### `ListenTimeout(network, address string, timeout time.Duration)` → (net.Listener, error)
- **Stage**: Network listener creation
- **Actor**: Timeout-protected listener
- **Props**: Network type, address, timeout

Wraps `net.Listen` with timeout protection for listener setup.

### `WithTimeoutContext(parent context.Context, timeout time.Duration)` → (context.Context, context.CancelFunc)
- **Stage**: Context hierarchy
- **Actor**: Context factory
- **Props**: Parent context, timeout duration

Creates a child context with automatic cancellation after timeout elapses.

## Usage

```go
// Connect to Minecraft server with timeout
conn, err := DialTimeout("tcp", "localhost:25565", TimeoutTCPConnect)

// Create listener with timeout
listener, err := ListenTimeout("tcp", ":25565", TimeoutNetwork)

// Generic timeout for custom operation
result, err := RunWithTimeout("custom-op", 5*time.Second, func() (string, error) {
    return doSomething()
})
```

## Dependencies

- Go standard library: `context`, `fmt`, `net`, `time`

## Notes

- Uses generics for type-safe timeout wrapping (Go 1.18+)
- Timeout operations use `time.After()` pattern with select
- Zero value returned on timeout (handled by caller)
- Used throughout app.go to prevent indefinite blocking
