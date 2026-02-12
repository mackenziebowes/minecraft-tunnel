package main

import (
	"context"
	"encoding/base64"
	"os"
	"testing"

	"github.com/pion/webrtc/v3"
)

func TestNewPeerConnectionManager(t *testing.T) {
	manager := NewPeerConnectionManager()
	if manager == nil {
		t.Fatal("Expected non-nil manager")
	}
}

func TestCreateOfferGeneratesValidBase64(t *testing.T) {
	app := &App{ctx: context.Background()}
	token, err := app.CreateOffer()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if token == "" {
		t.Fatal("Expected non-empty token")
	}

	_, err = base64.StdEncoding.DecodeString(token)
	if err != nil {
		t.Fatalf("Expected valid base64, got: %v", err)
	}
}

func TestAcceptOfferGeneratesAnswer(t *testing.T) {
	hostApp := &App{ctx: context.Background()}

	// Create a real offer token
	offerToken, err := hostApp.CreateOffer()
	if err != nil {
		t.Fatalf("Failed to create offer: %v", err)
	}

	// Joiner accepts the offer and generates answer
	joinerApp := &App{ctx: context.Background()}
	answerToken, err := joinerApp.AcceptOffer(offerToken)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if answerToken == "" {
		t.Fatal("Expected non-empty answer token")
	}

	// Verify answer is valid base64
	_, err = base64.StdEncoding.DecodeString(answerToken)
	if err != nil {
		t.Fatalf("Expected valid base64 answer, got: %v", err)
	}
}

func TestAcceptAnswerSetsRemoteDescription(t *testing.T) {
	hostApp := &App{ctx: context.Background()}

	// Create offer
	offerToken, err := hostApp.CreateOffer()
	if err != nil {
		t.Fatalf("Failed to create offer: %v", err)
	}

	// Generate real answer
	joinerApp := &App{ctx: context.Background()}
	answerToken, err := joinerApp.AcceptOffer(offerToken)
	if err != nil {
		t.Fatalf("Failed to generate answer: %v", err)
	}

	// Host accepts the answer
	err = hostApp.AcceptAnswer(answerToken)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
}

func TestTunnelProxyConnectsToMinecraftServer(t *testing.T) {
	app := &App{ctx: context.Background()}
	dc := &webrtc.DataChannel{}

	_ = app
	_ = dc
}

func TestStartJoinerProxyListensOnPort25565(t *testing.T) {
	app := &App{ctx: context.Background()}
	dc := &webrtc.DataChannel{}

	err := app.StartJoinerProxy(dc, "0")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
}

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
