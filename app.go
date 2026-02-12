package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"

	"github.com/pion/webrtc/v3"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// We exchange these JSON blobs to connect
type Signal struct {
	SDP string `json:"sdp"`
}

type App struct {
	ctx            context.Context
	peerConnection *webrtc.PeerConnection
}

type PeerConnectionManager struct {
	peerConnection *webrtc.PeerConnection
	dataChannel    *webrtc.DataChannel
}

func NewPeerConnectionManager() *PeerConnectionManager {
	return &PeerConnectionManager{}
}

func NewApp() *App {
	return &App{}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
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
	a.peerConnection = peerConnection

	// Create the Data Channel (This is our "Tunnel Cable")
	dataChannel, err := peerConnection.CreateDataChannel("minecraft", nil)
	if err != nil {
		return "", err
	}

	// HANDLE OPEN: When the tunnel connects, start forwarding Minecraft
	dataChannel.OnOpen(func() {
		runtime.EventsEmit(a.ctx, "status-change", "connected")
		runtime.EventsEmit(a.ctx, "log", "P2P Tunnel Established! 🚀")

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
	<-webrtc.GatheringCompletePromise(peerConnection)

	// Encode the offer to base64 so it's easy to copy-paste
	offerJson, _ := json.Marshal(peerConnection.LocalDescription())
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

// Helper: Connects DataChannel <-> Local Minecraft
func (a *App) pumpMinecraftToChannel(dc *webrtc.DataChannel) {
	// Connect to local Minecraft Server
	mcConn, err := net.Dial("tcp", "localhost:25565")
	if err != nil {
		runtime.EventsEmit(a.ctx, "log", "Error: Minecraft Server not running on 25565!")
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
	mcConn, err := net.Dial("tcp", targetAddress)
	if err != nil {
		runtime.EventsEmit(a.ctx, "log", fmt.Sprintf("Error connecting to Minecraft server: %v", err))
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
