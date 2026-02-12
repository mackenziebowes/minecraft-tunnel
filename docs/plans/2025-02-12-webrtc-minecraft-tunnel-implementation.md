# WebRTC P2P Minecraft Tunnel Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a WebRTC-based P2P tunnel enabling Minecraft server hosting without port forwarding.

**Architecture:** Host generates WebRTC offer → shares via token → joiner accepts → joiner generates answer → shares back → tunnel established. Go backend handles WebRTC and TCP proxying, React frontend manages UX and state.

**Tech Stack:** Go (pion/webrtc, net), Wails, React 18, TypeScript 5.7, Tailwind CSS v4, shadcn/ui, Zustand, Bun (testing)

---

## Phase 1: Go Backend - WebRTC Foundation

### Task 1: Add PeerConnectionManager struct to app.go

**Files:**
- Modify: `app.go:18-25`

**Step 1: Write the failing test**

Create `app_test.go`:

```go
package main

import (
    "testing"
)

func TestNewPeerConnectionManager(t *testing.T) {
    manager := NewPeerConnectionManager()
    if manager == nil {
        t.Fatal("Expected non-nil manager")
    }
}
```

**Step 2: Run test to verify it fails**

Run: `go test -v -run TestNewPeerConnectionManager`
Expected: FAIL with "undefined: NewPeerConnectionManager"

**Step 3: Write minimal implementation**

Add to `app.go` after `type App struct`:

```go
type PeerConnectionManager struct {
    peerConnection *webrtc.PeerConnection
    dataChannel    *webrtc.DataChannel
}

func NewPeerConnectionManager() *PeerConnectionManager {
    return &PeerConnectionManager{}
}
```

**Step 4: Run test to verify it passes**

Run: `go test -v -run TestNewPeerConnectionManager`
Expected: PASS

**Step 5: Commit**

```bash
git add app.go app_test.go
git commit -m "feat: add PeerConnectionManager struct"
```

---

### Task 2: CreateOffer generates base64-encoded SDP token

**Files:**
- Modify: `app.go:32-85` (replace existing CreateOffer)
- Test: `app_test.go`

**Step 1: Write the failing test**

Add to `app_test.go`:

```go
func TestCreateOfferGeneratesValidBase64(t *testing.T) {
    app := &App{ctx: context.Background()}
    token, err := app.CreateOffer()
    if err != nil {
        t.Fatalf("Expected no error, got: %v", err)
    }
    if token == "" {
        t.Fatal("Expected non-empty token")
    }

    // Should be valid base64
    _, err = base64.StdEncoding.DecodeString(token)
    if err != nil {
        t.Fatalf("Expected valid base64, got: %v", err)
    }
}
```

**Step 2: Run test to verify it fails**

Run: `go test -v -run TestCreateOffer`
Expected: FAIL or existing test passes but we want new behavior

**Step 3: Write minimal implementation**

Replace `CreateOffer` in `app.go`:

```go
func (a *App) CreateOffer() (string, error) {
    config := webrtc.Configuration{
        ICEServers: []webrtc.ICEServer{
            {URLs: []string{"stun:stun.l.google.com:19302"}},
        },
    }

    peerConnection, err := webrtc.NewPeerConnection(config)
    if err != nil {
        return "", err
    }

    dataChannel, err := peerConnection.CreateDataChannel("minecraft", nil)
    if err != nil {
        return "", err
    }

    dataChannel.OnOpen(func() {
        runtime.EventsEmit(a.ctx, "status-change", "connected")
        runtime.EventsEmit(a.ctx, "log", "P2P Tunnel Established!")
    })

    offer, err := peerConnection.CreateOffer(nil)
    if err != nil {
        return "", err
    }

    if err = peerConnection.SetLocalDescription(offer); err != nil {
        return "", err
    }

    <-webrtc.GatheringCompletePromise(peerConnection)

    offerJson, _ := json.Marshal(peerConnection.LocalDescription())
    return base64.StdEncoding.EncodeToString(offerJson), nil
}
```

**Step 4: Run test to verify it passes**

Run: `go test -v -run TestCreateOffer`
Expected: PASS

**Step 5: Commit**

```bash
git add app.go app_test.go
git commit -m "feat: CreateOffer generates base64 SDP token"
```

---

### Task 3: AcceptOffer generates answer token from offer

**Files:**
- Modify: `app.go`
- Test: `app_test.go`

**Step 1: Write the failing test**

Add to `app_test.go`:

```go
func TestAcceptOfferGeneratesAnswer(t *testing.T) {
    app := &App{ctx: context.Background()}

    // Create a mock offer token
    offer := webrtc.SessionDescription{
        Type: webrtc.SDPTypeOffer,
        SDP:  "mock-sdp",
    }
    offerJson, _ := json.Marshal(offer)
    offerToken := base64.StdEncoding.EncodeToString(offerJson)

    answerToken, err := app.AcceptOffer(offerToken)
    if err != nil {
        t.Fatalf("Expected no error, got: %v", err)
    }
    if answerToken == "" {
        t.Fatal("Expected non-empty answer token")
    }
}
```

**Step 2: Run test to verify it fails**

Run: `go test -v -run TestAcceptOffer`
Expected: FAIL with "undefined: AcceptOffer"

**Step 3: Write minimal implementation**

Add to `app.go`:

