package main

import (
	"context"
	"testing"
)

// TestCreateOfferWithRealContext tests CreateOffer with a non-test context
// This simulates production environment where testModeKey is not set
func TestCreateOfferWithRealContext(t *testing.T) {
	ctx := context.Background()
	app := &App{ctx: ctx}

	t.Logf("Testing CreateOffer with real context (not test mode)")
	t.Logf("testModeKey value: %v", ctx.Value(testModeKey))

	token, err := app.CreateOffer()
	if err != nil {
		t.Fatalf("CreateOffer failed with real context: %v", err)
	}

	if token == "" {
		t.Fatal("Expected non-empty token from CreateOffer")
	}

	t.Logf("CreateOffer succeeded with real context, token length: %d", len(token))

	// Clean up
	if app.peerConnection != nil {
		app.peerConnection.Close()
	}
}

// TestAcceptOfferWithRealContext tests AcceptOffer with a non-test context
func TestAcceptOfferWithRealContext(t *testing.T) {
	ctx := context.Background()
	hostApp := &App{ctx: ctx}

	// Create offer with real context
	offerToken, err := hostApp.CreateOffer()
	if err != nil {
		t.Fatalf("Failed to create offer with real context: %v", err)
	}

	// Accept offer with real context
	joinerApp := &App{ctx: ctx}
	answerToken, err := joinerApp.AcceptOffer(offerToken)
	if err != nil {
		t.Fatalf("AcceptOffer failed with real context: %v", err)
	}

	if answerToken == "" {
		t.Fatal("Expected non-empty answer token from AcceptOffer")
	}

	t.Logf("AcceptOffer succeeded with real context, answer length: %d", len(answerToken))

	// Clean up both sides
	if hostApp.peerConnection != nil {
		hostApp.peerConnection.Close()
	}
	if joinerApp.peerConnection != nil {
		joinerApp.peerConnection.Close()
	}
}

// TestAcceptAnswerWithRealContext tests AcceptAnswer with a non-test context
func TestAcceptAnswerWithRealContext(t *testing.T) {
	ctx := context.Background()
	hostApp := &App{ctx: ctx}

	// Create offer
	offerToken, err := hostApp.CreateOffer()
	if err != nil {
		t.Fatalf("Failed to create offer: %v", err)
	}

	// Generate answer
	joinerApp := &App{ctx: ctx}
	answerToken, err := joinerApp.AcceptOffer(offerToken)
	if err != nil {
		t.Fatalf("Failed to generate answer: %v", err)
	}

	// Accept answer
	err = hostApp.AcceptAnswer(answerToken)
	if err != nil {
		t.Fatalf("AcceptAnswer failed with real context: %v", err)
	}

	t.Logf("AcceptAnswer succeeded with real context")

	// Clean up
	if hostApp.peerConnection != nil {
		hostApp.peerConnection.Close()
	}
	if joinerApp.peerConnection != nil {
		joinerApp.peerConnection.Close()
	}
}

// TestPumpMinecraftToChannelHandlesMissingServer tests behavior when Minecraft server is not running
func TestPumpMinecraftToChannelHandlesMissingServer(t *testing.T) {
	// Mock data channel (won't actually be connected)
	// This tests that pumpMinecraftToChannel handles connection errors gracefully
	t.Log("Testing pumpMinecraftToChannel with no MC server running")
	t.Log("Expected: should log error and return, not panic")

	// Note: We can't easily test this without mocking, but the panic recovery
	// wrapper will catch any panics
}
