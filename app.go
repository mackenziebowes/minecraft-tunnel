package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"sync"
	"time"

	"github.com/pion/webrtc/v3"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"runtime/debug"
)

type contextKey string

const testModeKey contextKey = "testMode"

// We exchange these JSON blobs to connect
type Signal struct {
	SDP string `json:"sdp"`
}

type App struct {
	ctx            context.Context
	cancel         context.CancelFunc
	peerConnection *webrtc.PeerConnection
	joinerConns    map[net.Conn]struct{}
	joinerConnMu   sync.Mutex
	listener       net.Listener
}

type PeerConnectionManager struct {
	peerConnection *webrtc.PeerConnection
	dataChannel    *webrtc.DataChannel
}

func NewPeerConnectionManager() *PeerConnectionManager {
	return &PeerConnectionManager{}
}

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

func NewApp() *App {
	return &App{}
}

func (a *App) startup(ctx context.Context) {
	a.ctx, a.cancel = context.WithCancel(ctx)
}

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

func (a *App) CreateOffer() (string, error) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintf(os.Stderr, "[PANIC] CreateOffer recovered: %v\n", r)
			debug.PrintStack()
		}
	}()

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

	a.peerConnection = peerConnection

	dataChannel, err := peerConnection.CreateDataChannel("minecraft", nil)
	if err != nil {
		return "", err
	}

	dataChannel.OnOpen(func() {
		a.safeEventEmit("status-change", "connected")
		a.safeEventEmit("log", "P2P Tunnel Established!")
		go a.pumpMinecraftToChannel(dataChannel)
	})

	dataChannel.OnClose(func() {
		a.safeEventEmit("status-change", "disconnected")
		a.safeEventEmit("log", "DataChannel closed")
	})

	dataChannel.OnMessage(func(msg webrtc.DataChannelMessage) {
	})

	peerConnection.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		switch state {
		case webrtc.PeerConnectionStateDisconnected:
			a.safeEventEmit("status-change", "disconnected")
			a.safeEventEmit("log", "Peer disconnected")
		case webrtc.PeerConnectionStateFailed:
			a.safeEventEmit("status-change", "error")
			a.safeEventEmit("log", "Connection failed")
		}
	})

	offer, err := peerConnection.CreateOffer(nil)
	if err != nil {
		return "", err
	}

	if err = peerConnection.SetLocalDescription(offer); err != nil {
		return "", err
	}

	gatheringDone := webrtc.GatheringCompletePromise(peerConnection)
	select {
	case <-gatheringDone:
	case <-time.After(TimeoutWebRTCICE):
		cleanupNeeded = false
		peerConnection.Close()
		return "", fmt.Errorf("ICE gathering timeout: failed to gather candidates after %v", TimeoutWebRTCICE)
	}

	offerJson, err := json.Marshal(peerConnection.LocalDescription())
	if err != nil {
		return "", fmt.Errorf("failed to marshal offer: %w", err)
	}

	cleanupNeeded = false
	return base64.StdEncoding.EncodeToString(offerJson), nil
}

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

	a.peerConnection = peerConnection

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

	peerConnection.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		switch state {
		case webrtc.PeerConnectionStateDisconnected:
			a.safeEventEmit("status-change", "disconnected")
			a.safeEventEmit("log", "Peer disconnected")
		case webrtc.PeerConnectionStateFailed:
			a.safeEventEmit("status-change", "error")
			a.safeEventEmit("log", "Connection failed")
		}
	})

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

// Helper: Connects DataChannel <-> Local Minecraft
func (a *App) pumpMinecraftToChannel(dc *webrtc.DataChannel) {
	// Connect to local Minecraft Server
	mcConn, err := DialTimeout("tcp", "localhost:25565", TimeoutTCPConnect)
	if err != nil {
		a.safeEventEmit("status-change", "error")
		a.safeEventEmit("log", fmt.Sprintf("Error connecting to Minecraft server: %v", err))
		return
	}
	defer mcConn.Close()

	// 1. Minecraft -> WebRTC Tunnel
	go func() {
		buf := make([]byte, 1500)
		for {
			n, err := mcConn.Read(buf)
			if err != nil {
				return
			}
			// Send raw bytes over WebRTC
			dc.Send(buf[:n])
		}
	}()

	// 2. WebRTC Tunnel -> Minecraft
	dc.OnMessage(func(msg webrtc.DataChannelMessage) {
		mcConn.Write(msg.Data)
	})

	// Keep blocking until closed
	select {}
}

func (a *App) StartHostProxy(dc *webrtc.DataChannel, targetAddress string) error {
	mcConn, err := DialTimeout("tcp", targetAddress, TimeoutTCPConnect)
	if err != nil {
		a.safeEventEmit("log", fmt.Sprintf("Error connecting to Minecraft server: %v", err))
		return fmt.Errorf("cannot connect to Minecraft server: %w", err)
	}
	defer mcConn.Close()

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

func (a *App) StartJoinerProxy(dc *webrtc.DataChannel, port string) error {
	a.joinerConns = make(map[net.Conn]struct{})

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

	a.safeEventEmit("log", fmt.Sprintf("Listening on port %s for Minecraft client", port))

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

func (a *App) ExportToFile(token string, filepath string) error {
	resultChan := make(chan error, 1)
	go func() {
		resultChan <- os.WriteFile(filepath, []byte(token), 0644)
	}()

	select {
	case err := <-resultChan:
		return err
	case <-time.After(TimeoutFileIO):
		return fmt.Errorf("file write timeout: failed to write to %s after %v", filepath, TimeoutFileIO)
	}
}

func (a *App) ImportFromFile(filepath string) (string, error) {
	resultChan := make(chan struct {
		content string
		err     error
	}, 1)

	go func() {
		content, err := os.ReadFile(filepath)
		resultChan <- struct {
			content string
			err     error
		}{string(content), err}
	}()

	select {
	case result := <-resultChan:
		if result.err != nil {
			return "", fmt.Errorf("cannot read file: %w", result.err)
		}
		return result.content, nil
	case <-time.After(TimeoutFileIO):
		return "", fmt.Errorf("file read timeout: failed to read from %s after %v", filepath, TimeoutFileIO)
	}
}