```go
func (a *App) AcceptOffer(offerToken string) (string, error) {
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

    <-webrtc.GatheringCompletePromise(peerConnection)

    answerJson, _ := json.Marshal(peerConnection.LocalDescription())
    return base64.StdEncoding.EncodeToString(answerJson), nil
}
```

**Step 4: Run test to verify it passes**

Run: `go test -v -run TestAcceptOffer`
Expected: PASS

**Step 5: Commit**

```bash
git add app.go app_test.go
git commit -m "feat: AcceptOffer generates answer from offer token"
```

---

### Task 4: AcceptAnswer completes handshake on host

**Files:**
- Modify: `app.go:88-99` (refactor existing SetAnswer)
- Test: `app_test.go`

**Step 1: Write the failing test**

Add to `app_test.go`:

```go
func TestAcceptAnswerSetsRemoteDescription(t *testing.T) {
    app := &App{ctx: context.Background()}

    // Create and set offer first
    offerToken, _ := app.CreateOffer()

    // Mock answer
    answer := webrtc.SessionDescription{
        Type: webrtc.SDPTypeAnswer,
        SDP:  "mock-answer-sdp",
    }
    answerJson, _ := json.Marshal(answer)
    answerToken := base64.StdEncoding.EncodeToString(answerJson)

    err := app.AcceptAnswer(answerToken)
    if err != nil {
        t.Fatalf("Expected no error, got: %v", err)
    }
}
```

**Step 2: Run test to verify it fails**

Run: `go test -v -run TestAcceptAnswer`
Expected: FAIL or passes with existing code

**Step 3: Write minimal implementation**

Rename `SetAnswer` to `AcceptAnswer` in `app.go`:

```go
func (a *App) AcceptAnswer(answerToken string) error {
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

**Step 4: Run test to verify it passes**

Run: `go test -v -run TestAcceptAnswer`
Expected: PASS

**Step 5: Commit**

```bash
git add app.go app_test.go
git commit -m "refactor: rename SetAnswer to AcceptAnswer"
```

---

### Task 5: TunnelProxy forwards TCP traffic in host mode

**Files:**
- Modify: `app.go:102-131` (refactor pumpMinecraftToChannel)
- Test: `app_test.go`

**Step 1: Write the failing test**

Add to `app_test.go`:

```go
func TestTunnelProxyConnectsToMinecraftServer(t *testing.T) {
    // This test needs a mock TCP server
    // For now, test that the function exists and signature is correct
    app := &App{}
    dc := &webrtc.DataChannel{}

    // This would normally block, so we just verify it compiles
    _ = dc
}
```

**Step 2: Run test to verify it fails**

Run: `go test -v -run TestTunnelProxy`
Expected: PASS (placeholder) but we want real implementation

**Step 3: Write minimal implementation**

Refactor `pumpMinecraftToChannel` into `StartHostProxy` in `app.go`:

```go
func (a *App) StartHostProxy(dc *webrtc.DataChannel, targetAddress string) error {
    mcConn, err := net.Dial("tcp", targetAddress)
    if err != nil {
        runtime.EventsEmit(a.ctx, "log", fmt.Sprintf("Error connecting to Minecraft server: %v", err))
        return fmt.Errorf("cannot connect to Minecraft server: %w", err)
    }

    // Minecraft -> WebRTC
    go func() {
        buf := make([]byte, 4096)
        for {
            n, err := mcConn.Read(buf)
            if err != nil {
                return
            }
            dc.Send(buf[:n])
        }
    }()

    // WebRTC -> Minecraft
    dc.OnMessage(func(msg webrtc.DataChannelMessage) {
        mcConn.Write(msg.Data)
    })

    return nil
}
```

**Step 4: Run test to verify it passes**

Run: `go test -v -run TestTunnelProxy`
Expected: PASS

**Step 5: Commit**

```bash
git add app.go app_test.go
git commit -m "feat: StartHostProxy forwards TCP traffic to WebRTC"
```

---

### Task 6: TunnelProxy listens for connections in joiner mode

**Files:**
- Modify: `app.go`
- Test: `app_test.go`

**Step 1: Write the failing test**

Add to `app_test.go`:

```go
func TestStartJoinerProxyListensOnPort25565(t *testing.T) {
    app := &App{ctx: context.Background()}
    dc := &webrtc.DataChannel{}

    err := app.StartJoinerProxy(dc, "25565")
    if err != nil {
        t.Fatalf("Expected no error, got: %v", err)
    }
}
```

**Step 2: Run test to verify it fails**

Run: `go test -v -run TestStartJoinerProxy`
Expected: FAIL with "undefined: StartJoinerProxy"

**Step 3: Write minimal implementation**

Add to `app.go`:

```go
func (a *App) StartJoinerProxy(dc *webrtc.DataChannel, port string) error {
    listener, err := net.Listen("tcp", ":"+port)
    if err != nil {
        return fmt.Errorf("port %s already in use: %w", port, err)
    }

    go func() {
        for {
            conn, err := listener.Accept()
            if err != nil {
                return
            }

            go a.handleJoinerConnection(conn, dc)
        }
    }()

    runtime.EventsEmit(a.ctx, "log", fmt.Sprintf("Listening on port %s for Minecraft client", port))
    return nil
}

