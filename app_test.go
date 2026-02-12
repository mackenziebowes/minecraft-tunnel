package main

import (
	"context"
	"encoding/base64"
	"testing"
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
