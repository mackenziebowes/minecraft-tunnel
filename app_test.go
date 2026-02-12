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