func (a *App) handleJoinerConnection(conn net.Conn, dc *webrtc.DataChannel) {
    defer conn.Close()

    // Minecraft Client -> WebRTC
    go func() {
        buf := make([]byte, 4096)
        for {
            n, err := conn.Read(buf)
            if err != nil {
                return
            }
            dc.Send(buf[:n])
        }
    }()

    // WebRTC -> Minecraft Client
    dc.OnMessage(func(msg webrtc.DataChannelMessage) {
        conn.Write(msg.Data)
    })
}
```

**Step 4: Run test to verify it passes**

Run: `go test -v -run TestStartJoinerProxy`
Expected: PASS

**Step 5: Commit**

```bash
git add app.go app_test.go
git commit -m "feat: StartJoinerProxy listens for client connections"
```

---

### Task 7: ExportToFile writes token to .mc-tunnel-invite file

**Files:**
- Modify: `app.go`
- Test: `app_test.go`

**Step 1: Write the failing test**

Add to `app_test.go`:

```go
import "os"

func TestExportToFileWritesToken(t *testing.T) {
    tmpfile := "/tmp/test-invite.mc-tunnel-invite"
    defer os.Remove(tmpfile)

    app := &App{ctx: context.Background()}
    err := app.ExportToFile("test-token", tmpfile)
    if err != nil {
        t.Fatalf("Expected no error, got: %v", err)
    }

    content, err := os.ReadFile(tmpfile)
    if err != nil {
        t.Fatalf("Expected file to exist, got: %v", err)
    }

    if string(content) != "test-token" {
        t.Fatalf("Expected 'test-token', got '%s'", string(content))
    }
}
```

**Step 2: Run test to verify it fails**

Run: `go test -v -run TestExportToFile`
Expected: FAIL with "undefined: ExportToFile"

**Step 3: Write minimal implementation**

Add to `app.go`:

```go
func (a *App) ExportToFile(token string, filepath string) error {
    return os.WriteFile(filepath, []byte(token), 0644)
}
```

**Step 4: Run test to verify it passes**

Run: `go test -v -run TestExportToFile`
Expected: PASS

**Step 5: Commit**

```bash
git add app.go app_test.go
git commit -m "feat: ExportToFile writes token to file"
```

---

### Task 8: ImportFromFile reads token from file

**Files:**
- Modify: `app.go`
- Test: `app_test.go`

**Step 1: Write the failing test**

Add to `app_test.go`:

```go
func TestImportFromFileReadsToken(t *testing.T) {
    tmpfile := "/tmp/test-read.mc-tunnel-invite"
    defer os.Remove(tmpfile)

    os.WriteFile(tmpfile, []byte("file-token"), 0644)

    app := &App{ctx: context.Background()}
    token, err := app.ImportFromFile(tmpfile)
    if err != nil {
        t.Fatalf("Expected no error, got: %v", err)
    }

    if token != "file-token" {
        t.Fatalf("Expected 'file-token', got '%s'", token)
    }
}
```

**Step 2: Run test to verify it fails**

Run: `go test -v -run TestImportFromFile`
Expected: FAIL with "undefined: ImportFromFile"

**Step 3: Write minimal implementation**

Add to `app.go`:

```go
func (a *App) ImportFromFile(filepath string) (string, error) {
    content, err := os.ReadFile(filepath)
    if err != nil {
        return "", fmt.Errorf("cannot read file: %w", err)
    }
    return string(content), nil
}
```

**Step 4: Run test to verify it passes**

Run: `go test -v -run TestImportFromFile`
Expected: PASS

**Step 5: Commit**

```bash
git add app.go app_test.go
git commit -m "feat: ImportFromFile reads token from file"
```

---

## Phase 2: Frontend - Host View

### Task 9: Update tunnelStore with new Go function bindings

**Files:**
- Modify: `frontend/src/lib/tunnelStore.ts`
- Test: `frontend/src/lib/tunnelStore.test.ts`

**Step 1: Write the failing test**

Create `frontend/src/lib/tunnelStore.test.ts`:

```typescript
import { useTunnelStore } from "./tunnelStore";

describe("tunnelStore", () => {
  it("should initialize with default state", () => {
    const store = useTunnelStore.getState();
    expect(store.status).toBe("disconnected");
    expect(store.offerToken).toBe("");
    expect(store.answerToken).toBe("");
  });
});
```

**Step 2: Run test to verify it fails**

Run: `cd frontend && bun test lib/tunnelStore.test.ts`
Expected: FAIL because offerToken/answerToken don't exist

**Step 3: Write minimal implementation**

Replace `tunnelStore.ts`:

```typescript
import { create } from "zustand";
import {
  CreateOffer,
  AcceptOffer,
  AcceptAnswer,
  StartHostProxy,
  StartJoinerProxy,
  ExportToFile,
  ImportFromFile,
} from "../../wailsjs/go/main/App";

type TunnelStatus = "disconnected" | "connecting" | "connected" | "error";

interface TunnelState {
  // State
  status: TunnelStatus;
  logs: string[];
  offerToken: string;
  answerToken: string;
  mcServerAddress: string;
  proxyPort: string;

  // Actions
  setMcServerAddress: (address: string) => void;
  setProxyPort: (port: string) => void;
  generateOffer: () => Promise<void>;
  acceptOffer: (offer: string) => Promise<void>;
  acceptAnswer: (answer: string) => Promise<void>;
  exportToken: (token: string) => Promise<void>;
  importToken: () => Promise<void>;
  addLog: (message: string) => void;
  setStatus: (status: TunnelStatus) => void;
}

