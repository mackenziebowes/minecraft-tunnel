package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/pion/webrtc/v3"
	"github.com/wailsapp/wails/v2/pkg/runtime"
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
}

type PeerConnectionManager struct {
	peerConnection *webrtc.PeerConnection
	dataChannel    *webrtc.DataChannel
}

func NewPeerConnectionManager() *PeerConnectionManager {
	return &PeerConnectionManager{}
}

func (a *App) safeEventEmit(event string, data ...interface{}) {
	if a.ctx == nil {
		return
	}
	if mode, ok := a.ctx.Value(testModeKey).(bool); ok && mode {
		return
	}
	defer func() {
		if r := recover(); r != nil {
		}
	}()
	runtime.EventsEmit(a.ctx, event, data...)
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
	if a.peerConnection != nil {
		a.peerConnection.Close()
		a.peerConnection = nil
	}
}

// 1. HOST: Generates the Offer Token
func (a *App) CreateOffer() (string, error) {
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

// 2. HOST: Accepts the Answer Token from the Friend
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

// 3. JOINER: Accepts the Offer Token and generates Answer Token
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

	var cleanupNeeded = true
	defer func() {
		if cleanupNeeded && peerConnection != nil {
			peerConnection.Close()
		}
	}()

	a.peerConnection = peerConnection

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

// Helper: Connects DataChannel <-> Local Minecraft
func (a *App) pumpMinecraftToChannel(dc *webrtc.DataChannel) {
	// Connect to local Minecraft Server
	mcConn, err := DialTimeout("tcp", "localhost:25565", TimeoutTCPConnect)
	if err != nil {
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
	listener, err := ListenTimeout("tcp", ":"+port, TimeoutNetwork)
	if err != nil {
		return fmt.Errorf("failed to listen on port %s: %w", port, err)
	}

	go func() {
		defer func() {
			if r := recover(); r != nil {
			}
		}()
		a.safeEventEmit("log", fmt.Sprintf("Listening on port %s for Minecraft client", port))
	}()

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
