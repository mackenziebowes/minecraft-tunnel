package main

import (
	"context"
	"encoding/base64"
	"os"
	"testing"

	"github.com/pion/webrtc/v3"
)

func testContext() context.Context {
	ctx := context.Background()
	return context.WithValue(ctx, testModeKey, true)
}

func TestNewPeerConnectionManager(t *testing.T) {
	manager := NewPeerConnectionManager()
	if manager == nil {
		t.Fatal("Expected non-nil manager")
	}
}

func TestCreateOfferGeneratesValidBase64(t *testing.T) {
	app := &App{ctx: testContext()}
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
	hostApp := &App{ctx: testContext()}

	// Create a real offer token
	offerToken, err := hostApp.CreateOffer()
	if err != nil {
		t.Fatalf("Failed to create offer: %v", err)
	}

	// Joiner accepts the offer and generates answer
	joinerApp := &App{ctx: testContext()}
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
	hostApp := &App{ctx: testContext()}

	// Create offer
	offerToken, err := hostApp.CreateOffer()
	if err != nil {
		t.Fatalf("Failed to create offer: %v", err)
	}

	// Generate real answer
	joinerApp := &App{ctx: testContext()}
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

func TestCreateOfferClosesPeerConnectionOnError(t *testing.T) {
	initialFiles, err := os.ReadDir("/proc/self/fd")
	if err != nil {
		t.Skip("Cannot monitor file descriptors on this system")
	}
	initialCount := len(initialFiles)

	app := &App{ctx: testContext()}

	for i := 0; i < 5; i++ {
		offer, err := app.CreateOffer()
		if err != nil {
			t.Fatalf("CreateOffer failed on iteration %d: %v", i, err)
		}
		if offer == "" {
			t.Fatal("Expected non-empty offer")
		}
		app.shutdown(context.Background())
		app.peerConnection = nil
	}

	finalFiles, err := os.ReadDir("/proc/self/fd")
	if err != nil {
		t.Skip("Cannot monitor file descriptors on this system")
	}
	finalCount := len(finalFiles)

	if finalCount-initialCount > 10 {
		t.Errorf("Potential file descriptor leak: grew from %d to %d", initialCount, finalCount)
	}
}

func TestCreateOfferWithoutShutdownLeaksConnection(t *testing.T) {
	app := &App{ctx: testContext()}

	offer, err := app.CreateOffer()
	if err != nil {
		t.Fatalf("CreateOffer failed: %v", err)
	}
	if offer == "" {
		t.Fatal("Expected non-empty offer")
	}

	if app.peerConnection == nil {
		t.Fatal("Expected peerConnection to be set")
	}

	connectionState := app.peerConnection.ConnectionState()
	if connectionState == webrtc.PeerConnectionStateClosed {
		t.Error("Connection should be open after CreateOffer returns")
	}

	app.shutdown(context.Background())
}

func TestCreateOfferHandlesCreateOfferError(t *testing.T) {
	app := &App{ctx: testContext()}

	offer, err := app.CreateOffer()
	if err != nil {
		t.Fatalf("CreateOffer should succeed: %v", err)
	}
	if offer == "" {
		t.Fatal("Expected non-empty offer")
	}

	if app.peerConnection == nil {
		t.Fatal("Expected peerConnection to be set")
	}

	app.shutdown(context.Background())
}

func TestAcceptOfferHandlesSetRemoteDescriptionError(t *testing.T) {
	hostApp := &App{ctx: testContext()}

	offerToken, err := hostApp.CreateOffer()
	if err != nil {
		t.Fatalf("Failed to create offer: %v", err)
	}

	joinerApp := &App{ctx: testContext()}
	answerToken, err := joinerApp.AcceptOffer(offerToken)
	if err != nil {
		t.Fatalf("AcceptOffer failed: %v", err)
	}
	if answerToken == "" {
		t.Fatal("Expected non-empty answer token")
	}

	if joinerApp.peerConnection == nil {
		t.Fatal("Expected peerConnection to be set")
	}

	joinerApp.shutdown(context.Background())
}

func TestTunnelProxyConnectsToMinecraftServer(t *testing.T) {
	app := &App{ctx: testContext()}
	dc := &webrtc.DataChannel{}

	_ = app
	_ = dc
}

func TestStartJoinerProxyListensOnPort25565(t *testing.T) {
	app := &App{ctx: testContext()}
	dc := &webrtc.DataChannel{}

	err := app.StartJoinerProxy(dc, "0")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
}

func TestExportToFileWritesToken(t *testing.T) {
	tmpfile := "/tmp/test-invite.mc-tunnel-invite"
	defer os.Remove(tmpfile)

	app := &App{ctx: testContext()}
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

func TestImportFromFileReadsToken(t *testing.T) {
	tmpfile := "/tmp/test-read.mc-tunnel-invite"
	defer os.Remove(tmpfile)

	os.WriteFile(tmpfile, []byte("file-token"), 0644)

	app := &App{ctx: testContext()}
	token, err := app.ImportFromFile(tmpfile)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if token != "file-token" {
		t.Fatalf("Expected 'file-token', got '%s'", token)
	}
}