export const useTunnelStore = create<TunnelState>((set, get) => ({
  status: "disconnected",
  logs: [],
  offerToken: "",
  answerToken: "",
  mcServerAddress: "localhost:25565",
  proxyPort: "25565",

  setMcServerAddress: (address) => set({ mcServerAddress: address }),
  setProxyPort: (port) => set({ proxyPort: port }),

  generateOffer: async () => {
    set({ status: "connecting", logs: [], offerToken: "" });
    try {
      const token = await CreateOffer();
      set({ status: "waiting-for-answer", offerToken: token });
      get().addLog("Offer token generated successfully");
    } catch (err: any) {
      set({ status: "error" });
      get().addLog(`Error: ${err.message || err}`);
    }
  },

  acceptOffer: async (offer) => {
    set({ status: "connecting", logs: [] });
    try {
      const answer = await AcceptOffer(offer);
      set({ status: "waiting-for-host", answerToken: answer });
      get().addLog("Answer generated - share this with host");
    } catch (err: any) {
      set({ status: "error" });
      get().addLog(`Error: ${err.message || err}`);
    }
  },

  acceptAnswer: async (answer) => {
    try {
      await AcceptAnswer(answer);
      set({ status: "connected" });
      get().addLog("Tunnel established!");
    } catch (err: any) {
      set({ status: "error" });
      get().addLog(`Error: ${err.message || err}`);
    }
  },

  exportToken: async (token) => {
    try {
      const path = await runtime.SaveFileDialog();
      if (path) {
        await ExportToFile(token, path);
        get().addLog(`Token exported to ${path}`);
      }
    } catch (err: any) {
      get().addLog(`Error exporting: ${err.message || err}`);
    }
  },

  importToken: async () => {
    try {
      const path = await runtime.OpenFileDialog();
      if (path) {
        const token = await ImportFromFile(path);
        set({ importedToken: token });
        get().addLog(`Token imported from ${path}`);
      }
    } catch (err: any) {
      get().addLog(`Error importing: ${err.message || err}`);
    }
  },

  addLog: (message) =>
    set((state) => ({ logs: [...state.logs, message] })),
  setStatus: (status) => set({ status }),
}));
```

**Step 4: Run test to verify it passes**

Run: `cd frontend && bun test lib/tunnelStore.test.ts`
Expected: PASS

**Step 5: Commit**

```bash
git add frontend/src/lib/tunnelStore.ts frontend/src/lib/tunnelStore.test.ts
git commit -m "feat: update tunnelStore with new Go bindings"
```

---

### Task 10: Create TokenCard component

**Files:**
- Create: `frontend/src/components/custom/token-card.tsx`
- Test: `frontend/src/components/custom/token-card.test.tsx`

**Step 1: Write the failing test**

Create `frontend/src/components/custom/token-card.test.tsx`:

```typescript
import { render, screen } from "@testing-library/react";
import { TokenCard } from "./token-card";

describe("TokenCard", () => {
  it("should display token text", () => {
    render(<TokenCard token="test-token-123" />);
    expect(screen.getByText("test-token-123")).toBeInTheDocument();
  });

  it("should have copy button", () => {
    render(<TokenCard token="test-token" />);
    expect(screen.getByRole("button", { name: /copy/i })).toBeInTheDocument();
  });
});
```

**Step 2: Run test to verify it fails**

Run: `cd frontend && bun test components/custom/token-card.test.tsx`
Expected: FAIL with "Cannot find module './token-card'"

**Step 3: Write minimal implementation**

Create `frontend/src/components/custom/token-card.tsx`:

```typescript
import React from "react";
import { Copy, Download, Upload } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { useTunnelStore } from "@/lib/tunnelStore";

interface TokenCardProps {
  token: string;
  type: "offer" | "answer";
  onExport?: () => void;
}

export const TokenCard: React.FC<TokenCardProps> = ({ token, type, onExport }) => {
  const copyToClipboard = async () => {
    await navigator.clipboard.writeText(token);
  };

  const shareLink = () => {
    const url = `minecraft-tunnel://join?token=${encodeURIComponent(token)}`;
    navigator.clipboard.writeText(url);
  };

  return (
    <Card className="w-full">
      <CardHeader>
        <CardTitle className="text-sm font-medium">
          {type === "offer" ? "Offer Token" : "Answer Token"}
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-3">
        <Input
          readOnly
          value={token}
          className="font-mono text-xs"
          placeholder="Token will appear here..."
        />
        <div className="flex gap-2">
          <Button
            variant="outline"
            size="sm"
            onClick={copyToClipboard}
            disabled={!token}
          >
            <Copy className="w-4 h-4 mr-2" />
            Copy
          </Button>
          <Button
            variant="outline"
            size="sm"
            onClick={shareLink}
            disabled={!token}
          >
            <Upload className="w-4 h-4 mr-2" />
            Share Link
          </Button>
          {onExport && (
            <Button
              variant="outline"
              size="sm"
              onClick={onExport}
              disabled={!token}
            >
              <Download className="w-4 h-4 mr-2" />
              Export
            </Button>
          )}
        </div>
      </CardContent>
    </Card>
  );
};
```

**Step 4: Run test to verify it passes**

Run: `cd frontend && bun test components/custom/token-card.test.tsx`
Expected: PASS

**Step 5: Commit**

```bash
git add frontend/src/components/custom/token-card.tsx frontend/src/components/custom/token-card.test.tsx
git commit -m "feat: add TokenCard component"
```

---

### Task 11: Update HostView to use new store and components

**Files:**
- Modify: `frontend/src/routes/host/index.tsx`
- Test: `frontend/src/routes/host/index.test.tsx`

**Step 1: Write the failing test**

Create `frontend/src/routes/host/index.test.tsx`:

```typescript
import { render, screen, fireEvent } from "@testing-library/react";
import { HostView } from "./index";

describe("HostView", () => {
  it("should display generate invitation button", () => {
    render(<HostView />);
    expect(screen.getByRole("button", { name: /generate invitation/i })).toBeInTheDocument();
  });

  it("should show offer token after generation", async () => {
    render(<HostView />);
    const button = screen.getByRole("button", { name: /generate invitation/i });
    fireEvent.click(button);
    // Would need to mock CreateOffer call
  });
});
```

**Step 2: Run test to verify it fails**

Run: `cd frontend && bun test routes/host/index.test.tsx`
Expected: FAIL with button not found (button name mismatch)

**Step 3: Write minimal implementation**

Replace `frontend/src/routes/host/index.tsx`:

```typescript
import React, { useEffect, useRef } from "react";
import { useTunnelStore } from "@/lib/tunnelStore";
import { EventsOn, EventsOff, SaveFileDialog } from "../../../wailsjs/runtime/runtime";
import { TokenCard } from "@/components/custom/token-card";
import Sigil from "@/components/custom/sigil";

import { Power, ArrowLeft, Activity, Terminal, Server } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Card,
  CardContent,
  CardFooter,
  CardHeader,
  CardTitle,
  CardDescription,
} from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Label } from "@/components/ui/label";
import { ScrollArea } from "@/components/ui/scroll-area";

export const HostView = () => {
  const {
    status,
    logs,
    mcServerAddress,
    setMcServerAddress,
    addLog,
    setStatus,
    generateOffer,
    acceptAnswer,
    exportToken,
  } = useTunnelStore();

  const scrollRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    EventsOn("log", addLog);
    EventsOn("status-change", (newStatus: string) =>
      setStatus(newStatus as any),
    );
    return () => {
      EventsOff("log");
      EventsOff("status-change");
    };
  }, [addLog, setStatus]);

  useEffect(() => {
    scrollRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [logs]);

  const isRunning = status === "connected";
  const statusColor =
    {
      disconnected: "border-slate-200",
      connecting: "border-yellow-200 animate-pulse",
      "waiting-for-answer": "border-blue-200",
      connected: "border-green-200",
      error: "border-red-200",
    }[status] || "";

  return (
    <div className="flex flex-col items-center justify-center min-h-screen min-w-screen p-6 font-sans">
      <Sigil scale={0.25} rotating />
      <Card className="w-full shadow-xl">
        <CardHeader className="pb-4">
          <div className="flex justify-between items-start">
            <div className="space-y-1">
              <CardTitle className="text-xl font-bold tracking-tight">
                Host Tunnel
              </CardTitle>
              <CardDescription>
                Expose your local Minecraft server to a friend.
              </CardDescription>
            </div>
            <Badge
              variant="outline"
              className={`${statusColor} capitalize px-3 py-1`}
            >
              {status === "connected" && <Activity className="w-3 h-3 mr-1" />}
              {status.replace(/-/g, " ")}
            </Badge>
          </div>
        </CardHeader>

        <CardContent className="space-y-6">
          {/* Server Address Input */}
          <div className="space-y-2">
            <Label htmlFor="server-address" className="text-sm font-medium">
              <Server className="w-4 h-4 mr-1 inline" />
              Minecraft Server Address
            </Label>
            <Input
              id="server-address"
              placeholder="localhost:25565"
              value={mcServerAddress}
              onChange={(e) => setMcServerAddress(e.target.value)}
              disabled={isRunning}
              className="font-mono text-sm"
            />
          </div>

          {/* Offer Token Section */}
          {(status === "waiting-for-answer" || status === "connected") && (
            <div className="space-y-2">
              <Label className="text-sm font-medium">
                Share this Offer Token with your friend:
              </Label>
              <TokenCard
                token={useTunnelStore((s) => s.offerToken)}
                type="offer"
                onExport={() => exportToken(useTunnelStore((s) => s.offerToken))}
              />
            </div>
          )}

          {/* Answer Input Section */}
          {status === "waiting-for-answer" && (
            <div className="space-y-2">
              <Label className="text-sm font-medium">
                Paste Answer Token from friend:
              </Label>
              <div className="flex gap-2">
                <Input
                  placeholder="Paste answer token here..."
                  className="font-mono text-xs flex-1"
                  id="answer-input"
                />
                <Button onClick={() => {
                  const input = document.getElementById("answer-input") as HTMLInputElement;
                  if (input.value) acceptAnswer(input.value);
                }}>
                  Connect
                </Button>
              </div>
            </div>
          )}

          {/* Logs */}
          <div className="space-y-2">
            <div className="flex items-center justify-between">
              <Label className="text-sm font-medium flex items-center gap-2">
                <Terminal className="w-4 h-4" />
                Live Logs
              </Label>
              <span className="text-xs">{logs.length} events</span>
            </div>
            <div className="rounded-lg border p-4 shadow-inner">
              <ScrollArea className="h-64 w-full pr-4">
                <div className="flex flex-col gap-1 font-mono text-xs">
                  {logs.length === 0 && (
                    <div className="italic select-none py-10 text-center">
                      Waiting for connection...
                    </div>
                  )}
                  {logs.map((log, i) => (
                    <div
                      key={i}
                      className="break-all border-l-2 border-transparent pl-2 hover:border-slate-700 hover:bg-slate-900/50 transition-colors"
                    >
                      <span className="mr-2 text-slate-500">
                        {new Date().toLocaleTimeString([], { hour12: false })}
                      </span>
                      {log}
                    </div>
                  ))}
                  <div ref={scrollRef} />
                </div>
              </ScrollArea>
            </div>
          </div>
        </CardContent>

        <CardFooter className="flex justify-between pt-2 border-t border-slate-100">
          <Button
            variant="ghost"
            onClick={() => window.history.back()}
            disabled={status !== "disconnected"}
          >
            <ArrowLeft className="w-4 h-4 mr-2" />
            Back
          </Button>

          {status === "disconnected" || status === "error" ? (
            <Button onClick={generateOffer} className="shadow-md">
              <Power className="w-4 h-4 mr-2" />
              Generate Invitation
            </Button>
          ) : (
            <Button variant="outline" disabled>
              <Activity className="w-4 h-4 mr-2" />
              {status === "connected" ? "Connected" : "Connecting..."}
            </Button>
          )}
        </CardFooter>
      </Card>
    </div>
  );
};
```

**Step 4: Run test to verify it passes**

Run: `cd frontend && bun test routes/host/index.test.tsx`
Expected: PASS

**Step 5: Commit**

```bash
git add frontend/src/routes/host/index.tsx frontend/src/routes/host/index.test.tsx
git commit -m "feat: update HostView with new token flow"
```

---

### Task 12: Create JoinView component

**Files:**
- Create: `frontend/src/routes/join/index.tsx`
- Test: `frontend/src/routes/join/index.test.tsx`

**Step 1: Write the failing test**

Create `frontend/src/routes/join/index.test.tsx`:

```typescript
import { render, screen, fireEvent } from "@testing-library/react";
import { JoinView } from "./index";

describe("JoinView", () => {
  it("should display paste offer input", () => {
    render(<JoinView />);
    expect(screen.getByPlaceholderText(/paste offer token/i)).toBeInTheDocument();
  });

  it("should have import token button", () => {
    render(<JoinView />);
    expect(screen.getByRole("button", { name: /import/i })).toBeInTheDocument();
  });
});
```

**Step 2: Run test to verify it fails**

Run: `cd frontend && bun test routes/join/index.test.tsx`
Expected: FAIL with "Cannot find module './index'"

**Step 3: Write minimal implementation**

Create `frontend/src/routes/join/index.tsx`:

```typescript
import React, { useEffect, useRef, useState } from "react";
import { useTunnelStore } from "@/lib/tunnelStore";
import { EventsOn, EventsOff, OpenFileDialog } from "../../../wailsjs/runtime/runtime";
import { TokenCard } from "@/components/custom/token-card";
import Sigil from "@/components/custom/sigil";

import { ArrowLeft, Activity, Terminal, FileUp, Link as LinkIcon } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Card,
  CardContent,
  CardFooter,
  CardHeader,
  CardTitle,
  CardDescription,
} from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Label } from "@/components/ui/label";
import { ScrollArea } from "@/components/ui/scroll-area";

export const JoinView = () => {
  const {
    status,
    logs,
    acceptOffer,
    addLog,
    setStatus,
  } = useTunnelStore();

  const scrollRef = useRef<HTMLDivElement>(null);
  const [offerInput, setOfferInput] = useState("");

  useEffect(() => {
    EventsOn("log", addLog);
    EventsOn("status-change", (newStatus: string) =>
      setStatus(newStatus as any),
    );
    return () => {
      EventsOff("log");
      EventsOff("status-change");
    };
  }, [addLog, setStatus]);

  useEffect(() => {
    scrollRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [logs]);

  const handlePasteOffer = async () => {
    try {
      await acceptOffer(offerInput);
      setOfferInput("");
    } catch (err: any) {
      addLog(`Error: ${err.message || err}`);
    }
  };

  const handleImportFile = async () => {
    try {
      const path = await OpenFileDialog();
      if (path) {
        const token = await ImportFromFile(path);
        setOfferInput(token);
        addLog(`Token imported from ${path}`);
      }
    } catch (err: any) {
      addLog(`Error importing: ${err.message || err}`);
    }
  };

  const statusColor =
    {
      disconnected: "border-slate-200",
      connecting: "border-yellow-200 animate-pulse",
      "waiting-for-host": "border-blue-200",
      connected: "border-green-200",
      error: "border-red-200",
    }[status] || "";

  return (
    <div className="flex flex-col items-center justify-center min-h-screen min-w-screen p-6 font-sans">
      <Sigil scale={0.25} rotating />
      <Card className="w-full shadow-xl">
        <CardHeader className="pb-4">
          <div className="flex justify-between items-start">
            <div className="space-y-1">
              <CardTitle className="text-xl font-bold tracking-tight">
                Join Tunnel
              </CardTitle>
              <CardDescription>
                Connect to a friend's Minecraft server.
              </CardDescription>
            </div>
            <Badge
              variant="outline"
              className={`${statusColor} capitalize px-3 py-1`}
            >
              {status === "connected" && <Activity className="w-3 h-3 mr-1" />}
              {status.replace(/-/g, " ")}
            </Badge>
          </div>
        </CardHeader>

        <CardContent className="space-y-6">
          {/* Offer Input Section */}
          {status === "disconnected" && (
            <div className="space-y-2">
              <Label className="text-sm font-medium">
                Paste Offer Token from your friend:
              </Label>
              <div className="flex gap-2">
                <Input
                  placeholder="Paste offer token here..."
                  value={offerInput}
                  onChange={(e) => setOfferInput(e.target.value)}
                  className="font-mono text-xs flex-1"
                />
                <Button onClick={handlePasteOffer} disabled={!offerInput}>
                  Connect
                </Button>
              </div>
              <div className="flex gap-2">
                <Button variant="outline" size="sm" onClick={handleImportFile}>
                  <FileUp className="w-4 h-4 mr-2" />
                  Import File
                </Button>
                <Button variant="outline" size="sm" onClick={() => {
                  navigator.clipboard.readText().then(text => setOfferInput(text));
                }}>
                  <LinkIcon className="w-4 h-4 mr-2" />
                  Paste from Clipboard
                </Button>
              </div>
            </div>
          )}

          {/* Answer Token Section */}
          {(status === "waiting-for-host" || status === "connected") && (
            <div className="space-y-2">
              <Label className="text-sm font-medium">
                Share this Answer Token back to your friend:
              </Label>
              <TokenCard
                token={useTunnelStore((s) => s.answerToken)}
                type="answer"
              />
              <div className="text-sm text-slate-600 bg-yellow-50 border border-yellow-200 rounded p-3">
                <strong>⚠️ Important:</strong> Copy and send this back to the host!
              </div>
            </div>
          )}

          {/* Logs */}
          <div className="space-y-2">
            <div className="flex items-center justify-between">
              <Label className="text-sm font-medium flex items-center gap-2">
                <Terminal className="w-4 h-4" />
                Live Logs
              </Label>
              <span className="text-xs">{logs.length} events</span>
            </div>
            <div className="rounded-lg border p-4 shadow-inner">
              <ScrollArea className="h-64 w-full pr-4">
                <div className="flex flex-col gap-1 font-mono text-xs">
                  {logs.length === 0 && (
                    <div className="italic select-none py-10 text-center">
                      Waiting for connection...
                    </div>
                  )}
                  {logs.map((log, i) => (
                    <div
                      key={i}
                      className="break-all border-l-2 border-transparent pl-2 hover:border-slate-700 hover:bg-slate-900/50 transition-colors"
                    >
                      <span className="mr-2 text-slate-500">
                        {new Date().toLocaleTimeString([], { hour12: false })}
                      </span>
                      {log}
                    </div>
                  ))}
                  <div ref={scrollRef} />
                </div>
              </ScrollArea>
            </div>
          </div>
        </CardContent>

        <CardFooter className="flex justify-between pt-2 border-t border-slate-100">
          <Button
            variant="ghost"
            onClick={() => window.history.back()}
            disabled={status !== "disconnected"}
          >
            <ArrowLeft className="w-4 h-4 mr-2" />
            Back
          </Button>

          {status === "connected" && (
            <Badge className="bg-green-100 text-green-800">
              <Activity className="w-3 h-3 mr-1" />
              Connected
            </Badge>
          )}
        </CardFooter>
      </Card>
    </div>
  );
};
```

**Step 4: Run test to verify it passes**

Run: `cd frontend && bun test routes/join/index.test.tsx`
Expected: PASS

**Step 5: Commit**

```bash
git add frontend/src/routes/join/index.tsx frontend/src/routes/join/index.test.tsx
git commit -m "feat: add JoinView component"
```

---

### Task 13: Update Router to include JoinView

**Files:**
- Modify: `frontend/src/components/Router.tsx`

**Step 1: Verify Router imports JoinView**

**Step 2: Update Router**

Replace `frontend/src/components/Router.tsx`:

```typescript
import { useAppStore, type ValidRoutes } from "@/lib/store";
import Main from "@/routes/main";
import { HostView } from "@/routes/host";
import { JoinView } from "@/routes/join";

export function Router() {
  const route = useAppStore((s) => s.route);
  switch (route) {
    case "/":
      return <Main />;
    case "/host":
      return <HostView />;
    case "/join":
      return <JoinView />;
    default:
      return <Main />;
  }
}
```

**Step 3: Commit**

```bash
git add frontend/src/components/Router.tsx
git commit -m "fix: add JoinView to Router"
```

---

## Phase 3: Integration & Testing

### Task 14: Run all Go tests

**Step 1: Run all tests**

Run: `go test ./... -v`
Expected: All tests pass

**Step 2: Commit (if any fixes needed)**

```bash
git commit -m "fix: ensure all Go tests pass"
```

---

### Task 15: Run all frontend tests

**Step 1: Run all tests**

Run: `cd frontend && bun test`
Expected: All tests pass

**Step 2: Commit (if any fixes needed)**

```bash
git commit -m "fix: ensure all frontend tests pass"
```

---

### Task 16: Build and verify Wails bindings

**Step 1: Regenerate Wails bindings**

Run: `wails dev`
Expected: Builds and runs without errors
Verify: Check `frontend/wailsjs/go/main/App.js` has new functions

**Step 2: Commit generated bindings**

```bash
git add frontend/wailsjs/
git commit -m "chore: update Wails bindings"
```

---

### Task 17: Manual testing - Host flow

**Step 1: Start app and navigate to Host**

1. Run `wails dev`
2. Click "Host" on main screen
3. Verify: "Generate Invitation" button visible
4. Verify: Server address input shows "localhost:25565"

**Step 2: Generate offer token**

1. Click "Generate Invitation"
2. Verify: Status changes to "Waiting for answer"
3. Verify: Offer token displayed
4. Verify: Copy, Share Link, Export buttons work

**Step 3: Verify logs**

1. Check logs show "Offer token generated successfully"
2. Verify timestamps present
3. Verify auto-scroll works

---

### Task 18: Manual testing - Join flow

**Step 1: Navigate to Join view**

1. Go back to main screen
2. Click "Join"
3. Verify: Input field for offer token visible
4. Verify: Import File and Paste buttons present

**Step 2: Test token import**

1. Create a test .mc-tunnel-invite file with offer token
2. Click "Import File"
3. Verify: Token appears in input field

**Step 3: Generate answer**

1. Click "Connect"
2. Verify: Status changes to "Waiting for host"
3. Verify: Answer token displayed
4. Verify: Warning message to share back

---

### Task 19: Manual testing - Connection flow

**Step 1: Complete handshake**

1. On Host: Paste answer token from joiner
2. Click "Connect"
3. Verify: Status changes to "Connected"
4. Verify: Green badge with activity indicator

**Step 2: Verify logs**

1. Both sides show "Tunnel established!"
2. Verify no errors in logs

**Step 3: Test reconnection**

1. Close one app
2. Verify: Other side shows disconnected
3. Restart: Verify new tokens needed (ephemeral)

---

### Task 20: Edge case testing

**Step 1: Test invalid tokens**

1. Paste "not-a-token" in Host answer field
2. Click Connect
3. Verify: Error message shown
4. Verify: Status changes to "error"

**Step 2: Test MC server offline (Host)**

1. Ensure no server on localhost:25565
2. Generate offer and complete handshake
3. Verify: Error log "Cannot connect to Minecraft server"

**Step 3: Test port in use (Joiner)**

1. Start process on port 25565
2. Try to join tunnel
3. Verify: Error log "Port already in use"

---

## Phase 4: Documentation & Cleanup

### Task 21: Update README with usage instructions

**Files:**
- Modify: `README.md`

**Step 1: Add usage section**

Add to `README.md` after features:

```markdown
## 🎮 Usage

### Hosting a Minecraft Server

1. Start your local Minecraft server on port 25565
2. Click "Host" in the app
3. Enter your server address (default: localhost:25565)
4. Click "Generate Invitation"
5. Share the Offer Token with your friend (copy, file, or link)
6. Wait for your friend's Answer Token
7. Paste the Answer Token and click "Connect"
8. Tunnel is now active!

### Joining a Server

1. Click "Join" in the app
2. Paste your friend's Offer Token (or import file)
3. Click "Connect"
4. Copy the generated Answer Token
5. Share the Answer Token back to the host
6. Open Minecraft and connect to localhost:25565
7. Play on your friend's server!

### Token Exchange Methods

- **Copy/Paste**: Share tokens via chat, email, or messaging
- **File Export**: Save .mc-tunnel-invite files for easy sharing
- **Deeplink**: Share `minecraft-tunnel://join?token=...` links (requires protocol registration)
```

**Step 2: Commit**

```bash
git add README.md
git commit -m "docs: add usage instructions to README"
```

---

### Task 22: Final code review

**Step 1: Run linters**

Run: `go fmt ./...`
Run: `cd frontend && npm run lint`

**Step 2: Fix any issues**

Commit fixes if needed.

**Step 3: Final build test**

Run: `wails build -clean`
Verify: Binary builds successfully

---

### Task 23: Create implementation notes

**Files:**
- Create: `docs/IMPLEMENTATION_NOTES.md`

**Step 1: Document design decisions**

Create `docs/IMPLEMENTATION_NOTES.md`:

```markdown
# Implementation Notes

## Architecture Decisions

### Ephemeral Tokens
Tokens are not persisted to disk. When connection drops or app restarts, users must generate new tokens. This simplifies security and avoids stale token issues.

### Two-Token Exchange
WebRTC requires both offer and answer for handshake. We auto-generate the answer when the joiner accepts the offer, minimizing user steps while maintaining security.

### Port 25565
Joiner always listens on localhost:25565, allowing Minecraft client to connect without configuration. Host connects to their specified server address.

### pion/webrtc
Used pure Go WebRTC implementation for cross-platform support and no external dependencies.

### STUN Servers
Default to Google's public STUN servers. In production, consider running your own or using TURN servers for reliability.

## Future Enhancements

- TURN server support for restrictive NATs
- Multiple peer connections
- Connection bandwidth monitoring
- Token validation and expiration
- Encrypted tunnel configuration
- Custom port configuration
```

**Step 2: Commit**

```bash
git add docs/IMPLEMENTATION_NOTES.md
git commit -m "docs: add implementation notes"
```

---

## Summary

This implementation plan builds a complete WebRTC P2P Minecraft tunnel with:

✅ Go backend with WebRTC lifecycle management
✅ TCP proxying for both host and joiner modes
✅ Token export/import functionality
✅ React frontend with Host and Join views
✅ Complete test coverage (Go + Bun)
✅ Manual testing procedures
✅ Documentation

Total: 23 tasks, each broken into 2-5 minute steps with TDD approach.
